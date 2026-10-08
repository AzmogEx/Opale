import Foundation

// Le pilote automatique (P8) : fiscal, crédit, allocation, alertes
// personnalisées, Wrapped, cours automatiques, snapshots, démo, push.

// MARK: - Modèles

/// Une tranche du barème IR appliquée au revenu.
struct TaxBracketDetail: Decodable, Hashable {
    let upTo: Cents
    let rateBps: Int
    let taxed: Cents
    let tax: Cents
    enum CodingKeys: String, CodingKey {
        case upTo = "up_to_cents"
        case rateBps = "rate_bps"
        case taxed = "taxed_cents"
        case tax = "tax_cents"
    }
}

/// L'estimation d'impôt sur le revenu (indicative, moteur déterministe).
struct TaxEstimate: Decodable, Hashable {
    let tax: Cents
    let marginalRateBps: Int
    let averageRateBps: Int
    let details: [TaxBracketDetail]
    let netIncome: Cents
    enum CodingKeys: String, CodingKey {
        case tax = "tax_cents"
        case marginalRateBps = "marginal_rate_bps"
        case averageRateBps = "average_rate_bps"
        case details
        case netIncome = "net_income_cents"
    }
}

/// L'effet d'un versement PER sur l'impôt.
struct PEREffect: Decodable, Hashable {
    let contribution: Cents
    let taxBefore: Cents
    let taxAfter: Cents
    let savings: Cents
    let realCost: Cents
    enum CodingKeys: String, CodingKey {
        case contribution = "contribution_cents"
        case taxBefore = "tax_before_cents"
        case taxAfter = "tax_after_cents"
        case savings = "savings_cents"
        case realCost = "real_cost_cents"
    }
}

/// Une échéance fiscale de l'année.
struct TaxDeadline: Decodable, Hashable, Identifiable {
    let month: Int
    let day: Int
    let title: String
    var id: String { "\(month)-\(day)" }
}

/// Une année du tableau d'amortissement d'un prêt.
struct LoanYear: Decodable, Hashable, Identifiable {
    let year: Int
    let interest: Cents
    let principal: Cents
    let remaining: Cents
    var id: Int { year }
    enum CodingKeys: String, CodingKey {
        case year
        case interest = "interest_cents"
        case principal = "principal_cents"
        case remaining = "remaining_cents"
    }
}

/// La simulation complète d'un crédit.
struct LoanResult: Decodable, Hashable {
    let monthlyPayment: Cents
    let totalPaid: Cents
    let totalInterest: Cents
    let schedule: [LoanYear]
    enum CodingKeys: String, CodingKey {
        case monthlyPayment = "monthly_payment_cents"
        case totalPaid = "total_paid_cents"
        case totalInterest = "total_interest_cents"
        case schedule
    }
}

/// Une classe d'actifs : réel vs cible.
struct AllocationLine: Decodable, Hashable, Identifiable {
    let `class`: String
    let actual: Cents
    let actualBps: Int
    let targetBps: Int
    let drift: Cents
    var id: String { `class` }
    enum CodingKeys: String, CodingKey {
        case `class`
        case actual = "actual_cents"
        case actualBps = "actual_bps"
        case targetBps = "target_bps"
        case drift = "drift_cents"
    }

    var label: String {
        switch `class` {
        case "stocks": "Actions"
        case "real_estate": "Immobilier"
        case "crypto": "Crypto"
        case "cash": "Liquidités"
        default: "Autres"
        }
    }
}

/// L'état du portefeuille face à l'allocation cible.
struct AllocationStatus: Decodable {
    let total: Cents
    let lines: [AllocationLine]
    let hasTargets: Bool
    enum CodingKeys: String, CodingKey {
        case total = "total_cents"
        case lines
        case hasTargets = "has_targets"
    }
}

/// Un seuil personnalisé surveillé par le moteur d'alertes.
struct CustomAlert: Decodable, Hashable, Identifiable {
    let id: String
    let kind: String
    let threshold: Cents
    let enabled: Bool
    enum CodingKeys: String, CodingKey {
        case id, kind, enabled
        case threshold = "threshold_cents"
    }

    static let kinds: [(kind: String, label: String)] = [
        ("cash_below", "Cash sous un seuil"),
        ("net_worth_below", "Patrimoine sous un seuil"),
        ("expenses_month_above", "Dépenses du mois au-dessus d'un seuil"),
    ]

    var kindLabel: String {
        Self.kinds.first { $0.kind == kind }?.label ?? kind
    }
}

/// Une entrée du top (catégorie ou marchand) du bilan annuel.
struct WrappedEntry: Decodable, Hashable {
    let name: String
    let total: Cents
    enum CodingKeys: String, CodingKey {
        case name
        case total = "total_cents"
    }
}

/// Le bilan annuel « Wrapped » — chiffres 100 % moteur.
struct Wrapped: Decodable {
    struct BiggestExpense: Decodable {
        let label: String
        let total: Cents
        let occurredOn: String
        enum CodingKeys: String, CodingKey {
            case label
            case total = "total_cents"
            case occurredOn = "occurred_on"
        }
    }

    let year: Int
    let income: Cents
    let expenses: Cents
    let saved: Cents
    let savingsRateBps: Int
    let activeMonths: Int
    let transactionCount: Int
    let topCategories: [WrappedEntry]
    let topMerchants: [WrappedEntry]
    let biggestExpense: BiggestExpense?
    let netWorthStart: Cents?
    let netWorthEnd: Cents?
    enum CodingKeys: String, CodingKey {
        case year
        case income = "income_cents"
        case expenses = "expenses_cents"
        case saved = "saved_cents"
        case savingsRateBps = "savings_rate_bps"
        case activeMonths = "active_months"
        case transactionCount = "transaction_count"
        case topCategories = "top_categories"
        case topMerchants = "top_merchants"
        case biggestExpense = "biggest_expense"
        case netWorthStart = "net_worth_start_cents"
        case netWorthEnd = "net_worth_end_cents"
    }
}

/// Un point mensuel du patrimoine pris par le serveur.
struct MonthlySnapshot: Decodable, Hashable {
    let month: Date
    let recordedAt: Date?
    let assets: Cents
    let liabilities: Cents
    let netWorth: Cents
    enum CodingKeys: String, CodingKey {
        case month
        case recordedAt = "recorded_at"
        case assets = "assets_cents"
        case liabilities = "liabilities_cents"
        case netWorth = "net_worth_cents"
    }
}

// MARK: - Endpoints

extension APIClient {
    // ── Fiscal ────────────────────────────────────────────────────────────

    struct TaxEstimateResponse: Decodable {
        let estimate: TaxEstimate
        let perEffect: PEREffect?
        let assessmentYear: Int
        let incomeYear: Int
        let verifiedOn: String
        let sourceURL: String
        let disclaimer: String
        let deductible: Cents?
        enum CodingKeys: String, CodingKey {
            case estimate
            case perEffect = "per_effect"
            case assessmentYear = "assessment_year", incomeYear = "income_year", verifiedOn = "verified_on"
            case sourceURL = "source_url", disclaimer, deductible = "deductible_cents"
        }
    }

    func taxEstimate(incomeCents: Int64, partsTenths: Int, perCents: Int64? = nil, perCeilingCents: Int64? = nil) async throws -> TaxEstimateResponse {
        var query = [
            URLQueryItem(name: "income_cents", value: String(incomeCents)),
            URLQueryItem(name: "parts_tenths", value: String(partsTenths)),
        ]
        query.append(URLQueryItem(name: "year", value: "2026"))
        if let perCeilingCents { query.append(URLQueryItem(name: "per_ceiling_cents", value: String(perCeilingCents))) }
        if let perCents, perCents > 0 {
            query.append(URLQueryItem(name: "per_cents", value: String(perCents)))
        }
        return try await request("GET", "/v1/tax/estimate", query: query)
    }

    func taxDeadlines() async throws -> [TaxDeadline] {
        struct Envelope: Decodable { let deadlines: [TaxDeadline] }
        let env: Envelope = try await request("GET", "/v1/tax/deadlines")
        return env.deadlines
    }

    // ── Crédit ────────────────────────────────────────────────────────────

    func loanSimulate(principalCents: Int64, annualRateBps: Int, months: Int) async throws -> LoanResult {
        struct Body: Encodable {
            let principalCents: Int64
            let annualRateBps: Int
            let months: Int
            enum CodingKeys: String, CodingKey {
                case principalCents = "principal_cents"
                case annualRateBps = "annual_rate_bps"
                case months
            }
        }
        return try await request("POST", "/v1/loan/simulate",
                                 body: Body(principalCents: principalCents, annualRateBps: annualRateBps, months: months))
    }

    func loanCapacity(monthlyPaymentCents: Int64, annualRateBps: Int, months: Int) async throws -> Cents {
        struct Body: Encodable {
            let monthlyPaymentCents: Int64
            let annualRateBps: Int
            let months: Int
            enum CodingKeys: String, CodingKey {
                case monthlyPaymentCents = "monthly_payment_cents"
                case annualRateBps = "annual_rate_bps"
                case months
            }
        }
        struct Envelope: Decodable {
            let capacity: Cents
            enum CodingKeys: String, CodingKey { case capacity = "capacity_cents" }
        }
        let env: Envelope = try await request("POST", "/v1/loan/capacity",
                                              body: Body(monthlyPaymentCents: monthlyPaymentCents, annualRateBps: annualRateBps, months: months))
        return env.capacity
    }

    // ── Allocation cible ──────────────────────────────────────────────────

    func allocation() async throws -> AllocationStatus {
        try await request("GET", "/v1/allocation")
    }

    func setAllocation(targetsBps: [String: Int]) async throws {
        struct Target: Encodable {
            let `class`: String
            let targetBps: Int
            enum CodingKeys: String, CodingKey {
                case `class`
                case targetBps = "target_bps"
            }
        }
        struct Body: Encodable { let targets: [Target] }
        let body = Body(targets: targetsBps.map { Target(class: $0.key, targetBps: $0.value) })
        let _: EmptyResponse = try await request("PUT", "/v1/allocation", body: body)
    }

    // ── Alertes personnalisées ────────────────────────────────────────────

    func customAlerts() async throws -> [CustomAlert] {
        struct Envelope: Decodable { let alerts: [CustomAlert] }
        let env: Envelope = try await request("GET", "/v1/alerts/custom")
        return env.alerts
    }

    func createCustomAlert(kind: String, thresholdCents: Int64) async throws -> CustomAlert {
        struct Body: Encodable {
            let kind: String
            let thresholdCents: Int64
            enum CodingKeys: String, CodingKey {
                case kind
                case thresholdCents = "threshold_cents"
            }
        }
        return try await request("POST", "/v1/alerts/custom", body: Body(kind: kind, thresholdCents: thresholdCents))
    }

    func updateCustomAlert(id: String, thresholdCents: Int64, enabled: Bool) async throws {
        struct Body: Encodable {
            let thresholdCents: Int64
            let enabled: Bool
            enum CodingKeys: String, CodingKey {
                case thresholdCents = "threshold_cents"
                case enabled
            }
        }
        let _: EmptyResponse = try await request("PATCH", "/v1/alerts/custom/\(id)",
                                                 body: Body(thresholdCents: thresholdCents, enabled: enabled))
    }

    func deleteCustomAlert(id: String) async throws {
        let _: EmptyResponse = try await request("DELETE", "/v1/alerts/custom/\(id)")
    }

    // ── Wrapped, snapshots, cours, démo, push ─────────────────────────────

    func wrapped(year: Int? = nil) async throws -> Wrapped {
        var query: [URLQueryItem] = []
        if let year { query.append(URLQueryItem(name: "year", value: String(year))) }
        return try await request("GET", "/v1/wrapped", query: query)
    }

    func snapshots() async throws -> [MonthlySnapshot] {
        struct Envelope: Decodable { let snapshots: [MonthlySnapshot]? }
        let env: Envelope = try await request("GET", "/v1/snapshots")
        return env.snapshots ?? []
    }

    func setAssetQuote(assetID: String, symbol: String, quantityMicro: Int64) async throws {
        struct Body: Encodable {
            let symbol: String
            let quantityMicro: Int64
            enum CodingKeys: String, CodingKey {
                case symbol
                case quantityMicro = "quantity_micro"
            }
        }
        let _: EmptyResponse = try await request("PUT", "/v1/assets/\(assetID)/quote",
                                                 body: Body(symbol: symbol, quantityMicro: quantityMicro))
    }

    func refreshQuotes() async throws {
        let _: EmptyResponse = try await request("POST", "/v1/quotes/refresh")
    }

    func createDemoProfile() async throws -> Profile {
        struct Envelope: Decodable { let profile: Profile }
        let env: Envelope = try await request("POST", "/v1/profiles/demo", authenticated: false)
        return env.profile
    }

    func registerPushToken(_ token: String) async throws {
        struct Body: Encodable { let token: String }
        let _: EmptyResponse = try await request("POST", "/v1/push/register", body: Body(token: token))
    }
}
