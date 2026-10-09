import Foundation

enum JourneyStep: Int, CaseIterable, Identifiable, Codable, Sendable {
    case accounts, income, expenses, subscriptions, wealth, budget, review
    var id: String { key }
    var key: String { switch self { case .accounts: "accounts"; case .income: "income"; case .expenses: "expenses"; case .subscriptions: "subscriptions"; case .wealth: "wealth"; case .budget: "budget"; case .review: "review" } }
    var title: String { switch self {
    case .accounts: "Mes comptes et leurs soldes"
    case .income: "Ce que je gagne"
    case .expenses: "Mes charges fixes"
    case .subscriptions: "Mes abonnements"
    case .wealth: "Mes biens et mes crédits"
    case .budget: "Mon budget et mes projets"
    case .review: "Comprendre mon bilan"
    } }
    var icon: String { switch self { case .accounts: "building.columns"; case .income: "arrow.down.circle"; case .expenses: "house"; case .subscriptions: "repeat"; case .wealth: "square.stack.3d.up"; case .budget: "target"; case .review: "lightbulb" } }
    var introduction: String { switch self {
    case .accounts: "Ajoute d’abord le compte que tu utilises au quotidien, puis tes livrets. Recopie le solde et la date de ton relevé."
    case .income: "Indique ce qui entre régulièrement : salaire net versé, pension, missions, primes ou loyers."
    case .expenses: "Ajoute ce qui revient presque tous les mois : logement, énergie, assurance, transport ou remboursement de crédit."
    case .subscriptions: "Rassemble ici streaming, sport, téléphone et logiciels. Vérifie ce qui se renouvelle et les dates pour arrêter."
    case .wealth: "Ajoute tes autres biens et placements, puis le capital de tes crédits restant à rembourser."
    case .budget: "Prévois une somme pour les courses, les sorties et le quotidien, puis choisis un premier projet."
    case .review: "Relisons les informations actuelles pour savoir ce qu’elles veulent dire et quoi faire ensuite."
    } }
    var explanation: String { switch self {
    case .accounts: "Un solde est une photo de ton compte à une date précise. Ce n’est pas un revenu : tu ne dois pas le saisir une seconde fois comme un salaire. Une valeur inconnue reste inconnue."
    case .income: "Le salaire à recopier est le net effectivement payé après impôt. Pour les revenus irréguliers, le minimum déclaré aide à préparer un mois prudent. Une prévision n’est jamais un encaissement réalisé."
    case .expenses: "Une facture annuelle est répartie sur les mois pour comprendre son poids. Son paiement reste prévu à la vraie date dans le calendrier. Une mensualité de crédit est une sortie d’argent ; elle ne remplace pas le capital restant dû."
    case .subscriptions: "Un abonnement peu utilisé peut peser sur l’année. Un contrat arrêté dans Opale n’est pas résilié chez le fournisseur. Une hausse détectée doit être vérifiée avant de changer le tarif."
    case .wealth: "Le patrimoine net correspond aux biens valorisés, moins les dettes valorisées. Ce n’est pas l’argent disponible sur ton compte. Saisis le bien et son crédit séparément, sans déduire la dette de la valeur du bien."
    case .budget: "Le budget quotidien complète les charges et abonnements déjà saisis : évite de les inclure une deuxième fois. Un objectif donne un cap ; l’argent que tu lui réserves doit rester compatible avec tes dépenses."
    case .review: "Le budget déclaré décrit un mois type. Le solde et le patrimoine viennent des valeurs et mouvements enregistrés. Les dépenses réelles peuvent différer : compare-les régulièrement, avant de prendre une décision."
    } }
}

struct FinancialJourneyState: Codable, Equatable, Sendable {
    var revision: Int64 = 0
    var step = 0
    var reviewed: [String] = []
    var skipped: [String] = []
    var dailyBudget: Cents?
    var budgetCurrency = "EUR"
    enum CodingKeys: String, CodingKey { case revision, step, reviewed, skipped; case dailyBudget = "daily_budget_cents", budgetCurrency = "budget_currency" }
    var finished: Bool { JourneyStep.allCases.allSatisfy { reviewed.contains($0.key) || skipped.contains($0.key) } }
    var completedCount: Int { JourneyStep.allCases.filter { reviewed.contains($0.key) || skipped.contains($0.key) }.count }
    var next: JourneyStep { JourneyStep.allCases.first { !reviewed.contains($0.key) && !skipped.contains($0.key) } ?? .review }
    mutating func mark(_ step: JourneyStep, skipped skip: Bool) {
        reviewed.removeAll { $0 == step.key }; skipped.removeAll { $0 == step.key }
        if skip { skipped.append(step.key) } else { reviewed.append(step.key) }
        self.step = next.rawValue
    }
}

struct FinancialJourneySnapshot: Codable, Sendable {
    var state: FinancialJourneyState
    var assets: [Asset]
    var liabilities: [Liability]
    var rules: [CalendarRule]
    var contracts: [FinancialContract]
    var incomes: [VariableIncome]
    var goals: [GoalStatus]
    var netWorth: NetWorth
    var alerts: [ContractAlert]
    var loadedAt = Date.now
    var accounts: [Asset] { assets.filter { !$0.archived && [.checking, .savings].contains($0.kind) } }
    var otherAssets: [Asset] { assets.filter { !$0.archived && ![.checking, .savings].contains($0.kind) } }
    nonisolated static func annualMultiplier(_ frequency: String) -> Decimal { frequency == "weekly" ? 52 : FinancialTools.annualMultiplier(frequency) }
    var ongoingRules: [CalendarRule] { rules.filter { $0.active && ($0.end_date == nil || $0.end_date! >= Date.now.opaleDayString) && ($0.managed_by ?? "").isEmpty && Self.annualMultiplier($0.frequency) > 0 } }
    var currencyRows: [JourneyMonthlyBudget] {
        let accountCurrencies = Dictionary(uniqueKeysWithValues: assets.map { ($0.id, $0.currency) })
        let currencies = Set(contracts.filter(\.active).map(\.currency) + incomes.filter(\.active).map(\.currency) + ongoingRules.compactMap { accountCurrencies[$0.asset_id] } + [state.budgetCurrency])
        return currencies.sorted().map { currency in
            let manual = ongoingRules.filter { accountCurrencies[$0.asset_id] == currency }
            func annual(_ values: [(Decimal, String)]) -> Decimal { values.reduce(.zero) { $0 + $1.0 * Self.annualMultiplier($1.1) } }
            let income = annual(manual.filter { $0.amount_cents > .zero }.map { (Decimal($0.amount_cents.raw), $0.frequency) }) + annual(incomes.filter { $0.active && $0.currency == currency && $0.forecast != "off" }.map { (Decimal(($0.forecast == "usual" ? $0.usual : $0.low).raw), $0.frequency) })
            let charges = annual(manual.filter { $0.amount_cents < .zero }.map { (-Decimal($0.amount_cents.raw), $0.frequency) }) + annual(contracts.filter { $0.active && $0.currency == currency && $0.category != "subscription" }.map { (Decimal($0.amount.raw), $0.frequency) })
            let subscriptions = annual(contracts.filter { $0.active && $0.currency == currency && $0.category == "subscription" }.map { (Decimal($0.amount.raw), $0.frequency) })
            let budget = state.budgetCurrency == currency ? state.dailyBudget : nil
            let ready = ["income", "expenses", "subscriptions", "budget"].allSatisfy { state.reviewed.contains($0) }
            let remaining = ready ? budget.flatMap { FinancialSetupSummary.roundedCents((income - charges - subscriptions) / 12 - Decimal($0.raw)) } : nil
            return JourneyMonthlyBudget(currency: currency, income: FinancialSetupSummary.roundedCents(income / 12), charges: FinancialSetupSummary.roundedCents(charges / 12), subscriptions: FinancialSetupSummary.roundedCents(subscriptions / 12), daily: budget, remaining: remaining)
        }
    }
}
struct JourneyMonthlyBudget: Identifiable, Sendable {
    var currency: String; var income: Cents?; var charges: Cents?; var subscriptions: Cents?; var daily: Cents?; var remaining: Cents?
    var id: String { currency }
}

extension APIClient {
    func financialJourney() async throws -> FinancialJourneyState { try await request("GET", "/v1/journey") }
    func saveFinancialJourney(_ state: FinancialJourneyState) async throws -> FinancialJourneyState { try await request("PUT", "/v1/journey", body: state) }
    func journeySnapshot() async throws -> FinancialJourneySnapshot {
        async let state = financialJourney()
        async let assets = listAssets()
        async let liabilities = listLiabilities()
        async let contracts = contracts()
        async let incomes = variableIncomes()
        async let goals = listGoals()
        async let net = netWorth()
        async let calendar: CalendarData = request("GET", "/v1/calendar", query: [URLQueryItem(name: "from", value: Date.now.opaleDayString), URLQueryItem(name: "until", value: Date.now.opaleDayString)])
        return try await FinancialJourneySnapshot(state: state, assets: assets, liabilities: liabilities, rules: calendar.rules, contracts: contracts.contracts, incomes: incomes, goals: goals, netWorth: net, alerts: contracts.alerts)
    }
}
