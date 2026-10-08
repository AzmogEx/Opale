import SwiftUI

/// Onglet Flux — le quotidien (EF-020→022) : mouvements du mois, recherche,
/// résumé revenus/dépenses, correction de catégorie apprenante, import CSV.
struct FlowsView: View {
    @Environment(SessionStore.self) private var session

    /// Premier jour du mois affiché.
    @State private var month = Calendar.opale.dateInterval(of: .month, for: .now)!.start
    @State private var summary: MonthSummary?
    @State private var transactions: [Transaction] = []
    @State private var categories: [Category] = []
    @State private var accounts: [Asset] = []
    @State private var categoryFilter = ""
    @State private var accountFilter = ""
    @State private var canLoadMore = false
    @State private var loadingMore = false
    @State private var searchText = ""
    @State private var errorMessage: String?

    @State private var editing: Transaction?
    @State private var showManualForm = false
    @State private var showTransfer = false
    @State private var deleting: Transaction?
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

    private var monthKey: String { String(month.opaleDayString.prefix(7)) }

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
                    if segment == .movements { filterBar }
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
                        Button("Virement entre mes comptes", systemImage: "arrow.left.arrow.right") { showTransfer = true }
                        Button {
                            showManualForm = true
                        } label: {
                            Label("Nouvelle transaction", systemImage: "plus.circle")
                        }
                        Button {
                            showImport = true
                        } label: {
                            Label("Importer un relevé (CSV / OFX)", systemImage: "square.and.arrow.down")
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
            .task(id: requestKey) { await load() }
            .refreshable { await load() }
            .sheet(item: $editing) { tx in
                TransactionEditSheet(transaction: tx, categories: categories) {
                    Task { await load() }
                }
                .presentationDetents([.medium, .large])
            }
            .sheet(isPresented: $showTransfer) { TransferSheet { Task { await load() } } }
            .confirmationDialog(deleting?.transferID == nil ? "Supprimer ce mouvement ?" : "Supprimer le virement entier ?", isPresented: Binding(get: { deleting != nil }, set: { if !$0 { deleting = nil } }), titleVisibility: .visible) {
                Button("Supprimer", role: .destructive) { if let tx = deleting { Task {
                    do { if let id = tx.transferID { let _: APIClient.EmptyResponse = try await session.api.request("DELETE", "/v1/transfers/\(id)") } else { try await session.api.deleteTransaction(id: tx.id) }; session.changed(); await load() }
                    catch { errorMessage = error.localizedDescription }
                    deleting = nil
                } } }
            } message: { Text(deleting?.transferID == nil ? "Cette suppression est définitive." : "Les deux mouvements et leurs frais seront supprimés ensemble.") }
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

    private var filterBar: some View {
        HStack {
            Picker("Catégorie", selection: $categoryFilter) {
                Text("Toutes catégories").tag("")
                ForEach(categories) { Text($0.name).tag($0.id) }
            }
            Picker("Compte", selection: $accountFilter) {
                Text("Tous comptes").tag("")
                ForEach(accounts) { Text($0.name).tag($0.id) }
            }
        }.padding(.horizontal).font(.caption)
    }
    private var requestKey: String { [monthKey, searchText, categoryFilter, accountFilter, session.refreshID.uuidString].joined(separator: "|") }

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
            Text(month.opaleFormatted(.dateTime.month(.wide).year()).capitalized)
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
        Calendar.opale.isDate(month, equalTo: .now, toGranularity: .month)
    }

    private func shiftMonth(_ delta: Int) {
        if let next = Calendar.opale.date(byAdding: .month, value: delta, to: month) {
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
            Calendar.opale.startOfDay(for: $0.occurredOn)
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
                    Text(group.day.opaleFormatted(.dateTime.weekday(.wide).day().month(.wide)).capitalized)
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
                                        deleting = tx
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
            if canLoadMore { Button(loadingMore ? "Chargement…" : "Charger la suite") { Task { await loadMore() } }.disabled(loadingMore) }
        }
    }

    private func loadMore() async {
        guard !loadingMore else { return }
        let key = requestKey
        let profile = session.profileKey
        loadingMore = true; defer { loadingMore = false }
        do {
            let interval = Calendar.opale.dateInterval(of: .month, for: month)!
            let lastDay = Calendar.opale.date(byAdding: .day, value: -1, to: interval.end)!
            let page = try await session.api.listTransactions(from: interval.start.opaleDayString, to: lastDay.opaleDayString, query: searchText.isEmpty ? nil : searchText, categoryID: categoryFilter.isEmpty ? nil : categoryFilter, assetID: accountFilter.isEmpty ? nil : accountFilter, offset: transactions.count)
            guard key == requestKey, profile == session.profileKey, !Task.isCancelled else { return }
            transactions.append(contentsOf: page)
            canLoadMore = page.count == 100
        } catch { errorMessage = error.localizedDescription }
    }

    // MARK: - Chargement

    private func load() async {
        let key = requestKey
        let profile = session.profileKey
        do {
            let interval = Calendar.opale.dateInterval(of: .month, for: month)!
            let lastDay = Calendar.opale.date(byAdding: .day, value: -1, to: interval.end)!
            let comps = Calendar.opale.dateComponents([.year, .month], from: month)

            async let txs = session.api.listTransactions(
                from: interval.start.opaleDayString,
                to: lastDay.opaleDayString,
                query: searchText.isEmpty ? nil : searchText,
                categoryID: categoryFilter.isEmpty ? nil : categoryFilter,
                assetID: accountFilter.isEmpty ? nil : accountFilter
            )
            async let sum = session.api.monthSummary(year: comps.year!, month: comps.month!)
            async let loadedCategories = session.api.listCategories()
            async let loadedAccounts = session.api.listAssets()
            let (loadedTransactions, loadedSummary, newCategories, newAccounts) = try await (txs, sum, loadedCategories, loadedAccounts)
            guard key == requestKey, profile == session.profileKey, !Task.isCancelled else { return }
            categories = newCategories
            accounts = newAccounts
            transactions = loadedTransactions
            canLoadMore = transactions.count == 100
            summary = loadedSummary
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
            AmountText(cents: transaction.amount, style: .full, currency: transaction.currency ?? "EUR")
                .font(.callout.weight(.semibold))
                .foregroundStyle(transaction.amount.raw > 0 ? OpaleTheme.gain : .primary)
        }
    }
}
