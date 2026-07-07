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

    /// Données de l'écran, chargées en parallèle. Codable : c'est aussi
    /// l'instantané du mode hors-ligne (DiskCache).
    struct Snapshot: Codable {
        var netWorth: NetWorth
        var history: NetWorthHistory
        var cash: Cents
        var health: HealthScore?
        var alerts: [OpaleAlert] = []

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
    @State private var showSettings = false
    /// Mode hors-ligne : date du cache affiché quand l'API est injoignable.
    @State private var offlineSince: Date?
    /// Transition héros vers l'écran Analyses.
    @Namespace private var zoomSpace

    // Célébration de palier (EF-016) : dernier palier déjà fêté (euros).
    @AppStorage("home.celebratedMilestone") private var celebratedMilestone = 0
    @State private var showConfetti = false

    /// Paliers de patrimoine net (euros) — alignés sur la timeline (EF-045).
    private static let milestones = [10_000, 25_000, 50_000, 100_000, 250_000, 500_000, 1_000_000]

    var body: some View {
        @Bindable var session = session
        NavigationStack {
            ZStack {
                backdrop
                content
                if showConfetti {
                    ConfettiView()
                        .transition(.opacity)
                }
            }
            // Palier franchi : la célébration se SENT aussi (EF-016).
            .sensoryFeedback(.success, trigger: showConfetti) { _, new in new }
            .navigationTitle("Opale")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    // Mode discret (EF-004) : floute tous les montants.
                    Button {
                        withAnimation(.snappy) { session.discreetMode.toggle() }
                        WidgetBridge.setDiscreet(session.discreetMode)
                    } label: {
                        Image(systemName: session.discreetMode ? "eye.slash.fill" : "eye")
                            .contentTransition(.symbolEffect(.replace))
                    }
                    .sensoryFeedback(.impact(weight: .light), trigger: session.discreetMode)
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button {
                        showSettings = true
                    } label: {
                        Image(systemName: "gearshape")
                    }
                    .accessibilityLabel("Réglages")
                }
            }
            .sheet(isPresented: $showSettings) {
                SettingsView()
            }
            .task { await load() }
            .refreshable { await load() }
        }
    }

    /// Fond signature : voile irisé en clair, nuit à lueurs en sombre.
    private var backdrop: some View {
        OpaleBackdrop()
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
                    // L'écran se construit en cascade — chaque carte entre
                    // en scène avec son propre ressort (feel Revolut).
                    VStack(spacing: 16) {
                        if let offlineSince {
                            Label("Hors ligne — données du \(offlineSince.formatted(.dateTime.day().month().hour().minute()))",
                                  systemImage: "wifi.slash")
                                .font(.caption.weight(.medium))
                                .foregroundStyle(.secondary)
                                .padding(.horizontal, 12)
                                .padding(.vertical, 6)
                                .glassEffect(.regular, in: .capsule)
                        }
                        alertsBanner(snapshot.alerts)
                            .cascadeIn(0)
                        heroCard(snapshot)
                            .cascadeIn(1)
                        chartCard(snapshot)
                            .cascadeIn(2)
                        analyticsCard
                            .cascadeIn(3)
                        statsRow(snapshot)
                            .cascadeIn(4)
                        if let health = snapshot.health {
                            healthCard(health)
                                .cascadeIn(5)
                        }
                    }
                    .padding(.horizontal)
                    .padding(.bottom, 24)
                }
            }
            .scrollEdgeEffectStyle(.soft, for: .top)
        }
    }

    // MARK: - Analyses & Abonnements (transitions héros)

    private var analyticsCard: some View {
        HStack(spacing: 12) {
            featureCard(
                id: "analytics",
                icon: "chart.pie.fill",
                title: "Analyses",
                subtitle: "Où part l'argent ?"
            ) {
                AnalyticsView()
            }
            featureCard(
                id: "subscriptions",
                icon: "repeat.circle.fill",
                title: "Abonnements",
                subtitle: "Le vrai coût / an"
            ) {
                SubscriptionsView()
            }
        }
    }

    /// Une carte-fonctionnalité qui DEVIENT son écran (zoom héros).
    private func featureCard(
        id: String,
        icon: String,
        title: String,
        subtitle: String,
        @ViewBuilder destination: @escaping () -> some View
    ) -> some View {
        NavigationLink {
            destination()
                .navigationTransition(.zoom(sourceID: id, in: zoomSpace))
        } label: {
            GlassCard {
                VStack(alignment: .leading, spacing: 6) {
                    Image(systemName: icon)
                        .font(.title2)
                        .foregroundStyle(OpaleTheme.iridescent)
                    Text(title)
                        .font(.subheadline.weight(.bold))
                    Text(subtitle)
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                        .lineLimit(1)
                }
                .frame(maxWidth: .infinity, alignment: .leading)
            }
        }
        .buttonStyle(.pressable)
        .matchedTransitionSource(id: id, in: zoomSpace)
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
                    .iridescentShimmer() // le dégradé respire — vivant, pas agité
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

    // MARK: - Alertes (EF-053)

    @ViewBuilder
    private func alertsBanner(_ alerts: [OpaleAlert]) -> some View {
        ForEach(alerts) { alert in
            GlassCard(tint: alert.severity == "critical" ? OpaleTheme.loss : .orange) {
                HStack(spacing: 12) {
                    Image(systemName: alert.severity == "critical"
                        ? "exclamationmark.octagon.fill" : "exclamationmark.triangle.fill")
                        .font(.title3)
                        .foregroundStyle(alert.severity == "critical" ? OpaleTheme.loss : .orange)
                    VStack(alignment: .leading, spacing: 2) {
                        Text(alert.title).font(.subheadline.weight(.semibold))
                        Text(alert.detail).font(.caption).foregroundStyle(.secondary)
                    }
                }
            }
        }
    }

    // MARK: - Score de santé (EF-015)

    @State private var showHealthDetail = false

    private func healthCard(_ health: HealthScore) -> some View {
        Button {
            showHealthDetail = true
        } label: {
            GlassCard(interactive: true) {
                HStack(spacing: 16) {
                    scoreGauge(health.score)
                    VStack(alignment: .leading, spacing: 4) {
                        Text("Santé financière")
                            .font(.footnote.weight(.semibold))
                            .foregroundStyle(.secondary)
                            .textCase(.uppercase)
                        Text(healthVerdict(health.score))
                            .font(.headline)
                        if let weakest = health.components.min(by: {
                            $0.max == 0 ? false : ($1.max == 0 ? true :
                                Double($0.score) / Double($0.max) < Double($1.score) / Double($1.max))
                        }) {
                            Text("Point faible : \(weakest.name.lowercased())")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                    }
                    Spacer()
                    Image(systemName: "chevron.right")
                        .font(.caption)
                        .foregroundStyle(.tertiary)
                }
            }
        }
        .buttonStyle(.plain)
        .sheet(isPresented: $showHealthDetail) {
            HealthDetailSheet(health: health)
                .presentationDetents([.medium, .large])
        }
    }

    private func scoreGauge(_ score: Int) -> some View {
        Gauge(value: Double(score), in: 0...100) {
            EmptyView()
        } currentValueLabel: {
            Text("\(score)")
                .font(.system(.title3, design: .rounded).bold())
        }
        .gaugeStyle(.accessoryCircularCapacity)
        .tint(score >= 70 ? OpaleTheme.gain : score >= 40 ? .orange : OpaleTheme.loss)
    }

    private func healthVerdict(_ score: Int) -> String {
        switch score {
        case 85...: "Excellente"
        case 70..<85: "Solide"
        case 50..<70: "Correcte"
        case 30..<50: "Fragile"
        default: "En difficulté"
        }
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

    private var cacheKey: String { "home-\(session.profileID)" }

    private func load() async {
        // Ouverture instantanée : le dernier état connu s'affiche tout de
        // suite, le réseau ne fait que rafraîchir.
        if case .loading = viewState,
           let cached = DiskCache.load(Snapshot.self, key: cacheKey) {
            viewState = .loaded(cached.value)
        }
        do {
            async let netWorth = session.api.netWorth()
            async let history = session.api.netWorthHistory(months: 12)
            async let assets = session.api.listAssets()
            async let health = session.api.healthScore()
            async let alerts = session.api.alerts()

            // Cash disponible (EF-014) : somme des comptes courants + livrets.
            // Somme d'affichage en entiers ; le patrimoine net, lui, vient du backend.
            let cash = try await assets
                .filter { ($0.kind == .checking || $0.kind == .savings) && !$0.archived }
                .compactMap(\.latestValue)
                .reduce(.zero, +)

            let loadedNetWorth = try await netWorth
            let loadedHistory = try await history
            viewState = .loaded(Snapshot(
                netWorth: loadedNetWorth,
                history: loadedHistory,
                cash: cash,
                health: try? await health,
                alerts: (try? await alerts) ?? []
            ))

            // Publie l'instantané pour le widget (App Group, local uniquement).
            WidgetBridge.publish(
                netWorthCents: loadedNetWorth.net.raw,
                historyCents: loadedHistory.points.map(\.net.raw)
            )

            celebrateIfMilestoneReached(net: loadedNetWorth.net)
            offlineSince = nil
            if case .loaded(let snapshot) = viewState {
                DiskCache.save(snapshot, key: cacheKey)
            }
        } catch {
            // API injoignable : on RESTE sur le cache, avec un bandeau.
            if case .loaded = viewState {
                offlineSince = DiskCache.load(Snapshot.self, key: cacheKey)?.at ?? .now
            } else {
                viewState = .error(error.localizedDescription)
            }
        }
    }

    /// Confettis (EF-016) quand un nouveau palier vient d'être franchi.
    /// Au premier lancement, on enregistre le palier courant SANS fêter
    /// (on ne célèbre que les paliers franchis « en direct »).
    private func celebrateIfMilestoneReached(net: Cents) {
        let euros = Int(net.raw / 100)
        let reached = Self.milestones.last { euros >= $0 } ?? 0
        if celebratedMilestone == 0 && reached > 0 {
            celebratedMilestone = reached
            return
        }
        guard reached > celebratedMilestone else { return }
        celebratedMilestone = reached
        SoundPlayer.play(.success)
        withAnimation(.easeIn(duration: 0.2)) { showConfetti = true }
        Task {
            try? await Task.sleep(for: .seconds(4))
            withAnimation(.easeOut(duration: 0.6)) { showConfetti = false }
        }
    }
}

/// Détail du score : les cinq composantes et leurs verdicts (EF-015).
private struct HealthDetailSheet: View {
    let health: HealthScore
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            List {
                ForEach(health.components) { c in
                    VStack(alignment: .leading, spacing: 6) {
                        HStack {
                            Text(c.name).font(.body.weight(.medium))
                            Spacer()
                            Text("\(c.score)/\(c.max)")
                                .font(.callout.weight(.bold))
                                .monospacedDigit()
                        }
                        ProgressView(value: Double(c.score), total: Double(c.max))
                            .tint(c.score * 2 >= c.max ? OpaleTheme.gain : OpaleTheme.loss)
                        Text(c.comment)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                    .padding(.vertical, 4)
                }
            }
            .navigationTitle("Santé financière — \(health.score)/100")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("OK") { dismiss() }
                }
            }
        }
    }
}
