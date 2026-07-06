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
    @State private var showBank = false

    /// Sous-vues de l'onglet Flux (EF-020 / EF-028 / EF-025-027).
    enum Segment: String, CaseIterable, Identifiable {
        case movements = "Mouvements"
        case envelopes = "Enveloppes"
        case upcoming = "À venir"
        case shared = "Commun"
        var id: String { rawValue }
    }

    @State private var segment: Segment = .movements

    private var monthKey: String {
        month.formatted(.iso8601.year().month())
    }

    /// Sélection glissante des segments (pill qui voyage).
    @Namespace private var segmentSpace
    @FocusState private var searchFocused: Bool

    var body: some View {
        NavigationStack {
            ZStack {
                // Le fond signature couvre TOUTE la page — plus de bloc noir.
                OpaleBackdrop()

                VStack(spacing: 12) {
                    header
                    switch segment {
                    case .movements: movementsList
                    case .envelopes: EnvelopesView()
                    case .upcoming: UpcomingView()
                    case .shared: SharedSpaceView()
                    }
                }
            }
            .navigationTitle("Flux")
            .navigationBarTitleDisplayMode(.inline)
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
                        Button {
                            showBank = true
                        } label: {
                            Label("Ma banque (synchro)", systemImage: "building.columns")
                        }
                    } label: {
                        Image(systemName: "plus")
                    }
                    .accessibilityLabel("Ajouter")
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
            .sheet(isPresented: $showBank) {
                BankSheet { Task { await load() } }
            }
        }
    }

    // MARK: - En-tête custom : recherche en verre + pills animées

    private var header: some View {
        VStack(spacing: 10) {
            // Recherche — une barre de verre, pas le bloc système.
            HStack(spacing: 8) {
                Image(systemName: "magnifyingglass")
                    .foregroundStyle(.secondary)
                TextField("Rechercher un mouvement", text: $searchText)
                    .focused($searchFocused)
                    .submitLabel(.search)
                if !searchText.isEmpty {
                    Button {
                        searchText = ""
                        searchFocused = false
                    } label: {
                        Image(systemName: "xmark.circle.fill")
                            .foregroundStyle(.tertiary)
                    }
                }
            }
            .padding(.horizontal, 14)
            .padding(.vertical, 10)
            .glassEffect(.regular, in: .capsule)

            // Segments : la pill de sélection GLISSE d'un onglet à l'autre.
            HStack(spacing: 4) {
                ForEach(Segment.allCases) { s in
                    Button {
                        withAnimation(.snappy(duration: 0.3)) { segment = s }
                    } label: {
                        Text(s.rawValue)
                            .font(.footnote.weight(segment == s ? .bold : .medium))
                            .foregroundStyle(segment == s ? OpaleTheme.accent : .secondary)
                            .padding(.vertical, 8)
                            .frame(maxWidth: .infinity)
                            .background {
                                if segment == s {
                                    Capsule()
                                        .fill(OpaleTheme.accent.opacity(0.16))
                                        .matchedGeometryEffect(id: "pill", in: segmentSpace)
                                }
                            }
                    }
                    .buttonStyle(.plain)
                }
            }
            .padding(4)
            .glassEffect(.regular, in: .capsule)
            .sensoryFeedback(.selection, trigger: segment)
        }
        .padding(.horizontal)
    }

    // MARK: - Segment Mouvements (EF-020)

    private var movementsList: some View {
        ScrollView {
            VStack(spacing: 14) {
                monthHeader
                if let summary {
                    summaryTiles(summary)
                }
                transactionSections
                if let errorMessage {
                    Text(errorMessage).foregroundStyle(OpaleTheme.loss)
                }
            }
            .padding(.horizontal)
            .padding(.bottom, 24)
        }
        .scrollEdgeEffectStyle(.soft, for: .top)
        .scrollDismissesKeyboard(.immediately)
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
            Text(month.formatted(.dateTime.month(.wide).year()).capitalized)
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
        .buttonStyle(.borderless)
        .padding(.horizontal, 6)
        .sensoryFeedback(.selection, trigger: month)
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

    private func summaryTiles(_ s: MonthSummary) -> some View {
        HStack(spacing: 12) {
            summaryTile("Revenus", cents: s.income, tint: OpaleTheme.gain)
            summaryTile("Dépenses", cents: Cents(-s.expenses.raw), tint: OpaleTheme.loss)
            summaryTile("Solde", cents: s.net, tint: OpaleTheme.accent)
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
            EmptyStateView(
                icon: "arrow.left.arrow.right",
                title: "Aucun mouvement",
                message: "Importe un relevé CSV ou ajoute une transaction avec le bouton +."
            )
        } else {
            // Une carte de verre par jour — l'écran respire.
            ForEach(Array(grouped.enumerated()), id: \.element.day) { index, group in
                VStack(alignment: .leading, spacing: 8) {
                    Text(group.day.formatted(.dateTime.weekday(.wide).day().month(.wide)).capitalized)
                        .font(.footnote.weight(.semibold))
                        .foregroundStyle(.secondary)
                        .padding(.leading, 6)

                    GlassCard {
                        VStack(spacing: 0) {
                            ForEach(group.items) { tx in
                                Button {
                                    editing = tx
                                } label: {
                                    TransactionRow(transaction: tx, categories: categories)
                                        .contentShape(.rect)
                                }
                                .buttonStyle(.pressable)
                                .contextMenu {
                                    Button {
                                        editing = tx
                                    } label: {
                                        Label("Modifier", systemImage: "square.and.pencil")
                                    }
                                    Button(role: .destructive) {
                                        Task {
                                            try? await session.api.deleteTransaction(id: tx.id)
                                            await load()
                                        }
                                    } label: {
                                        Label("Supprimer", systemImage: "trash")
                                    }
                                }
                                if tx.id != group.items.last?.id {
                                    Divider().padding(.leading, 50)
                                }
                            }
                        }
                    }
                }
                .cascadeIn(index)
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
            // Pastille dégradée façon avatar de marchand.
            ZStack {
                Circle()
                    .fill(OpaleTheme.iridescent)
                    .opacity(transaction.categoryID == nil ? 0.10 : 0.18)
                Image(systemName: icon)
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(transaction.categoryID == nil ? AnyShapeStyle(.tertiary) : AnyShapeStyle(OpaleTheme.iridescent))
            }
            .frame(width: 38, height: 38)

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
