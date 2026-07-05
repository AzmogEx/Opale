import SwiftUI
import Charts

/// Accueil — le cœur émotionnel d'Opale (EF-010→014).
///
/// Le patrimoine net en héros (compteur animé, dégradé irisé), sa trajectoire
/// en graphe scrubable, et les totaux actifs/dettes/cash en cartes de verre.
struct HomeView: View {
    @Environment(SessionStore.self) private var session

    enum ViewState {
        case loading
        case error(String)
        case loaded(Snapshot)
    }

    /// Données de l'écran, chargées en parallèle.
    struct Snapshot {
        var netWorth: NetWorth
        var history: NetWorthHistory
        var cash: Cents

        /// Variation depuis le point précédent de l'historique (EF-012).
        /// Affichage uniquement — le calcul de référence reste côté backend.
        var monthlyDelta: Cents? {
            let pts = history.points
            guard pts.count >= 2 else { return nil }
            return pts[pts.count - 1].net - pts[pts.count - 2].net
        }
    }

    @State private var viewState: ViewState = .loading
    @State private var selectedDate: Date?

    var body: some View {
        @Bindable var session = session
        NavigationStack {
            ZStack {
                backdrop
                content
            }
            .navigationTitle("Opale")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    // Mode discret (EF-004) : floute tous les montants.
                    Button {
                        withAnimation(.snappy) { session.discreetMode.toggle() }
                    } label: {
                        Image(systemName: session.discreetMode ? "eye.slash.fill" : "eye")
                    }
                    .sensoryFeedback(.impact(weight: .light), trigger: session.discreetMode)
                }
            }
            .task { await load() }
            .refreshable { await load() }
        }
    }

    /// Fond : voile irisé très léger, signature visuelle d'Opale.
    private var backdrop: some View {
        OpaleTheme.iridescent
            .opacity(0.14)
            .ignoresSafeArea()
    }

    @ViewBuilder
    private var content: some View {
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
        case .loaded(let snapshot):
            ScrollView {
                GlassEffectContainer(spacing: 16) {
                    VStack(spacing: 16) {
                        heroCard(snapshot)
                        chartCard(snapshot)
                        statsRow(snapshot)
                    }
                    .padding(.horizontal)
                    .padding(.bottom, 24)
                }
            }
            .scrollEdgeEffectStyle(.soft, for: .top)
        }
    }

    // MARK: - Héro : patrimoine net

    private func heroCard(_ snapshot: Snapshot) -> some View {
        GlassCard {
            VStack(alignment: .leading, spacing: 8) {
                Text("Patrimoine net")
                    .font(.footnote.weight(.semibold))
                    .foregroundStyle(.secondary)
                    .textCase(.uppercase)

                AmountText(cents: displayedNet(snapshot), style: .whole)
                    .font(.system(size: 44, weight: .bold, design: .rounded))
                    .foregroundStyle(OpaleTheme.iridescent)
                    .minimumScaleFactor(0.6)
                    .lineLimit(1)

                if let selectedDate {
                    Text(selectedDate.formatted(.dateTime.day().month(.wide).year()))
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .contentTransition(.numericText())
                } else if let delta = snapshot.monthlyDelta {
                    HStack(spacing: 6) {
                        Image(systemName: delta.raw >= 0 ? "arrow.up.right" : "arrow.down.right")
                            .font(.caption.bold())
                            .foregroundStyle(OpaleTheme.delta(delta))
                        AmountText(cents: delta, style: .signedDelta)
                            .font(.subheadline.weight(.semibold))
                        Text("ce mois-ci")
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                    }
                }
            }
        }
    }

    /// Valeur affichée dans le héro : le point scrubé, sinon le patrimoine actuel.
    private func displayedNet(_ snapshot: Snapshot) -> Cents {
        guard let selectedDate,
              let point = closestPoint(to: selectedDate, in: snapshot.history.points)
        else { return snapshot.netWorth.net }
        return point.net
    }

    // MARK: - Graphe scrubable (EF-013)

    @ViewBuilder
    private func chartCard(_ snapshot: Snapshot) -> some View {
        let points = snapshot.history.points
        GlassCard {
            VStack(alignment: .leading, spacing: 12) {
                Text("Évolution — 12 mois")
                    .font(.footnote.weight(.semibold))
                    .foregroundStyle(.secondary)
                    .textCase(.uppercase)

                if points.allSatisfy({ $0.net == .zero }) {
                    ContentUnavailableView(
                        "Pas encore d'historique",
                        systemImage: "chart.line.uptrend.xyaxis",
                        description: Text("Ajoute tes premiers actifs dans l'onglet Patrimoine.")
                    )
                    .frame(minHeight: 160)
                } else {
                    netWorthChart(points)
                        .frame(minHeight: 180, maxHeight: 260)
                }
            }
        }
    }

    private func netWorthChart(_ points: [NetWorthPoint]) -> some View {
        Chart(points) { point in
            // Aplat dégradé sous la courbe.
            AreaMark(
                x: .value("Date", point.asOf),
                y: .value("Patrimoine", point.net.chartValue)
            )
            .interpolationMethod(.catmullRom)
            .foregroundStyle(
                .linearGradient(
                    colors: [OpaleTheme.accent.opacity(0.35), .clear],
                    startPoint: .top, endPoint: .bottom
                )
            )

            // La courbe elle-même.
            LineMark(
                x: .value("Date", point.asOf),
                y: .value("Patrimoine", point.net.chartValue)
            )
            .interpolationMethod(.catmullRom)
            .lineStyle(StrokeStyle(lineWidth: 3, lineCap: .round))
            .foregroundStyle(OpaleTheme.accent)
            .accessibilityLabel(point.asOf.formatted(.dateTime.month(.wide).year()))
            .accessibilityValue(MoneyFormat.eurosWhole(point.net))

            // Repère du point scrubé.
            if let selectedDate,
               let selected = closestPoint(to: selectedDate, in: points) {
                RuleMark(x: .value("Sélection", selected.asOf))
                    .foregroundStyle(.secondary.opacity(0.5))
                    .lineStyle(StrokeStyle(lineWidth: 1, dash: [4, 4]))
                PointMark(
                    x: .value("Sélection", selected.asOf),
                    y: .value("Patrimoine", selected.net.chartValue)
                )
                .symbolSize(120)
                .foregroundStyle(OpaleTheme.accent)
            }
        }
        .chartXSelection(value: $selectedDate)
        .chartYScale(domain: .automatic(includesZero: false))
        .chartXAxis {
            AxisMarks(values: .stride(by: .month, count: 3)) { _ in
                AxisGridLine()
                AxisValueLabel(format: .dateTime.month(.abbreviated))
            }
        }
        .chartYAxis {
            AxisMarks(position: .trailing) { value in
                AxisGridLine()
                AxisValueLabel {
                    if let euros = value.as(Double.self) {
                        Text(Self.compactEuros(euros))
                    }
                }
            }
        }
        .sensoryFeedback(.selection, trigger: selectedDate)
    }

    /// "48 k€" — étiquettes compactes de l'axe Y (affichage uniquement).
    private static func compactEuros(_ euros: Double) -> String {
        if abs(euros) >= 1_000_000 { return String(format: "%.1f M€", euros / 1_000_000) }
        if abs(euros) >= 1_000 { return String(format: "%.0f k€", euros / 1_000) }
        return String(format: "%.0f €", euros)
    }

    private func closestPoint(to date: Date, in points: [NetWorthPoint]) -> NetWorthPoint? {
        points.min { abs($0.asOf.timeIntervalSince(date)) < abs($1.asOf.timeIntervalSince(date)) }
    }

    // MARK: - Cartes de synthèse (EF-014)

    private func statsRow(_ snapshot: Snapshot) -> some View {
        HStack(spacing: 16) {
            GlassStat(
                label: "Actifs",
                systemImage: "arrow.up.right.circle.fill",
                cents: snapshot.netWorth.assetsTotal
            )
            GlassStat(
                label: "Dettes",
                systemImage: "arrow.down.right.circle.fill",
                cents: snapshot.netWorth.liabilitiesTotal,
                tint: OpaleTheme.loss
            )
            GlassStat(
                label: "Cash",
                systemImage: "banknote.fill",
                cents: snapshot.cash,
                tint: OpaleTheme.gain
            )
        }
    }

    // MARK: - Chargement

    private func load() async {
        do {
            async let netWorth = session.api.netWorth()
            async let history = session.api.netWorthHistory(months: 12)
            async let assets = session.api.listAssets()

            // Cash disponible (EF-014) : somme des comptes courants + livrets.
            // Somme d'affichage en entiers ; le patrimoine net, lui, vient du backend.
            let cash = try await assets
                .filter { ($0.kind == .checking || $0.kind == .savings) && !$0.archived }
                .compactMap(\.latestValue)
                .reduce(.zero, +)

            viewState = .loaded(Snapshot(
                netWorth: try await netWorth,
                history: try await history,
                cash: cash
            ))
        } catch {
            viewState = .error(error.localizedDescription)
        }
    }
}
