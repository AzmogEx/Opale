import SwiftUI

/// Onglet Patrimoine — actifs et passifs, saisie manuelle (EF-030→032).
struct WealthView: View {
    @Environment(SessionStore.self) private var session

    enum ViewState {
        case loading
        case error(String)
        case loaded([Asset], [Liability])
    }

    enum Sheet: String, Identifiable {
        case newAsset, newLiability, fxRates
        var id: String { rawValue }
    }

    @State private var viewState: ViewState = .loading
    @State private var activeSheet: Sheet?
    @State private var selectedCenter: WealthCenter?
    /// Transition héros : la tuile du centre DEVIENT l'écran.
    @Namespace private var zoomSpace

    var body: some View {
        NavigationStack {
            Group {
                switch viewState {
                case .loading:
                    ProgressView()
                case .error(let message):
                    ContentUnavailableView {
                        Label("Impossible de charger", systemImage: "bolt.horizontal.circle")
                    } description: {
                        Text(message)
                    } actions: {
                        Button("Réessayer") { Task { await load() } }
                            .buttonStyle(.glassProminent)
                    }
                case .loaded(let assets, let liabilities):
                    list(assets: assets, liabilities: liabilities)
                }
            }
            .navigationTitle("Patrimoine")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Menu {
                        Button {
                            activeSheet = .newAsset
                        } label: {
                            Label("Nouvel actif", systemImage: "plus.circle")
                        }
                        Button {
                            activeSheet = .newLiability
                        } label: {
                            Label("Nouvelle dette", systemImage: "minus.circle")
                        }
                        Divider()
                        Button {
                            activeSheet = .fxRates
                        } label: {
                            Label("Devises & taux", systemImage: "eurosign.arrow.circlepath")
                        }
                    } label: {
                        Image(systemName: "plus")
                    }
                }
            }
            .sheet(item: $activeSheet) { sheet in
                switch sheet {
                case .newAsset:
                    AssetFormSheet { Task { await load() } }
                case .newLiability:
                    LiabilityFormSheet { Task { await load() } }
                case .fxRates:
                    FXRatesSheet { Task { await load() } }
                }
            }
            .navigationDestination(for: Asset.self) { asset in
                AssetDetailView(asset: asset) { Task { await load() } }
            }
            .navigationDestination(item: $selectedCenter) { center in
                Group {
                    switch center {
                    case .realEstate: RealEstateView()
                    case .investments: InvestmentsView()
                    case .objects: ObjectsView()
                    case .company: CompanyView()
                    case .timeline: TimelineView()
                    case .vault: VaultView()
                    case .transmission: TransmissionView()
                    }
                }
                .navigationTransition(.zoom(sourceID: center, in: zoomSpace))
            }
            .navigationDestination(for: Liability.self) { liability in
                LiabilityDetailView(liability: liability) { Task { await load() } }
            }
            .task { await load() }
            .refreshable { await load() }
        }
    }

    @ViewBuilder
    private func list(assets: [Asset], liabilities: [Liability]) -> some View {
        List {
            // La profondeur (P6) : les centres spécialisés.
            // ⚠️ Pas de NavigationLink ici : plusieurs liens dans UNE ligne de
            // List routent tous les taps vers le premier. Boutons `.plain`
            // (hit-test individuel) + navigation programmatique.
            Section("Centres") {
                LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible()), GridItem(.flexible())], spacing: 12) {
                    ForEach(WealthCenter.allCases) { center in
                        Button {
                            selectedCenter = center
                        } label: {
                            VStack(spacing: 6) {
                                Image(systemName: center.systemImage)
                                    .font(.title3)
                                    .foregroundStyle(OpaleTheme.accent)
                                Text(center.label)
                                    .font(.caption2.weight(.semibold))
                                    .multilineTextAlignment(.center)
                                    .lineLimit(2)
                            }
                            .frame(maxWidth: .infinity, minHeight: 64)
                            .contentShape(.rect)
                        }
                        .buttonStyle(.pressable)
                        .matchedTransitionSource(id: center, in: zoomSpace)
                    }
                }
                .listRowInsets(EdgeInsets(top: 8, leading: 8, bottom: 8, trailing: 8))
            }
            Section("Actifs") {
                if assets.isEmpty {
                    Text("Aucun actif — ajoute ton premier compte, livret ou bien.")
                        .foregroundStyle(.secondary)
                }
                ForEach(assets) { asset in
                    NavigationLink(value: asset) {
                        row(
                            name: asset.name,
                            systemImage: asset.kind.systemImage,
                            kindLabel: asset.currency == "EUR"
                                ? asset.kind.label
                                : asset.kind.label + " · " + asset.currency,
                            value: asset.latestValue,
                            negative: false
                        )
                    }
                }
                .onDelete { indexSet in
                    Task { await deleteAssets(at: indexSet, in: assets) }
                }
            }
            Section("Dettes") {
                if liabilities.isEmpty {
                    Text("Aucune dette.")
                        .foregroundStyle(.secondary)
                }
                ForEach(liabilities) { liability in
                    NavigationLink(value: liability) {
                        row(
                            name: liability.name,
                            systemImage: liability.kind.systemImage,
                            kindLabel: liability.kind.label,
                            value: liability.latestValue,
                            negative: true
                        )
                    }
                }
                .onDelete { indexSet in
                    Task { await deleteLiabilities(at: indexSet, in: liabilities) }
                }
            }
        }
        .opaleList()
    }

    private func row(
        name: String,
        systemImage: String,
        kindLabel: String,
        value: Cents?,
        negative: Bool
    ) -> some View {
        HStack(spacing: 12) {
            ZStack {
                Circle()
                    .fill(negative ? AnyShapeStyle(OpaleTheme.loss.opacity(0.14))
                                   : AnyShapeStyle(OpaleTheme.iridescent.opacity(0.18)))
                Image(systemName: systemImage)
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(negative ? AnyShapeStyle(OpaleTheme.loss) : AnyShapeStyle(OpaleTheme.iridescent))
            }
            .frame(width: 38, height: 38)
            VStack(alignment: .leading, spacing: 2) {
                Text(name).font(.body.weight(.medium))
                Text(kindLabel).font(.caption).foregroundStyle(.secondary)
            }
            Spacer()
            if let value {
                AmountText(cents: negative ? Cents(-value.raw) : value, style: .whole)
                    .font(.callout.weight(.semibold))
                    .foregroundStyle(negative ? OpaleTheme.loss : .primary)
            } else {
                Text("À valoriser")
                    .font(.caption)
                    .foregroundStyle(.tertiary)
            }
        }
    }

    private func load() async {
        do {
            async let assets = session.api.listAssets()
            async let liabilities = session.api.listLiabilities()
            viewState = .loaded(try await assets, try await liabilities)
        } catch {
            viewState = .error(error.localizedDescription)
        }
    }

    private func deleteAssets(at indexSet: IndexSet, in assets: [Asset]) async {
        for index in indexSet {
            try? await session.api.deleteAsset(id: assets[index].id)
        }
        await load()
    }

    private func deleteLiabilities(at indexSet: IndexSet, in liabilities: [Liability]) async {
        for index in indexSet {
            try? await session.api.deleteLiability(id: liabilities[index].id)
        }
        await load()
    }
}

// MARK: - Centres (P6)

/// Les six centres de la profondeur patrimoniale.
enum WealthCenter: String, CaseIterable, Identifiable, Hashable {
    case realEstate, investments, objects, company, timeline, vault, transmission

    var id: String { rawValue }

    var label: String {
        switch self {
        case .realEstate: "Immobilier"
        case .investments: "Placements"
        case .objects: "Objets"
        case .company: "Entreprise"
        case .timeline: "Timeline"
        case .vault: "Coffre-fort"
        case .transmission: "Transmission"
        }
    }

    var systemImage: String {
        switch self {
        case .realEstate: "house.fill"
        case .investments: "chart.pie.fill"
        case .objects: "sparkle.magnifyingglass"
        case .company: "briefcase.fill"
        case .timeline: "calendar.day.timeline.left"
        case .vault: "lock.doc.fill"
        case .transmission: "figure.2.and.child.holdinghands"
        }
    }
}

// MARK: - Détails

/// Détail d'un actif : historique des valorisations + ajout (EF-032).
struct AssetDetailView: View {
    let asset: Asset
    var onChanged: () -> Void

    @Environment(SessionStore.self) private var session
    @State private var valuations: [Valuation] = []
    @State private var showValuationSheet = false

    var body: some View {
        ValuationHistoryList(
            title: asset.name,
            subtitle: asset.kind.label,
            systemImage: asset.kind.systemImage,
            valuations: valuations,
            negative: false,
            onAdd: { showValuationSheet = true }
        )
        .task { await load() }
        .sheet(isPresented: $showValuationSheet) {
            ValuationSheet(
                title: asset.name,
                save: { cents, day in
                    _ = try await session.api.addAssetValuation(
                        assetID: asset.id, valueCents: cents, asOf: day
                    )
                },
                onSaved: {
                    Task { await load() }
                    onChanged()
                }
            )
            .presentationDetents([.medium])
        }
    }

    private func load() async {
        valuations = (try? await session.api.assetValuations(assetID: asset.id)) ?? []
    }
}

/// Détail d'une dette : historique du capital restant dû + ajout.
struct LiabilityDetailView: View {
    let liability: Liability
    var onChanged: () -> Void

    @Environment(SessionStore.self) private var session
    @State private var valuations: [Valuation] = []
    @State private var showValuationSheet = false

    var body: some View {
        ValuationHistoryList(
            title: liability.name,
            subtitle: liability.kind.label,
            systemImage: liability.kind.systemImage,
            valuations: valuations,
            negative: true,
            onAdd: { showValuationSheet = true }
        )
        .task { await load() }
        .sheet(isPresented: $showValuationSheet) {
            ValuationSheet(
                title: liability.name,
                save: { cents, day in
                    _ = try await session.api.addLiabilityValuation(
                        liabilityID: liability.id, valueCents: cents, asOf: day
                    )
                },
                onSaved: {
                    Task { await load() }
                    onChanged()
                }
            )
            .presentationDetents([.medium])
        }
    }

    private func load() async {
        valuations = (try? await session.api.liabilityValuations(liabilityID: liability.id)) ?? []
    }
}

/// Liste partagée de l'historique des valorisations.
private struct ValuationHistoryList: View {
    var title: String
    var subtitle: String
    var systemImage: String
    var valuations: [Valuation]
    var negative: Bool
    var onAdd: () -> Void

    var body: some View {
        List {
            Section {
                HStack(spacing: 12) {
                    Image(systemName: systemImage)
                        .font(.title)
                        .foregroundStyle(negative ? OpaleTheme.loss : OpaleTheme.accent)
                    VStack(alignment: .leading) {
                        Text(title).font(.headline)
                        Text(subtitle).font(.caption).foregroundStyle(.secondary)
                    }
                    Spacer()
                    if let latest = valuations.first {
                        AmountText(cents: latest.value, style: .whole)
                            .font(.title3.weight(.bold))
                    }
                }
            }
            Section("Historique") {
                if valuations.isEmpty {
                    Text("Aucune valorisation — ajoute la première.")
                        .foregroundStyle(.secondary)
                }
                ForEach(valuations) { valuation in
                    HStack {
                        Text(valuation.asOf.formatted(.dateTime.day().month(.abbreviated).year()))
                            .foregroundStyle(.secondary)
                        Spacer()
                        AmountText(cents: valuation.value, style: .full)
                            .font(.callout.weight(.medium))
                    }
                }
            }
        }
        .navigationTitle(title)
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    onAdd()
                } label: {
                    Label("Nouvelle valorisation", systemImage: "plus")
                }
            }
        }
    }
}
