import SwiftUI
import Charts

/// Comparateur de scénarios (EF-044) : deux futurs côte à côte, projetés
/// par le moteur depuis le patrimoine réel. « Rester locataire vs acheter »,
/// « épargner plus vs profiter »… les chiffres tranchent.
struct ComparisonView: View {
    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss

    // Scénario A prérempli avec les hypothèses de l'onglet Projection.
    @AppStorage("projection.savingsEuros") private var baseSavings = 500
    @AppStorage("projection.expensesEuros") private var baseExpenses = 2000
    @AppStorage("projection.returnBps") private var baseReturn = 500

    @State private var a = ScenarioForm(label: "Aujourd'hui")
    @State private var b = ScenarioForm(label: "Alternative")
    @State private var result: ScenarioComparison?
    @State private var isLoading = false
    @State private var errorMessage: String?
    // Comparaisons enregistrées (EF-044+) — persistées sur l'appareil.
    @State private var saved: [SavedComparison] = SavedComparison.all()
    @State private var askName = false
    @State private var newName = ""

    struct SavedComparison: Codable, Identifiable {
        var id = UUID()
        var name: String
        var a: ScenarioForm
        var b: ScenarioForm

        static let key = "comparisons.saved"
        static func all() -> [SavedComparison] {
            guard let data = UserDefaults.standard.data(forKey: key),
                  let list = try? JSONDecoder().decode([SavedComparison].self, from: data)
            else { return [] }
            return list
        }
        static func persist(_ list: [SavedComparison]) {
            if let data = try? JSONEncoder().encode(list) {
                UserDefaults.standard.set(data, forKey: key)
            }
        }
    }

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: 16) {
                    HStack(alignment: .top, spacing: 12) {
                        scenarioCard($a, tint: OpaleTheme.accent)
                        scenarioCard($b, tint: .orange)
                    }

                    Button {
                        Task { await compare() }
                    } label: {
                        if isLoading {
                            ProgressView().frame(maxWidth: .infinity)
                        } else {
                            Label("Comparer ces deux futurs", systemImage: "arrow.triangle.branch")
                                .frame(maxWidth: .infinity)
                        }
                    }
                    .buttonStyle(.glassProminent)
                    .disabled(isLoading)

                    if let errorMessage {
                        Text(errorMessage).foregroundStyle(OpaleTheme.loss)
                    }

                    if let result {
                        verdictCard(result)
                        chartCard(result)
                    }
                }
                .padding()
            }
            .background(OpaleTheme.iridescent.opacity(0.14).ignoresSafeArea())
            .navigationTitle("Comparateur")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Fermer") { dismiss() }
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Menu {
                        Button {
                            newName = "\(a.label) vs \(b.label)"
                            askName = true
                        } label: {
                            Label("Enregistrer cette comparaison", systemImage: "bookmark")
                        }
                        if !saved.isEmpty {
                            Section("Enregistrées") {
                                ForEach(saved) { comparison in
                                    Button(comparison.name) {
                                        a = comparison.a
                                        b = comparison.b
                                        Task { await compare() }
                                    }
                                }
                            }
                            Button(role: .destructive) {
                                saved = []
                                SavedComparison.persist([])
                            } label: {
                                Label("Tout effacer", systemImage: "trash")
                            }
                        }
                    } label: {
                        Image(systemName: "bookmark")
                    }
                }
            }
            .alert("Nom de la comparaison", isPresented: $askName) {
                TextField("Ex. Louer vs acheter", text: $newName)
                Button("Enregistrer") {
                    saved.append(SavedComparison(name: newName, a: a, b: b))
                    SavedComparison.persist(saved)
                }
                Button("Annuler", role: .cancel) {}
            }
            .onAppear {
                a.savings = baseSavings
                a.expenses = baseExpenses
                b.savings = baseSavings
                b.expenses = baseExpenses
            }
        }
    }

    // MARK: - Saisie d'un scénario

    @ViewBuilder
    private func scenarioCard(_ form: Binding<ScenarioForm>, tint: Color) -> some View {
        GlassCard {
            VStack(alignment: .leading, spacing: 10) {
                TextField("Nom", text: form.label)
                    .font(.subheadline.weight(.bold))
                    .foregroundStyle(tint)
                stepperRow("Épargne/mois", value: form.savings, step: 100, suffix: "€")
                stepperRow("Dépenses/mois", value: form.expenses, step: 100, suffix: "€")
                stepperRow("Coût immédiat", value: form.oneTime, step: 5000, suffix: "€")
            }
        }
    }

    @ViewBuilder
    private func stepperRow(_ title: String, value: Binding<Int>, step: Int, suffix: String) -> some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(title).font(.caption2).foregroundStyle(.secondary)
            HStack(spacing: 6) {
                Text("\(value.wrappedValue) \(suffix)")
                    .font(.callout.weight(.semibold))
                    .monospacedDigit()
                    .contentTransition(.numericText())
                    .animation(.snappy, value: value.wrappedValue)
                Spacer()
                Stepper("", value: value, step: step)
                    .labelsHidden()
                    .scaleEffect(0.8)
            }
        }
    }

    // MARK: - Verdict

    @ViewBuilder
    private func verdictCard(_ r: ScenarioComparison) -> some View {
        GlassCard {
            VStack(alignment: .leading, spacing: 10) {
                Text("Verdict — « \(r.b.label) » vs « \(r.a.label) »")
                    .font(.footnote.weight(.semibold))
                    .foregroundStyle(.secondary)
                    .textCase(.uppercase)

                HStack {
                    deltaStat("À 5 ans", r.delta5y)
                    Divider()
                    deltaStat("À 10 ans", r.delta10y)
                    Divider()
                    deltaStat("À 20 ans", r.deltaEnd)
                }

                if r.a.independence.reached || r.b.independence.reached {
                    HStack(spacing: 12) {
                        independenceLabel(r.a, tint: OpaleTheme.accent)
                        independenceLabel(r.b, tint: .orange)
                    }
                }
            }
        }
    }

    private func deltaStat(_ title: String, _ delta: Cents) -> some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(title).font(.caption2).foregroundStyle(.secondary)
            AmountText(cents: delta, style: .signedDelta)
                .font(.subheadline.weight(.bold))
                .foregroundStyle(delta.raw < 0 ? OpaleTheme.loss : OpaleTheme.gain)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    @ViewBuilder
    private func independenceLabel(_ s: ScenarioResult, tint: Color) -> some View {
        let text: String = if s.independence.reached {
            s.independence.months == 0 ? "déjà libre" : "libre dans \(s.independence.months / 12) ans"
        } else {
            "liberté hors d'atteinte"
        }
        Label("\(s.label) : \(text)", systemImage: "bird")
            .font(.caption)
            .foregroundStyle(tint)
    }

    // MARK: - Courbes superposées

    @ViewBuilder
    private func chartCard(_ r: ScenarioComparison) -> some View {
        GlassCard {
            VStack(alignment: .leading, spacing: 10) {
                Text("Deux trajectoires — \(r.months / 12) ans")
                    .font(.footnote.weight(.semibold))
                    .foregroundStyle(.secondary)
                    .textCase(.uppercase)

                Chart {
                    ForEach(r.a.points) { point in
                        LineMark(
                            x: .value("Année", Double(point.month) / 12),
                            y: .value("Patrimoine", point.net.chartValue),
                            series: .value("Scénario", r.a.label)
                        )
                        .interpolationMethod(.monotone)
                        .lineStyle(StrokeStyle(lineWidth: 3, lineCap: .round))
                        .foregroundStyle(by: .value("Scénario", r.a.label))
                    }
                    ForEach(r.b.points) { point in
                        LineMark(
                            x: .value("Année", Double(point.month) / 12),
                            y: .value("Patrimoine", point.net.chartValue),
                            series: .value("Scénario", r.b.label)
                        )
                        .interpolationMethod(.monotone)
                        .lineStyle(StrokeStyle(lineWidth: 3, lineCap: .round))
                        .foregroundStyle(by: .value("Scénario", r.b.label))
                    }
                }
                .chartForegroundStyleScale([
                    r.a.label: OpaleTheme.accent,
                    r.b.label: Color.orange,
                ])
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
                .chartLegend(position: .bottom)
                .frame(minHeight: 220, maxHeight: 280)
            }
        }
    }

    // MARK: - Appel moteur

    private func compare() async {
        isLoading = true
        errorMessage = nil
        defer { isLoading = false }
        do {
            result = try await session.api.compareScenarios(
                a: a.params(returnBps: baseReturn),
                b: b.params(returnBps: baseReturn)
            )
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}

/// Les hypothèses saisies pour un scénario. Codable : les comparaisons
/// peuvent être ENREGISTRÉES et rechargées (local, par appareil).
struct ScenarioForm: Codable {
    var label: String
    var savings = 500
    var expenses = 2000
    var oneTime = 0

    func params(returnBps: Int) -> APIClient.ScenarioParams {
        .init(
            label: label,
            monthlySavingsCents: Int64(savings) * 100,
            monthlyExpensesCents: Int64(expenses) * 100,
            annualReturnBps: returnBps,
            oneTimeCostCents: Int64(oneTime) * 100
        )
    }
}
