import SwiftUI
import Charts

/// Onglet Projection — le futur du patrimoine (EF-040/041).
///
/// « Si j'épargne X €/mois → où j'atterris, et quand suis-je libre ? »
/// Tous les chiffres viennent du moteur déterministe du backend (EIA-040) ;
/// cette vue ne fait que régler les hypothèses et afficher.
struct ProjectionView: View {
    @Environment(SessionStore.self) private var session

    // Hypothèses persistées sur l'appareil (en euros entiers / bps).
    @AppStorage("projection.savingsEuros") private var savingsEuros = 500
    @AppStorage("projection.returnBps") private var returnBps = 500
    @AppStorage("projection.expensesEuros") private var expensesEuros = 2000

    @State private var result: ProjectionResponse?
    @State private var errorMessage: String?
    @State private var showComparison = false

    var body: some View {
        NavigationStack {
            ZStack {
                OpaleTheme.iridescent
                    .opacity(0.14)
                    .ignoresSafeArea()

                ScrollView {
                    GlassEffectContainer(spacing: 16) {
                        VStack(spacing: 16) {
                            if let result {
                                independenceCard(result)
                                    .cascadeIn(0)
                                projectionChartCard(result)
                                    .cascadeIn(1)
                            } else if let errorMessage {
                                ContentUnavailableView(
                                    "Impossible de projeter",
                                    systemImage: "bolt.horizontal.circle",
                                    description: Text(errorMessage)
                                )
                            } else {
                                ProgressView()
                                    .frame(minHeight: 200)
                            }
                            assumptionsCard
                                .cascadeIn(2)
                            GoalsSection()
                                .cascadeIn(3)
                        }
                        .padding(.horizontal)
                        .padding(.bottom, 24)
                    }
                }
                .scrollEdgeEffectStyle(.soft, for: .top)
            }
            .navigationTitle("Projection")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button {
                        showComparison = true
                    } label: {
                        Label("Comparer", systemImage: "arrow.triangle.branch")
                    }
                }
            }
            .sheet(isPresented: $showComparison) {
                ComparisonView()
            }
            // Recharge à chaque changement d'hypothèse (annule la requête
            // précédente automatiquement).
            .task(id: "\(savingsEuros)-\(returnBps)-\(expensesEuros)") {
                await load()
            }
        }
    }

    // MARK: - Héro : la date de liberté (EF-040)

    @ViewBuilder
    private func independenceCard(_ result: ProjectionResponse) -> some View {
        let independence = result.independence
        GlassCard {
            VStack(alignment: .leading, spacing: 8) {
                Text("Indépendance financière")
                    .font(.footnote.weight(.semibold))
                    .foregroundStyle(.secondary)
                    .textCase(.uppercase)

                if independence.reached {
                    let years = independence.months / 12
                    let freedomYear = Calendar.current.component(.year, from: .now)
                        + years
                    Text(independence.months == 0 ? "Déjà libre 🎉" : "Libre en \(String(freedomYear))")
                        .font(.system(size: 40, weight: .bold, design: .rounded))
                        .foregroundStyle(OpaleTheme.iridescent)
                        .iridescentShimmer()
                        .contentTransition(.numericText())
                        .animation(.spring(duration: 0.5), value: independence.months)

                    if independence.months > 0 {
                        Text(durationLabel(months: independence.months))
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                    }
                } else {
                    Text("Hors d'atteinte")
                        .font(.system(size: 34, weight: .bold, design: .rounded))
                        .foregroundStyle(.secondary)
                    Text("À ce rythme, la cible n'est pas atteinte en 100 ans — augmente l'épargne ou réduis les dépenses.")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }

                HStack(spacing: 6) {
                    Image(systemName: "target")
                        .font(.caption)
                        .foregroundStyle(OpaleTheme.accent)
                    Text("Cible :")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    AmountText(cents: independence.target, style: .whole)
                        .font(.caption.weight(.semibold))
                    Text("(retrait \(formattedPercent(result.swrBps)))")
                        .font(.caption)
                        .foregroundStyle(.tertiary)
                }
            }
        }
    }

    private func durationLabel(months: Int) -> String {
        let years = months / 12
        let rest = months % 12
        if years == 0 { return "dans \(rest) mois" }
        if rest == 0 { return "dans \(years) ans" }
        return "dans \(years) ans et \(rest) mois"
    }

    // MARK: - Courbe projetée (EF-041)

    private func projectionChartCard(_ result: ProjectionResponse) -> some View {
        GlassCard {
            VStack(alignment: .leading, spacing: 12) {
                Text("Trajectoire — 30 ans")
                    .font(.footnote.weight(.semibold))
                    .foregroundStyle(.secondary)
                    .textCase(.uppercase)

                Chart {
                    ForEach(result.points) { point in
                        AreaMark(
                            x: .value("Année", Double(point.month) / 12),
                            y: .value("Patrimoine", point.net.chartValue)
                        )
                        .interpolationMethod(.monotone)
                        .foregroundStyle(
                            .linearGradient(
                                colors: [OpaleTheme.accent.opacity(0.3), .clear],
                                startPoint: .top, endPoint: .bottom
                            )
                        )

                        LineMark(
                            x: .value("Année", Double(point.month) / 12),
                            y: .value("Patrimoine", point.net.chartValue)
                        )
                        .interpolationMethod(.monotone)
                        .lineStyle(StrokeStyle(lineWidth: 3, lineCap: .round))
                        .foregroundStyle(OpaleTheme.accent)
                    }

                    // Ligne cible : le patrimoine d'indépendance.
                    if result.independence.target > .zero {
                        RuleMark(y: .value("Cible", result.independence.target.chartValue))
                            .foregroundStyle(OpaleTheme.gain.opacity(0.8))
                            .lineStyle(StrokeStyle(lineWidth: 1.5, dash: [6, 4]))
                            .annotation(position: .top, alignment: .leading) {
                                Text("Liberté")
                                    .font(.caption2.weight(.semibold))
                                    .foregroundStyle(OpaleTheme.gain)
                            }
                    }

                    // Le moment de la liberté.
                    if result.independence.reached, result.independence.months > 0,
                       let point = result.points.first(where: { $0.month >= result.independence.months }) {
                        PointMark(
                            x: .value("Année", Double(point.month) / 12),
                            y: .value("Patrimoine", point.net.chartValue)
                        )
                        .symbolSize(140)
                        .foregroundStyle(OpaleTheme.gain)
                    }
                }
                .chartXAxis {
                    AxisMarks(values: .stride(by: 5)) { value in
                        AxisGridLine()
                        AxisValueLabel {
                            if let years = value.as(Double.self) {
                                Text("\(Int(years)) ans")
                            }
                        }
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
                .frame(minHeight: 200, maxHeight: 260)
            }
        }
    }

    private static func compactEuros(_ euros: Double) -> String {
        if abs(euros) >= 1_000_000 { return String(format: "%.1f M€", euros / 1_000_000) }
        if abs(euros) >= 1_000 { return String(format: "%.0f k€", euros / 1_000) }
        return String(format: "%.0f €", euros)
    }

    // MARK: - Hypothèses (curseurs)

    private var assumptionsCard: some View {
        GlassCard {
            VStack(alignment: .leading, spacing: 20) {
                Text("Hypothèses")
                    .font(.footnote.weight(.semibold))
                    .foregroundStyle(.secondary)
                    .textCase(.uppercase)

                assumptionSlider(
                    label: "Épargne mensuelle",
                    systemImage: "arrow.down.circle.fill",
                    value: $savingsEuros,
                    range: 0...5000, step: 50,
                    format: { "\($0) €" }
                )
                assumptionSlider(
                    label: "Rendement annuel",
                    systemImage: "percent",
                    value: $returnBps,
                    range: 0...1200, step: 50,
                    format: { formattedPercent($0) }
                )
                assumptionSlider(
                    label: "Dépenses mensuelles",
                    systemImage: "cart.fill",
                    value: $expensesEuros,
                    range: 500...10000, step: 100,
                    format: { "\($0) €" }
                )
            }
        }
    }

    @ViewBuilder
    private func assumptionSlider(
        label: String,
        systemImage: String,
        value: Binding<Int>,
        range: ClosedRange<Int>,
        step: Int,
        format: @escaping (Int) -> String
    ) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack {
                Label(label, systemImage: systemImage)
                    .font(.subheadline.weight(.medium))
                Spacer()
                Text(format(value.wrappedValue))
                    .font(.subheadline.weight(.bold))
                    .foregroundStyle(OpaleTheme.accent)
                    .contentTransition(.numericText())
                    .animation(.snappy, value: value.wrappedValue)
                    .monospacedDigit()
            }
            Slider(
                value: Binding(
                    get: { Double(value.wrappedValue) },
                    set: { value.wrappedValue = Int(($0 / Double(step)).rounded()) * step }
                ),
                in: Double(range.lowerBound)...Double(range.upperBound)
            )
            .sensoryFeedback(.selection, trigger: value.wrappedValue)
        }
    }

    private func formattedPercent(_ bps: Int) -> String {
        let whole = bps / 100
        let frac = (bps % 100) / 10
        return frac == 0 ? "\(whole) %" : "\(whole),\(frac) %"
    }

    // MARK: - Chargement

    private func load() async {
        do {
            result = try await session.api.projection(
                monthlySavingsCents: Int64(savingsEuros) * 100,
                annualReturnBps: returnBps,
                monthlyExpensesCents: Int64(expensesEuros) * 100
            )
            errorMessage = nil
        } catch is CancellationError {
            // Curseur encore en mouvement — la requête suivante arrive.
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
