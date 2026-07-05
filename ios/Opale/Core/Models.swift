import Foundation

// Modèles miroirs de l'API Opale (backend Go). Les clés JSON sont en
// snake_case — chaque type déclare ses CodingKeys explicitement.

// MARK: - Profil & session

struct Profile: Identifiable, Codable, Hashable, Sendable {
    let id: String
    var name: String
    var privacyDefault: String

    enum CodingKeys: String, CodingKey {
        case id, name
        case privacyDefault = "privacy_default"
    }
}

struct LoginResponse: Codable, Sendable {
    let token: String
    let profile: Profile

    enum CodingKeys: String, CodingKey { case token, profile }
}

// MARK: - Patrimoine

enum AssetKind: String, Codable, CaseIterable, Identifiable, Sendable {
    case checking, savings
    case lifeInsurance = "life_insurance"
    case pea, cto, crypto
    case realEstate = "real_estate"
    case preciousMetal = "precious_metal"
    case vehicle, valuable
    case companyShare = "company_share"
    case other

    var id: String { rawValue }

    /// Libellé français.
    var label: String {
        switch self {
        case .checking: "Compte courant"
        case .savings: "Livret / épargne"
        case .lifeInsurance: "Assurance-vie"
        case .pea: "PEA"
        case .cto: "Compte-titres"
        case .crypto: "Crypto"
        case .realEstate: "Immobilier"
        case .preciousMetal: "Or / métaux"
        case .vehicle: "Véhicule"
        case .valuable: "Objet de valeur"
        case .companyShare: "Parts de société"
        case .other: "Autre"
        }
    }

    var systemImage: String {
        switch self {
        case .checking: "creditcard"
        case .savings: "banknote"
        case .lifeInsurance: "shield.lefthalf.filled"
        case .pea, .cto: "chart.pie"
        case .crypto: "bitcoinsign.circle"
        case .realEstate: "house"
        case .preciousMetal: "sparkle"
        case .vehicle: "car"
        case .valuable: "crown"
        case .companyShare: "briefcase"
        case .other: "square.grid.2x2"
        }
    }
}

enum LiabilityKind: String, Codable, CaseIterable, Identifiable, Sendable {
    case mortgage
    case autoLoan = "auto_loan"
    case consumerLoan = "consumer_loan"
    case other

    var id: String { rawValue }

    var label: String {
        switch self {
        case .mortgage: "Crédit immobilier"
        case .autoLoan: "Crédit auto"
        case .consumerLoan: "Crédit conso"
        case .other: "Autre dette"
        }
    }

    var systemImage: String {
        switch self {
        case .mortgage: "house.lodge"
        case .autoLoan: "car.rear"
        case .consumerLoan: "cart"
        case .other: "minus.circle"
        }
    }
}

struct Asset: Identifiable, Codable, Hashable, Sendable {
    let id: String
    var name: String
    var kind: AssetKind
    var currency: String
    var note: String
    var archived: Bool
    var latestValue: Cents?

    enum CodingKeys: String, CodingKey {
        case id, name, kind, currency, note, archived
        case latestValue = "latest_value_cents"
    }
}

struct Liability: Identifiable, Codable, Hashable, Sendable {
    let id: String
    var name: String
    var kind: LiabilityKind
    var currency: String
    var note: String
    var archived: Bool
    var latestValue: Cents?

    enum CodingKeys: String, CodingKey {
        case id, name, kind, currency, note, archived
        case latestValue = "latest_value_cents"
    }
}

struct Valuation: Identifiable, Codable, Hashable, Sendable {
    let id: String
    var value: Cents
    var asOf: Date
    var note: String

    enum CodingKeys: String, CodingKey {
        case id, note
        case value = "value_cents"
        case asOf = "as_of"
    }
}

// MARK: - Patrimoine net (CA-1 : calcul côté backend, jamais côté client)

struct NetWorth: Codable, Hashable, Sendable {
    var assetsTotal: Cents
    var liabilitiesTotal: Cents
    var net: Cents
    var currency: String

    enum CodingKeys: String, CodingKey {
        case assetsTotal = "assets_total_cents"
        case liabilitiesTotal = "liabilities_total_cents"
        case net = "net_cents"
        case currency
    }
}

struct NetWorthPoint: Codable, Hashable, Identifiable, Sendable {
    var asOf: Date
    var assetsTotal: Cents
    var liabilitiesTotal: Cents
    var net: Cents

    var id: Date { asOf }

    enum CodingKeys: String, CodingKey {
        case asOf = "as_of"
        case assetsTotal = "assets_total_cents"
        case liabilitiesTotal = "liabilities_total_cents"
        case net = "net_cents"
    }
}

struct NetWorthHistory: Codable, Hashable, Sendable {
    var points: [NetWorthPoint]
    var currency: String
}

// MARK: - Projection (EF-040/041) — calculs du moteur déterministe backend

struct ProjectionPoint: Codable, Hashable, Identifiable, Sendable {
    var month: Int
    var net: Cents

    var id: Int { month }

    enum CodingKeys: String, CodingKey {
        case month
        case net = "net_cents"
    }
}

struct IndependenceResult: Codable, Hashable, Sendable {
    var reached: Bool
    var months: Int
    var target: Cents

    enum CodingKeys: String, CodingKey {
        case reached, months
        case target = "target_cents"
    }
}

struct ProjectionResponse: Codable, Hashable, Sendable {
    var startNet: Cents
    var monthlySavings: Cents
    var annualReturnBps: Int
    var monthlyExpenses: Cents
    var swrBps: Int
    var points: [ProjectionPoint]
    var independence: IndependenceResult

    enum CodingKeys: String, CodingKey {
        case startNet = "start_net_cents"
        case monthlySavings = "monthly_savings_cents"
        case annualReturnBps = "annual_return_bps"
        case monthlyExpenses = "monthly_expenses_cents"
        case swrBps = "swr_bps"
        case points, independence
    }
}

// MARK: - Flux : catégories & transactions (EF-020→022)

struct Category: Identifiable, Codable, Hashable, Sendable {
    let id: String
    var name: String
    var icon: String

    enum CodingKeys: String, CodingKey { case id, name, icon }
}

struct Transaction: Identifiable, Codable, Hashable, Sendable {
    let id: String
    var assetID: String
    var amount: Cents
    var occurredOn: Date
    var label: String
    var rawLabel: String
    var categoryID: String?
    var categoryName: String?
    var note: String

    enum CodingKeys: String, CodingKey {
        case id
        case assetID = "asset_id"
        case amount = "amount_cents"
        case occurredOn = "occurred_on"
        case label
        case rawLabel = "raw_label"
        case categoryID = "category_id"
        case categoryName = "category_name"
        case note
    }
}

struct MonthSummary: Codable, Hashable, Sendable {
    var income: Cents
    var expenses: Cents
    var net: Cents

    enum CodingKeys: String, CodingKey {
        case income = "income_cents"
        case expenses = "expenses_cents"
        case net = "net_cents"
    }
}

struct ImportResult: Codable, Hashable, Sendable {
    var imported: Int
    var duplicates: Int
    var categorized: Int
}

// MARK: - Pilotage (P4) : enveloppes, récurrents, cashflow, score, objectifs

struct EnvelopeStatus: Identifiable, Codable, Hashable, Sendable {
    let id: String
    var categoryID: String
    var categoryName: String
    var categoryIcon: String
    var budget: Cents
    var spent: Cents
    var remaining: Cents

    enum CodingKeys: String, CodingKey {
        case id
        case categoryID = "category_id"
        case categoryName = "category_name"
        case categoryIcon = "category_icon"
        case budget = "monthly_budget_cents"
        case spent = "spent_cents"
        case remaining = "remaining_cents"
    }
}

struct RecurringFlow: Identifiable, Codable, Hashable, Sendable {
    var merchantKey: String
    var label: String
    var amount: Cents
    var intervalDays: Int
    var periodicity: String
    var nextDate: Date
    var active: Bool

    var id: String { merchantKey }

    var periodicityLabel: String {
        switch periodicity {
        case "weekly": "Hebdomadaire"
        case "monthly": "Mensuel"
        case "quarterly": "Trimestriel"
        case "yearly": "Annuel"
        default: periodicity
        }
    }

    enum CodingKeys: String, CodingKey {
        case merchantKey = "merchant_key"
        case label
        case amount = "amount_cents"
        case intervalDays = "interval_days"
        case periodicity
        case nextDate = "next_date"
        case active
    }
}

struct UpcomingFlow: Codable, Hashable, Identifiable, Sendable {
    var date: Date
    var label: String
    var amount: Cents

    var id: String { "\(date.timeIntervalSince1970)-\(label)" }

    enum CodingKeys: String, CodingKey {
        case date, label
        case amount = "amount_cents"
    }
}

struct CashProjection: Codable, Hashable, Sendable {
    var startCash: Cents
    var endCash: Cents
    var until: Date
    var upcoming: [UpcomingFlow]

    enum CodingKeys: String, CodingKey {
        case startCash = "start_cash_cents"
        case endCash = "end_cash_cents"
        case until, upcoming
    }
}

struct HealthComponent: Codable, Hashable, Identifiable, Sendable {
    var name: String
    var score: Int
    var max: Int
    var comment: String

    var id: String { name }
}

struct HealthScore: Codable, Hashable, Sendable {
    var score: Int
    var components: [HealthComponent]
}

struct GoalStatus: Identifiable, Codable, Hashable, Sendable {
    let id: String
    var name: String
    var icon: String
    var target: Cents
    var targetDate: Date?
    var assetID: String?
    var assetName: String?
    var progress: Cents
    var percent: Int
    var onTrack: Bool?

    enum CodingKeys: String, CodingKey {
        case id, name, icon, percent
        case target = "target_cents"
        case targetDate = "target_date"
        case assetID = "asset_id"
        case assetName = "asset_name"
        case progress = "progress_cents"
        case onTrack = "on_track"
    }
}

struct OpaleAlert: Codable, Hashable, Identifiable, Sendable {
    var kind: String
    var severity: String
    var title: String
    var detail: String

    var id: String { kind + title }
}

// MARK: - Le cerveau (P5)

/// Un risque détecté par le radar (EF-061).
struct Risk: Codable, Hashable, Identifiable, Sendable {
    var id: String
    var title: String
    var severity: String // info | warning | critical
    var detail: String
}

/// État de la cascade IA (EIA-021/022).
struct AssistantStatus: Codable, Hashable, Sendable {
    var homelabAvailable: Bool
    var cloudConfigured: Bool

    enum CodingKeys: String, CodingKey {
        case homelabAvailable = "homelab_available"
        case cloudConfigured = "cloud_configured"
    }
}

/// Réponse de l'assistant (EF-050/051).
struct AskResponse: Codable, Hashable, Sendable {
    var answer: String
    var tier: String // "n2" | "n3" | "" (repli moteur)
}

/// Un scénario du Mode Décision (EF-052).
struct DecisionScenario: Codable, Hashable, Identifiable, Sendable {
    var name: String
    var returnBps: Int
    var in5y: Cents
    var in10y: Cents
    var delta5y: Cents
    var delta10y: Cents
    var baselineReached: Bool
    var decisionReached: Bool
    var delayMonths: Int

    var id: String { name }

    enum CodingKeys: String, CodingKey {
        case name
        case returnBps = "return_bps"
        case in5y = "in_5y_cents"
        case in10y = "in_10y_cents"
        case delta5y = "delta_5y_cents"
        case delta10y = "delta_10y_cents"
        case baselineReached = "baseline_reached"
        case decisionReached = "decision_reached"
        case delayMonths = "delay_months"
    }
}

/// Le verdict complet du Mode Décision (chiffres du moteur).
struct DecisionImpact: Codable, Hashable, Sendable {
    var netWorthAfter: Cents
    var affordableCash: Bool
    var savingsAfter: Cents
    var riskLevel: String // faible | modéré | élevé
    var recommendation: String
    var scenarios: [DecisionScenario]

    enum CodingKeys: String, CodingKey {
        case netWorthAfter = "net_worth_after_cents"
        case affordableCash = "affordable_cash"
        case savingsAfter = "savings_after_cents"
        case riskLevel = "risk_level"
        case recommendation, scenarios
    }
}

/// Réponse complète de POST /v1/decision.
struct DecisionResponse: Codable, Hashable, Sendable {
    var label: String
    var impact: DecisionImpact
    var narrative: String
    var narrativeTier: String

    enum CodingKeys: String, CodingKey {
        case label, impact, narrative
        case narrativeTier = "narrative_tier"
    }
}

/// Un poste de dépense du bilan mensuel.
struct CategorySpend: Codable, Hashable, Identifiable, Sendable {
    var name: String
    var icon: String
    var total: Cents

    var id: String { name }

    enum CodingKeys: String, CodingKey {
        case name, icon
        case total = "total_cents"
    }
}

/// Bilan mensuel intelligent (EF-062).
struct MonthlyReview: Codable, Hashable, Sendable {
    var year: Int
    var month: Int
    var summary: MonthSummary
    var savingsRateBps: Int
    var topCategories: [CategorySpend]?
    var healthScore: Int
    var narrative: String
    var narrativeTier: String

    enum CodingKeys: String, CodingKey {
        case year, month, summary, narrative
        case savingsRateBps = "savings_rate_bps"
        case topCategories = "top_categories"
        case healthScore = "health_score"
        case narrativeTier = "narrative_tier"
    }
}

// MARK: - Enveloppes de listes renvoyées par l'API

struct AssetsEnvelope: Codable, Sendable { let assets: [Asset] }
struct LiabilitiesEnvelope: Codable, Sendable { let liabilities: [Liability] }
struct ValuationsEnvelope: Codable, Sendable { let valuations: [Valuation] }
struct ProfilesEnvelope: Codable, Sendable { let profiles: [Profile] }
struct CategoriesEnvelope: Codable, Sendable { let categories: [Category] }
struct TransactionsEnvelope: Codable, Sendable { let transactions: [Transaction] }
