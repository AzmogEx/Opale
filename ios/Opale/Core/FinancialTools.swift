import Foundation

struct FinancialContract: Codable, Equatable, Identifiable, Sendable {
    var id = UUID().uuidString
    var revision: Int64 = 0
    var name = ""
    var category = "subscription"
    var amount = Cents.zero
    var currency = "EUR"
    var frequency = "monthly"
    var nextDueDate = Date.now.opaleDayString
    var assetID = ""
    var calendarRuleID = ""
    var merchantKey = ""
    var trialEnd = ""
    var commitmentEnd = ""
    var renewalDate = ""
    var autoRenew = false
    var noticeDays = 30
    var reminderDays = 7
    var active = true
    var note = ""
    var priceSince = ""
    var pendingPrice: ContractPriceObservation?
    enum CodingKeys: String, CodingKey {
        case id, revision, name, category, currency, frequency, active, note
        case amount = "amount_cents", nextDueDate = "next_due_date", assetID = "asset_id"
        case calendarRuleID = "calendar_rule_id", merchantKey = "merchant_key", trialEnd = "trial_end"
        case commitmentEnd = "commitment_end", renewalDate = "renewal_date", autoRenew = "auto_renew"
        case noticeDays = "notice_days", reminderDays = "reminder_days", priceSince = "price_since", pendingPrice = "pending_price"
    }
}
struct ContractPriceObservation: Codable, Equatable, Sendable {
    var transactionID: String
    var date: String
    var amount: Cents
    var previous: Cents
    var annualDelta: Cents
    enum CodingKeys: String, CodingKey {
        case date
        case transactionID = "transaction_id", amount = "amount_cents", previous = "previous_cents", annualDelta = "annual_delta_cents"
    }
}
struct ContractPrice: Codable, Equatable, Identifiable, Sendable {
    var id: String
    var amount: Cents
    var currency: String
    var effectiveOn: String
    var source: String
    enum CodingKeys: String, CodingKey { case id, currency, source; case amount = "amount_cents", effectiveOn = "effective_on" }
}
struct ContractAlert: Codable, Equatable, Identifiable, Sendable {
    var id: String
    var contractID: String
    var kind: String
    var severity: String
    var title: String
    var detail: String
    var date: String
    enum CodingKeys: String, CodingKey { case id, kind, severity, title, detail, date; case contractID = "contract_id" }
}
struct ContractsResponse: Codable, Sendable {
    var contracts: [FinancialContract]
    var alerts: [ContractAlert]
}
struct VariableIncome: Codable, Equatable, Identifiable, Sendable {
    var id = UUID().uuidString
    var revision: Int64 = 0
    var name = ""
    var kind = "freelance"
    var currency = "EUR"
    var low = Cents.zero
    var usual = Cents.zero
    var high = Cents.zero
    var frequency = "monthly"
    var nextDate = Date.now.opaleDayString
    var forecast = "off"
    var assetID = ""
    var calendarRuleID = ""
    var merchantKey = ""
    var active = true
    var note = ""
    enum CodingKeys: String, CodingKey {
        case id, revision, name, kind, currency, frequency, forecast, active, note
        case low = "low_cents", usual = "usual_cents", high = "high_cents", nextDate = "next_date"
        case assetID = "asset_id", calendarRuleID = "calendar_rule_id", merchantKey = "merchant_key"
    }
}

/// Exact annualisation, grouped by currency; one-off income is never monthly income.
enum FinancialTools {
    static let currencies = ["EUR", "USD", "GBP", "CHF", "JPY", "KWD", "CAD", "AUD"]
    static func frequencyLabel(_ value: String) -> String {
        switch value { case "once": "Ponctuel"; case "monthly": "Mensuel"; case "quarterly": "Trimestriel"; case "yearly": "Annuel"; default: value }
    }
    static func annualMultiplier(_ frequency: String) -> Decimal {
        switch frequency { case "monthly": 12; case "quarterly": 4; case "yearly": 1; default: 0 }
    }
    static func amount(_ text: String, currency: String, zeroAllowed: Bool = false) -> Cents? {
        guard let value = Cents.parse(text, currency: currency), value.raw >= (zeroAllowed ? 0 : 1), value.raw <= FinancialSetupMoney.maximumCents else { return nil }
        return value
    }
    static func monthlyContracts(_ items: [FinancialContract], currency: String) -> Cents? {
        let total = items.filter { $0.active && $0.currency == currency }.reduce(Decimal.zero) { $0 + Decimal($1.amount.raw) * annualMultiplier($1.frequency) }
        return FinancialSetupSummary.roundedCents(total / 12)
    }
    static func monthlyIncome(_ items: [VariableIncome], currency: String, keyPath: KeyPath<VariableIncome, Cents>) -> Cents? {
        let total = items.filter { $0.active && $0.currency == currency && $0.frequency != "once" }.reduce(Decimal.zero) { $0 + Decimal($1[keyPath: keyPath].raw) * annualMultiplier($1.frequency) }
        return FinancialSetupSummary.roundedCents(total / 12)
    }
    static func dayLabel(_ day: String) -> String {
        Date.fromOpaleDay(day)?.opaleFormatted(.dateTime.day().month(.abbreviated).year()) ?? day
    }
}

extension APIClient {
    func contracts() async throws -> ContractsResponse { try await request("GET", "/v1/contracts") }
    func saveContract(_ value: FinancialContract) async throws -> FinancialContract { try await request("PUT", "/v1/contracts/\(value.id)", body: value) }
    func deleteContract(_ value: FinancialContract) async throws {
        let _: EmptyResponse = try await request("DELETE", "/v1/contracts/\(value.id)", query: [URLQueryItem(name: "revision", value: String(value.revision))])
    }
    func contractPrices(_ id: String) async throws -> [ContractPrice] {
        struct Envelope: Decodable { var prices: [ContractPrice] }
        let response: Envelope = try await request("GET", "/v1/contracts/\(id)/prices")
        return response.prices
    }
    func resolvePrice(_ value: FinancialContract, accept: Bool) async throws {
        guard let observation = value.pendingPrice else { throw APIError.invalidResponse }
        struct Body: Encodable { var revision: Int64; var transaction_id: String; var accept: Bool }
        let _: EmptyResponse = try await request("POST", "/v1/contracts/\(value.id)/price-observation", body: Body(revision: value.revision, transaction_id: observation.transactionID, accept: accept))
    }
    func variableIncomes() async throws -> [VariableIncome] {
        struct Envelope: Decodable { var incomes: [VariableIncome] }
        let response: Envelope = try await request("GET", "/v1/incomes/variable")
        return response.incomes
    }
    func saveVariableIncome(_ value: VariableIncome) async throws -> VariableIncome { try await request("PUT", "/v1/incomes/variable/\(value.id)", body: value) }
    func deleteVariableIncome(_ value: VariableIncome) async throws {
        let _: EmptyResponse = try await request("DELETE", "/v1/incomes/variable/\(value.id)", query: [URLQueryItem(name: "revision", value: String(value.revision))])
    }
}
