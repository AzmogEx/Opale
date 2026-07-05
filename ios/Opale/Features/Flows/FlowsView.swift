import SwiftUI

/// Onglet Flux — le quotidien (EF-020→022) : mouvements du mois, recherche,
/// résumé revenus/dépenses, correction de catégorie apprenante, import CSV.
struct FlowsView: View {
    @Environment(SessionStore.self) private var session

    /// Premier jour du mois affiché.
    @State private var month = Calendar.current.dateInterval(of: .month, for: .now)!.start
    @State private var summary: MonthSummary?
    @State private var transactions: [Transaction] = []
    @State private var categories: [Category] = []
    @State private var searchText = ""
    @State private var errorMessage: String?

    @State private var editing: Transaction?
    @State private var showManualForm = false
    @State private var showImport = false

    /// Sous-vues de l'onglet Flux (EF-020 / EF-028 / EF-025-027).
    enum Segment: String, CaseIterable, Identifiable {
        case movements = "Mouvements"
        case envelopes = "Enveloppes"
        case upcoming = "À venir"
        var id: String { rawValue }
    }

    @State private var segment: Segment = .movements

    private var monthKey: String {
        month.formatted(.iso8601.year().month())
    }

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                Picker("Vue", selection: $segment) {
                    ForEach(Segment.allCases) { s in
                        Text(s.rawValue).tag(s)
                    }
                }
                .pickerStyle(.segmented)
                .padding(.horizontal)
                .padding(.bottom, 4)

                switch segment {
                case .movements: movementsList
                case .envelopes: EnvelopesView()
                case .upcoming: UpcomingView()
                }
            }
            .navigationTitle("Flux")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Menu {
                        Button {
                            showManualForm = true
                        } label: {
                            Label("Nouvelle transaction", systemImage: "plus.circle")
                        }
                        Button {
                            showImport = true
                        } label: {
                            Label("Importer un relevé (CSV)", systemImage: "square.and.arrow.down")
                        }
                    } label: {
                        Image(systemName: "plus")
                    }
                }
            }
            .task(id: monthKey + "|" + searchText) { await load() }
            .refreshable { await load() }
            .sheet(item: $editing) { tx in
                TransactionEditSheet(transaction: tx, categories: categories) {
                    Task { await load() }
                }
                .presentationDetents([.medium, .large])
            }
            .sheet(isPresented: $showManualForm) {
                ManualTransactionSheet(categories: categories) {
                    Task { await load() }
                }
            }
            .sheet(isPresented: $showImport) {
                ImportCSVSheet {
                    Task { await load() }
                }
            }
        }
    }

    // MARK: - Segment Mouvements (EF-020)

    private var movementsList: some View {
        List {
            monthHeader
            if let summary {
                summarySection(summary)
            }
            transactionSections
            if let errorMessage {
                Text(errorMessage).foregroundStyle(OpaleTheme.loss)
            }
        }
        .listStyle(.insetGrouped)
        .searchable(text: $searchText, prompt: "Rechercher un mouvement")
    }

    // MARK: - Navigation de mois

    private var monthHeader: some View {
        HStack {
            Button {
                shiftMonth(-1)
            } label: {
                Image(systemName: "chevron.left")
            }
            Spacer()
            Text(month.formatted(.dateTime.month(.wide).year()))
                .font(.headline)
                .contentTransition(.numericText())
                .animation(.snappy, value: month)
            Spacer()
            Button {
                shiftMonth(1)
            } label: {
                Image(systemName: "chevron.right")
            }
            .disabled(isCurrentMonth)
        }
        .listRowBackground(Color.clear)
        .buttonStyle(.borderless)
    }

    private var isCurrentMonth: Bool {
        Calendar.current.isDate(month, equalTo: .now, toGranularity: .month)
    }

    private func shiftMonth(_ delta: Int) {
        if let next = Calendar.current.date(byAdding: .month, value: delta, to: month) {
            month = next
        }
    }

    // MARK: - Résumé du mois

    private func summarySection(_ s: MonthSummary) -> some View {
        Section {
            HStack(spacing: 12) {
                summaryTile("Revenus", cents: s.income, tint: OpaleTheme.gain)
                summaryTile("Dépenses", cents: Cents(-s.expenses.raw), tint: OpaleTheme.loss)
                summaryTile("Solde", cents: s.net, tint: OpaleTheme.accent)
            }
            .listRowInsets(EdgeInsets())
            .listRowBackground(Color.clear)
        }
    }

    private func summaryTile(_ label: String, cents: Cents, tint: Color) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(label)
                .font(.caption2.weight(.semibold))
                .foregroundStyle(.secondary)
                .textCase(.uppercase)
            AmountText(cents: cents, style: .whole)
                .font(.callout.weight(.bold))
                .foregroundStyle(tint)
                .minimumScaleFactor(0.7)
                .lineLimit(1)
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .glassEffect(.regular, in: .rect(cornerRadius: 16))
    }

    // MARK: - Liste groupée par jour

    private var grouped: [(day: Date, items: [Transaction])] {
        let dict = Dictionary(grouping: transactions) {
            Calendar.current.startOfDay(for: $0.occurredOn)
        }
        return dict.keys.sorted(by: >).map { (day: $0, items: dict[$0]!) }
    }

    @ViewBuilder
    private var transactionSections: some View {
        if transactions.isEmpty {
            ContentUnavailableView(
                "Aucun mouvement",
                systemImage: "arrow.left.arrow.right",
                description: Text("Importe un relevé CSV ou ajoute une transaction avec le bouton +.")
            )
            .listRowBackground(Color.clear)
        } else {
            ForEach(grouped, id: \.day) { group in
                Section(group.day.formatted(.dateTime.weekday(.wide).day().month(.wide))) {
                    ForEach(group.items) { tx in
                        Button {
                            editing = tx
                        } label: {
                            TransactionRow(transaction: tx, categories: categories)
                        }
                        .buttonStyle(.plain)
                    }
                    .onDelete { indexSet in
                        Task { await delete(at: indexSet, in: group.items) }
                    }
                }
            }
        }
    }

    // MARK: - Chargement

    private func load() async {
        do {
            let interval = Calendar.current.dateInterval(of: .month, for: month)!
            let lastDay = Calendar.current.date(byAdding: .day, value: -1, to: interval.end)!
            let comps = Calendar.current.dateComponents([.year, .month], from: month)

            async let txs = session.api.listTransactions(
                from: interval.start.opaleDayString,
                to: lastDay.opaleDayString,
                query: searchText.isEmpty ? nil : searchText
            )
            async let sum = session.api.monthSummary(year: comps.year!, month: comps.month!)
            if categories.isEmpty {
                categories = try await session.api.listCategories()
            }
            transactions = try await txs
            summary = try await sum
            errorMessage = nil
        } catch is CancellationError {
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func delete(at indexSet: IndexSet, in items: [Transaction]) async {
        for index in indexSet {
            try? await session.api.deleteTransaction(id: items[index].id)
        }
        await load()
    }
}

// MARK: - Ligne de transaction

struct TransactionRow: View {
    let transaction: Transaction
    let categories: [Category]

    private var icon: String {
        categories.first { $0.id == transaction.categoryID }?.icon ?? "questionmark.circle.dashed"
    }

    var body: some View {
        HStack(spacing: 12) {
            Image(systemName: icon)
                .font(.body)
                .foregroundStyle(transaction.categoryID == nil ? AnyShapeStyle(.tertiary) : AnyShapeStyle(OpaleTheme.accent))
                .frame(width: 30)

            VStack(alignment: .leading, spacing: 2) {
                Text(transaction.label)
                    .font(.body.weight(.medium))
                    .lineLimit(1)
                Text(transaction.categoryName ?? "À catégoriser")
                    .font(.caption)
                    .foregroundStyle(transaction.categoryName == nil ? AnyShapeStyle(OpaleTheme.accent) : AnyShapeStyle(.secondary))
            }
            Spacer()
            AmountText(cents: transaction.amount, style: .full)
                .font(.callout.weight(.semibold))
                .foregroundStyle(transaction.amount.raw > 0 ? OpaleTheme.gain : .primary)
        }
    }
}
