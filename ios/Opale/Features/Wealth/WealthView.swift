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
    @State private var showArchived = false
    @State private var deletingAsset: Asset?
    @State private var deletingLiability: Liability?
    @State private var mutationError: String?
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
                            .accessibilityLabel("Ajouter au patrimoine")
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
            .confirmationDialog("Supprimer définitivement ?", isPresented: Binding(get: { deletingAsset != nil || deletingLiability != nil }, set: { if !$0 { deletingAsset = nil; deletingLiability = nil } }), titleVisibility: .visible) {
                Button("Supprimer", role: .destructive) { Task { await deleteSelected() } }
            } message: { Text("L’historique et les mouvements associés seront supprimés. Pour garder l’historique, archive la fiche à la place.") }
            .alert("Modification impossible", isPresented: Binding(get: { mutationError != nil }, set: { if !$0 { mutationError = nil } })) { Button("OK") { mutationError = nil } } message: { Text(mutationError ?? "") }
            .task(id: session.refreshID) { await load() }
            .refreshable { await load() }
        }
    }

    @ViewBuilder
    private func list(assets: [Asset], liabilities: [Liability]) -> some View {
        List {
            let missing = assets.filter { !$0.archived && $0.latestValue == nil }.count + liabilities.filter { !$0.archived && $0.latestValue == nil }.count
            if missing > 0 { Section { Label("Total incomplet : \(missing) actifs ou dettes sans valorisation", systemImage: "exclamationmark.triangle").foregroundStyle(.orange).font(.caption) } }
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
            Section { Toggle("Afficher les archives", isOn: $showArchived) }
            Section("Actifs") {
                if assets.isEmpty {
                    Text("Aucun actif — ajoute ton premier compte, livret ou bien.")
                        .foregroundStyle(.secondary)
                }
                ForEach(assets.filter { showArchived || !$0.archived }) { asset in
                    NavigationLink(value: asset) {
                        row(
                            name: asset.name,
                            systemImage: asset.kind.systemImage,
                            kindLabel: asset.currency == "EUR"
                                ? asset.kind.label
                                : asset.kind.label + " · " + asset.currency,
                            value: asset.currentValue ?? asset.latestValue,
                            negative: false, currency: asset.currency
                        )
                    }
                }
                .onDelete { indexSet in
                    if let index = indexSet.first { deletingAsset = assets.filter { showArchived || !$0.archived }[index] }
                }
            }
            Section("Dettes") {
                if liabilities.isEmpty {
                    Text("Aucune dette.")
                        .foregroundStyle(.secondary)
                }
                ForEach(liabilities.filter { showArchived || !$0.archived }) { liability in
                    NavigationLink(value: liability) {
                        row(
                            name: liability.name,
                            systemImage: liability.kind.systemImage,
                            kindLabel: liability.kind.label,
                            value: liability.latestValue,
                            negative: true, currency: liability.currency
                        )
                    }
                }
                .onDelete { indexSet in
                    if let index = indexSet.first { deletingLiability = liabilities.filter { showArchived || !$0.archived }[index] }
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
        negative: Bool, currency: String = "EUR"
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
                AmountText(cents: negative ? Cents(-value.raw) : value, style: .whole, currency: currency)
                    .font(.callout.weight(.semibold))
                    .foregroundStyle(negative ? OpaleTheme.loss : .primary)
            } else {
                Text("À valoriser")
                    .font(.caption)
                    .foregroundStyle(.tertiary)
            }
        }
    }

    /// Instantané Codable du mode hors-ligne.
    private struct WealthCacheSnapshot: Codable {
        var assets: [Asset]
        var liabilities: [Liability]
    }

    private var cacheKey: String { "wealth-\(session.profileKey)" }

    private func load() async {
        let requestedProfile = session.profileKey
        let requestedCacheKey = cacheKey
        if case .loading = viewState,
           let cached = DiskCache.load(WealthCacheSnapshot.self, key: requestedCacheKey) {
            viewState = .loaded(cached.value.assets, cached.value.liabilities)
        }
        do {
            async let assets = session.api.listAssets()
            async let liabilities = session.api.listLiabilities()
            let (a, l) = (try await assets, try await liabilities)
            guard requestedProfile == session.profileKey, !Task.isCancelled else { return }
            viewState = .loaded(a, l)
            DiskCache.save(WealthCacheSnapshot(assets: a, liabilities: l), key: requestedCacheKey)
        } catch {
            guard requestedProfile == session.profileKey, !Task.isCancelled else { return }
            if case .loaded = viewState { return } // on reste sur le cache
            viewState = .error(error.localizedDescription)
        }
    }

    private func deleteSelected() async {
        do {
            if let item = deletingAsset { try await session.api.deleteAsset(id: item.id) }
            if let item = deletingLiability { try await session.api.deleteLiability(id: item.id) }
            deletingAsset = nil; deletingLiability = nil
            session.changed()
            await load()
        } catch { mutationError = error.localizedDescription; deletingAsset = nil; deletingLiability = nil }
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

struct AssetDetailView: View {
    let asset: Asset
    var onChanged: () -> Void
    var body: some View { HoldingDetailView(id: asset.id, kind: asset.kind.label, currency: asset.currency, liability: false, onChanged: onChanged, name: asset.name, note: asset.note, archived: asset.archived) }
}
struct LiabilityDetailView: View {
    let liability: Liability
    var onChanged: () -> Void
    var body: some View { HoldingDetailView(id: liability.id, kind: liability.kind.label, currency: liability.currency, liability: true, onChanged: onChanged, name: liability.name, note: liability.note, archived: liability.archived) }
}
