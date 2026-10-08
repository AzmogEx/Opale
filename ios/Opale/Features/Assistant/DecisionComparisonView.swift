import SwiftUI

struct DecisionAlternative: Decodable {
    let label: String
    let initial_outflow_cents: Cents
    let monthly_cost_cents: Cents
    let final_liquid_cents: Cents
    let asset_value_cents: Cents
    let remaining_debt_cents: Cents
    let total_interest_cents: Cents
    let total_fees_cents: Cents
    let final_net_cents: Cents
    let real_final_net_cents: Cents
    let minimum_liquid_cents: Cents
    let feasible: Bool
}
struct DecisionComparisonResult: Decodable {
    let a: DecisionAlternative
    let b: DecisionAlternative
    let delta_b_minus_a_cents: Cents
    let assumptions: [String]
    struct Sensitivity: Decodable, Hashable { let investment_return_bps: Int; let delta_b_minus_a_cents: Cents }
    let sensitivity: [Sensitivity]
    struct Impact: Decodable { let months: Int; let a: DecisionAlternative; let b: DecisionAlternative; let delta_b_minus_a_cents: Cents }
    struct Scenario: Decodable { let name: String; let investment_return_bps: Int; let a: DecisionAlternative; let b: DecisionAlternative; let delta_b_minus_a_cents: Cents }
    struct Recommendation: Decodable { let preferred: String; let message: String }
    let timeline: [Impact]?
    let scenarios: [Scenario]?
    let recommendation: Recommendation?
    let risks: [String]?
}
struct DecisionComparisonView: View {
    @Environment(SessionStore.self) private var session
    @State private var kind = "buy_rent"
    @State private var months = 120
    @State private var loanMonths = 240
    @State private var values: [String: String] = [:]
    @State private var rates: [String: Int] = ["loan_rate_bps": 350, "investment_return_bps": 400, "asset_growth_bps": 100, "rent_growth_bps": 100, "inflation_bps": 200, "sale_fee_bps": 300]
    @State private var result: DecisionComparisonResult?
    @State private var error: String?
    @State private var busy = false
    private let common = [("initial_cash_cents", "Capital disponible"), ("monthly_budget_cents", "Budget mensuel avant logement / crédit")]
    private var fields: [(String, String)] {
        switch kind {
        case "buy_rent": common + [("asset_price_cents", "Prix du bien"), ("down_payment_cents", "Apport"), ("purchase_fees_cents", "Frais d'achat"), ("monthly_rent_cents", "Loyer alternatif"), ("monthly_ownership_cost_cents", "Charges propriétaire / mois")]
        case "cash_credit": common + [("asset_price_cents", "Prix de l'achat"), ("down_payment_cents", "Apport crédit"), ("purchase_fees_cents", "Frais d'achat")]
        default: common + [("loan_principal_cents", "Capital restant dû"), ("repayment_cents", "Remboursement envisagé"), ("repayment_fee_cents", "Frais de remboursement")]
        }
    }
    var body: some View {
        Form {
            Section("Alternatives") {
                Picker("Comparaison", selection: $kind) { Text("Acheter / louer").tag("buy_rent"); Text("Cash / crédit").tag("cash_credit"); Text("Rembourser / investir").tag("repay_invest") }
                Stepper("Horizon : \(months) mois", value: $months, in: 12...600, step: 12)
            }
            Section("Montants en euros") {
                ForEach(fields, id: \.0) { field in TextField(field.1, text: Binding(get: { values[field.0, default: ""] }, set: { values[field.0] = $0 })).keyboardType(.decimalPad) }
            }
            Section("Crédit & hypothèses") {
                Stepper("Durée du crédit : \(loanMonths) mois", value: $loanMonths, in: 12...600, step: 12)
                rate("loan_rate_bps", "Taux du crédit", 0...2000)
                rate("investment_return_bps", "Rendement investissement", -5000...2000)
                rate("inflation_bps", "Inflation", 0...2000)
                if kind != "repay_invest" { rate("asset_growth_bps", "Variation annuelle du bien", -5000...2000); rate("sale_fee_bps", "Frais de revente", 0...10000) }
                if kind == "buy_rent" { rate("rent_growth_bps", "Évolution du loyer", -5000...2000) }
                Text("Les rendements sont des hypothèses. Compare les sensibilités ; aucune performance future n'est garantie.").font(.caption)
                Button(busy ? "Calcul…" : "Comparer") { Task { await compare() } }.disabled(busy)
            }
            if let result {
                alternative(result.a)
                alternative(result.b)
                Section("Écart B − A à l'horizon") { AmountText(cents: result.delta_b_minus_a_cents, style: .signedDelta) }
                if let recommendation = result.recommendation {
                    Section("Orientation conditionnelle") { Text(recommendation.message).sensitive() }
                }
                if let risks = result.risks, !risks.isEmpty {
                    Section("Risques et points de vigilance") { ForEach(risks, id: \.self) { Text($0).sensitive() } }
                }
                if let timeline = result.timeline {
                    Section("Impacts à 0, 5 et 10 ans") {
                        ForEach(timeline, id: \.months) { impact in
                            DisclosureGroup(impact.months == 0 ? "Au départ" : "À \(impact.months / 12) ans") {
                                comparisonSummary(a: impact.a, b: impact.b, delta: impact.delta_b_minus_a_cents)
                            }
                        }
                    }
                }
                if let scenarios = result.scenarios {
                    Section("Trois scénarios à l’horizon choisi") {
                        ForEach(scenarios, id: \.name) { scenario in
                            DisclosureGroup(scenario.name.capitalized) {
                                Text("Rendement hypothétique : \((Decimal(scenario.investment_return_bps) / 100).formatted(.number.precision(.fractionLength(0...2)))) %").font(.caption).sensitive()
                                comparisonSummary(a: scenario.a, b: scenario.b, delta: scenario.delta_b_minus_a_cents)
                            }
                        }
                    }
                }
                Section("Conventions") { ForEach(result.assumptions, id: \.self) { Text($0).font(.caption).sensitive() } }
                Section("Sensibilités") { ForEach(result.sensitivity, id: \.self) { item in LabeledContent("Rendement \(String(describing: Decimal(item.investment_return_bps) / 100)) %") { AmountText(cents: item.delta_b_minus_a_cents, style: .signedDelta) } } }
            }
            ToolError(message: error)
        }.navigationTitle("Comparer une décision")
        .onChange(of: kind) { result = nil }
    }
    private func rate(_ key: String, _ title: String, _ range: ClosedRange<Int>) -> some View {
        Stepper("\(title) : \(String(describing: Decimal(rates[key, default: 0]) / 100)) %", value: Binding(get: { rates[key, default: 0] }, set: { rates[key] = $0 }), in: range, step: 25)
    }
    @ViewBuilder private func comparisonSummary(a: DecisionAlternative, b: DecisionAlternative, delta: Cents) -> some View {
        LabeledContent("Patrimoine A · " + a.label) { AmountText(cents: a.final_net_cents) }
        LabeledContent("Patrimoine B · " + b.label) { AmountText(cents: b.final_net_cents) }
        LabeledContent("Écart B − A") { AmountText(cents: delta, style: .signedDelta) }
        LabeledContent("A en euros constants") { AmountText(cents: a.real_final_net_cents) }
        LabeledContent("B en euros constants") { AmountText(cents: b.real_final_net_cents) }
        LabeledContent("Liquidités A") { AmountText(cents: a.final_liquid_cents) }
        LabeledContent("Liquidités B") { AmountText(cents: b.final_liquid_cents) }
        if !a.feasible || !b.feasible { Text("Trésorerie insuffisante pour \(!a.feasible ? a.label : b.label) dans ces hypothèses").font(.caption).foregroundStyle(.orange) }
    }
    private func alternative(_ a: DecisionAlternative) -> some View {
        Section(a.label) {
            Label(a.feasible ? "Budget compatible avec les hypothèses" : "Trésorerie insuffisante", systemImage: a.feasible ? "checkmark.circle" : "exclamationmark.triangle").foregroundStyle(a.feasible ? Color.green : .red)
            LabeledContent("Effort initial") { AmountText(cents: a.initial_outflow_cents) }
            LabeledContent("Coût mensuel") { AmountText(cents: a.monthly_cost_cents) }
            LabeledContent("Patrimoine final nominal") { AmountText(cents: a.final_net_cents) }
            LabeledContent("En euros constants") { AmountText(cents: a.real_final_net_cents) }
            LabeledContent("Liquidités finales") { AmountText(cents: a.final_liquid_cents) }
            LabeledContent("Bien à la fin") { AmountText(cents: a.asset_value_cents) }
            LabeledContent("Dette restante") { AmountText(cents: a.remaining_debt_cents) }
            LabeledContent("Intérêts") { AmountText(cents: a.total_interest_cents) }
            LabeledContent("Frais") { AmountText(cents: a.total_fees_cents) }
            LabeledContent("Trésorerie minimum") { AmountText(cents: a.minimum_liquid_cents) }
        }
    }
    private func compare() async {
        result = nil
        var numbers: [String: Int64] = [:]
        for (key, _) in fields { guard let amount = Cents.parse(values[key, default: "0"].isEmpty ? "0" : values[key]!) , amount.raw >= 0 else { error = "Montants positifs valides requis"; return }; numbers[key] = amount.raw }
        for (key, value) in rates { numbers[key] = Int64(value) }
        numbers["horizon_months"] = Int64(months); numbers["loan_months"] = Int64(loanMonths)
        let body = DecisionInput(kind: kind, numbers: numbers)
        busy = true; defer { busy = false }
        do { result = try await session.api.request("POST", "/v1/decisions/compare", body: body); error = nil } catch { self.error = error.localizedDescription }
    }
}
private struct DecisionInput: Encodable {
    let kind: String
    let numbers: [String: Int64]
    struct Key: CodingKey { var stringValue: String; var intValue: Int? { nil }; init(stringValue: String) { self.stringValue = stringValue }; init?(intValue: Int) { nil } }
    func encode(to encoder: Encoder) throws { var c = encoder.container(keyedBy: Key.self); try c.encode(kind, forKey: Key(stringValue: "kind")); for (key,value) in numbers { try c.encode(value, forKey: Key(stringValue: key)) } }
}
