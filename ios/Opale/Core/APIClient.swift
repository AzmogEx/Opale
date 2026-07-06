import Foundation

/// Erreurs de l'API Opale.
enum APIError: LocalizedError {
    case badStatus(Int, message: String?)
    case invalidResponse
    case notAuthenticated

    var errorDescription: String? {
        switch self {
        case .badStatus(let code, let message):
            message ?? "Erreur serveur (\(code))"
        case .invalidResponse:
            "Réponse invalide du serveur"
        case .notAuthenticated:
            "Session expirée — reconnecte-toi"
        }
    }
}

/// Client HTTP de l'API Opale (backend Go du homelab).
///
/// Service simple injecté via `SessionStore` ; toutes les méthodes sont
/// asynchrones et lèvent `APIError` en cas d'échec.
final class APIClient: Sendable {
    /// URL de base, ex. `http://localhost:8080` (simulateur → backend local).
    let baseURL: URL
    private let tokenProvider: @Sendable () -> String?

    init(baseURL: URL, tokenProvider: @escaping @Sendable () -> String?) {
        self.baseURL = baseURL
        self.tokenProvider = tokenProvider
    }

    // MARK: - Décodage

    /// Décodeur JSON tolérant aux deux formats de date du backend :
    /// ISO 8601 avec fractions ("…T17:34:48.753405+02:00") et sans ("…T00:00:00Z").
    nonisolated static func makeDecoder() -> JSONDecoder {
        let withFractional = ISO8601DateFormatter()
        withFractional.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        let plain = ISO8601DateFormatter()
        plain.formatOptions = [.withInternetDateTime]

        let d = JSONDecoder()
        d.dateDecodingStrategy = .custom { decoder in
            let s = try decoder.singleValueContainer().decode(String.self)
            if let date = withFractional.date(from: s) ?? plain.date(from: s) {
                return date
            }
            throw DecodingError.dataCorrupted(.init(
                codingPath: decoder.codingPath,
                debugDescription: "Date invalide : \(s)"
            ))
        }
        return d
    }

    // MARK: - Requête générique

    private struct APIErrorBody: Decodable {
        struct Inner: Decodable { let code: String; let message: String }
        let error: Inner
    }

    private func request<T: Decodable>(
        _ method: String,
        _ path: String,
        query: [URLQueryItem] = [],
        body: (some Encodable)? = Optional<Int>.none,
        authenticated: Bool = true
    ) async throws -> T {
        var components = URLComponents(
            url: baseURL.appending(path: path),
            resolvingAgainstBaseURL: false
        )!
        if !query.isEmpty { components.queryItems = query }

        var req = URLRequest(url: components.url!)
        req.httpMethod = method
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        if authenticated {
            guard let token = tokenProvider() else { throw APIError.notAuthenticated }
            req.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        if let body {
            req.httpBody = try JSONEncoder().encode(body)
        }

        let (data, response) = try await URLSession.shared.data(for: req)
        guard let http = response as? HTTPURLResponse else { throw APIError.invalidResponse }

        guard (200..<300).contains(http.statusCode) else {
            if http.statusCode == 401 { throw APIError.notAuthenticated }
            let message = try? Self.makeDecoder().decode(APIErrorBody.self, from: data).error.message
            throw APIError.badStatus(http.statusCode, message: message)
        }

        if T.self == EmptyResponse.self { return EmptyResponse() as! T }
        return try Self.makeDecoder().decode(T.self, from: data)
    }

    struct EmptyResponse: Decodable {}

    // MARK: - Corps de requêtes

    struct CreateProfileRequest: Encodable {
        let name: String
        let pin: String
        let privacyDefault = "N1"
        enum CodingKeys: String, CodingKey {
            case name, pin
            case privacyDefault = "privacy_default"
        }
    }

    struct LoginRequest: Encodable {
        let profileID: String
        let pin: String
        enum CodingKeys: String, CodingKey {
            case profileID = "profile_id"
            case pin
        }
    }

    struct UpsertAssetRequest: Encodable {
        let name: String
        let kind: String
        let note: String
        let currency: String
    }

    struct UpsertLiabilityRequest: Encodable {
        let name: String
        let kind: String
        let note: String
    }

    struct AddValuationRequest: Encodable {
        let valueCents: Int64
        /// Format `yyyy-MM-dd` (colonne DATE côté backend).
        let asOf: String
        enum CodingKeys: String, CodingKey {
            case valueCents = "value_cents"
            case asOf = "as_of"
        }
    }

    // MARK: - Endpoints

    func listProfiles() async throws -> [Profile] {
        let env: ProfilesEnvelope = try await request("GET", "/v1/profiles", authenticated: false)
        return env.profiles
    }

    func createProfile(name: String, pin: String) async throws -> Profile {
        try await request(
            "POST", "/v1/profiles",
            body: CreateProfileRequest(name: name, pin: pin),
            authenticated: false
        )
    }

    func login(profileID: String, pin: String) async throws -> LoginResponse {
        try await request(
            "POST", "/v1/auth/login",
            body: LoginRequest(profileID: profileID, pin: pin),
            authenticated: false
        )
    }

    func logout() async throws {
        let _: EmptyResponse = try await request("POST", "/v1/auth/logout")
    }

    /// Profil de la session courante (valide le jeton restauré au lancement).
    func me() async throws -> Profile {
        try await request("GET", "/v1/me")
    }

    func netWorth() async throws -> NetWorth {
        try await request("GET", "/v1/net-worth")
    }

    func netWorthHistory(months: Int) async throws -> NetWorthHistory {
        try await request(
            "GET", "/v1/net-worth/history",
            query: [URLQueryItem(name: "months", value: String(months))]
        )
    }

    /// Projection du patrimoine + date d'indépendance (EF-040/041).
    /// Les calculs sont faits par le moteur déterministe du backend.
    func projection(
        monthlySavingsCents: Int64,
        annualReturnBps: Int,
        monthlyExpensesCents: Int64,
        months: Int = 360
    ) async throws -> ProjectionResponse {
        try await request(
            "GET", "/v1/projection",
            query: [
                URLQueryItem(name: "monthly_savings_cents", value: String(monthlySavingsCents)),
                URLQueryItem(name: "annual_return_bps", value: String(annualReturnBps)),
                URLQueryItem(name: "monthly_expenses_cents", value: String(monthlyExpensesCents)),
                URLQueryItem(name: "months", value: String(months)),
            ]
        )
    }

    func listAssets() async throws -> [Asset] {
        let env: AssetsEnvelope = try await request("GET", "/v1/assets/")
        return env.assets
    }

    func createAsset(name: String, kind: AssetKind, note: String = "", currency: String = "EUR") async throws -> Asset {
        try await request(
            "POST", "/v1/assets/",
            body: UpsertAssetRequest(name: name, kind: kind.rawValue, note: note, currency: currency)
        )
    }

    func deleteAsset(id: String) async throws {
        let _: EmptyResponse = try await request("DELETE", "/v1/assets/\(id)/")
    }

    func assetValuations(assetID: String) async throws -> [Valuation] {
        let env: ValuationsEnvelope = try await request("GET", "/v1/assets/\(assetID)/valuations")
        return env.valuations
    }

    func addAssetValuation(assetID: String, valueCents: Int64, asOf: String) async throws -> Valuation {
        try await request(
            "POST", "/v1/assets/\(assetID)/valuations",
            body: AddValuationRequest(valueCents: valueCents, asOf: asOf)
        )
    }

    // MARK: Flux (EF-020→022)

    func listCategories() async throws -> [Category] {
        let env: CategoriesEnvelope = try await request("GET", "/v1/categories")
        return env.categories
    }

    func listTransactions(
        from: String? = nil, to: String? = nil,
        query: String? = nil, categoryID: String? = nil
    ) async throws -> [Transaction] {
        var items: [URLQueryItem] = []
        if let from { items.append(URLQueryItem(name: "from", value: from)) }
        if let to { items.append(URLQueryItem(name: "to", value: to)) }
        if let query, !query.isEmpty { items.append(URLQueryItem(name: "q", value: query)) }
        if let categoryID { items.append(URLQueryItem(name: "category_id", value: categoryID)) }
        let env: TransactionsEnvelope = try await request("GET", "/v1/transactions/", query: items)
        return env.transactions
    }

    struct CreateTransactionRequest: Encodable {
        let assetID: String
        let amountCents: Int64
        let occurredOn: String
        let label: String
        let categoryID: String
        let note: String
        enum CodingKeys: String, CodingKey {
            case assetID = "asset_id"
            case amountCents = "amount_cents"
            case occurredOn = "occurred_on"
            case label
            case categoryID = "category_id"
            case note
        }
    }

    func createTransaction(_ req: CreateTransactionRequest) async throws -> Transaction {
        try await request("POST", "/v1/transactions/", body: req)
    }

    struct PatchTransactionRequest: Encodable {
        var label: String?
        var note: String?
        var categoryID: String?
        var applyToSimilar: Bool?
        enum CodingKeys: String, CodingKey {
            case label, note
            case categoryID = "category_id"
            case applyToSimilar = "apply_to_similar"
        }
    }

    func updateTransaction(id: String, _ patch: PatchTransactionRequest) async throws -> Transaction {
        try await request("PATCH", "/v1/transactions/\(id)/", body: patch)
    }

    func deleteTransaction(id: String) async throws {
        let _: EmptyResponse = try await request("DELETE", "/v1/transactions/\(id)/")
    }

    func monthSummary(year: Int, month: Int) async throws -> MonthSummary {
        try await request(
            "GET", "/v1/transactions/summary",
            query: [
                URLQueryItem(name: "year", value: String(year)),
                URLQueryItem(name: "month", value: String(month)),
            ]
        )
    }

    struct ImportCSVRequest: Encodable {
        let assetID: String
        let csv: String
        enum CodingKeys: String, CodingKey {
            case assetID = "asset_id"
            case csv
        }
    }

    func importCSV(assetID: String, csv: String) async throws -> ImportResult {
        try await request("POST", "/v1/transactions/import", body: ImportCSVRequest(assetID: assetID, csv: csv))
    }

    // MARK: Pilotage (P4)

    func envelopeStatuses(year: Int, month: Int) async throws -> [EnvelopeStatus] {
        struct Env: Decodable { let envelopes: [EnvelopeStatus] }
        let env: Env = try await request(
            "GET", "/v1/envelopes/",
            query: [
                URLQueryItem(name: "year", value: String(year)),
                URLQueryItem(name: "month", value: String(month)),
            ]
        )
        return env.envelopes
    }

    struct UpsertEnvelopeRequest: Encodable {
        let categoryID: String
        let budgetCents: Int64
        enum CodingKeys: String, CodingKey {
            case categoryID = "category_id"
            case budgetCents = "monthly_budget_cents"
        }
    }

    func upsertEnvelope(categoryID: String, budgetCents: Int64) async throws {
        struct Env: Decodable { let id: String }
        let _: Env = try await request(
            "PUT", "/v1/envelopes/",
            body: UpsertEnvelopeRequest(categoryID: categoryID, budgetCents: budgetCents)
        )
    }

    func deleteEnvelope(id: String) async throws {
        let _: EmptyResponse = try await request("DELETE", "/v1/envelopes/\(id)")
    }

    func recurringFlows() async throws -> [RecurringFlow] {
        struct Env: Decodable { let recurring: [RecurringFlow] }
        let env: Env = try await request("GET", "/v1/recurring")
        return env.recurring
    }

    func cashflow(days: Int = 30) async throws -> CashProjection {
        try await request(
            "GET", "/v1/cashflow",
            query: [URLQueryItem(name: "days", value: String(days))]
        )
    }

    func healthScore() async throws -> HealthScore {
        try await request("GET", "/v1/health-score")
    }

    func analytics(year: Int? = nil, month: Int? = nil) async throws -> MonthAnalytics {
        var query: [URLQueryItem] = []
        if let year { query.append(URLQueryItem(name: "year", value: String(year))) }
        if let month { query.append(URLQueryItem(name: "month", value: String(month))) }
        return try await request("GET", "/v1/analytics", query: query)
    }

    func subscriptions() async throws -> (items: [SubscriptionStatus], monthly: Cents, yearly: Cents) {
        struct Env: Decodable {
            let subscriptions: [SubscriptionStatus]?
            let totalMonthly: Cents
            let totalYearly: Cents
            enum CodingKeys: String, CodingKey {
                case subscriptions
                case totalMonthly = "total_monthly_cents"
                case totalYearly = "total_yearly_cents"
            }
        }
        let env: Env = try await request("GET", "/v1/subscriptions")
        return (env.subscriptions ?? [], env.totalMonthly, env.totalYearly)
    }

    func alerts() async throws -> [OpaleAlert] {
        struct Env: Decodable { let alerts: [OpaleAlert] }
        let env: Env = try await request("GET", "/v1/alerts")
        return env.alerts
    }

    func listGoals() async throws -> [GoalStatus] {
        struct Env: Decodable { let goals: [GoalStatus] }
        let env: Env = try await request("GET", "/v1/goals/")
        return env.goals
    }

    struct CreateGoalRequest: Encodable {
        let name: String
        let icon: String
        let targetCents: Int64
        let targetDate: String
        let assetID: String
        enum CodingKeys: String, CodingKey {
            case name, icon
            case targetCents = "target_cents"
            case targetDate = "target_date"
            case assetID = "asset_id"
        }
    }

    func createGoal(_ req: CreateGoalRequest) async throws {
        struct Env: Decodable { let id: String }
        let _: Env = try await request("POST", "/v1/goals/", body: req)
    }

    func deleteGoal(id: String) async throws {
        let _: EmptyResponse = try await request("DELETE", "/v1/goals/\(id)")
    }

    func listLiabilities() async throws -> [Liability] {
        let env: LiabilitiesEnvelope = try await request("GET", "/v1/liabilities/")
        return env.liabilities
    }

    func createLiability(name: String, kind: LiabilityKind, note: String = "") async throws -> Liability {
        try await request(
            "POST", "/v1/liabilities/",
            body: UpsertLiabilityRequest(name: name, kind: kind.rawValue, note: note)
        )
    }

    func deleteLiability(id: String) async throws {
        let _: EmptyResponse = try await request("DELETE", "/v1/liabilities/\(id)/")
    }

    func liabilityValuations(liabilityID: String) async throws -> [Valuation] {
        let env: ValuationsEnvelope = try await request("GET", "/v1/liabilities/\(liabilityID)/valuations")
        return env.valuations
    }

    func addLiabilityValuation(liabilityID: String, valueCents: Int64, asOf: String) async throws -> Valuation {
        try await request(
            "POST", "/v1/liabilities/\(liabilityID)/valuations",
            body: AddValuationRequest(valueCents: valueCents, asOf: asOf)
        )
    }

    // MARK: - Le cerveau (P5)

    func assistantStatus() async throws -> AssistantStatus {
        try await request("GET", "/v1/assistant/status")
    }

    struct AskRequest: Encodable {
        let question: String
        let allowCloud: Bool
        enum CodingKeys: String, CodingKey {
            case question
            case allowCloud = "allow_cloud"
        }
    }

    func ask(question: String, allowCloud: Bool = false) async throws -> AskResponse {
        try await request("POST", "/v1/assistant/ask",
                          body: AskRequest(question: question, allowCloud: allowCloud))
    }

    func risks() async throws -> [Risk] {
        struct Env: Decodable { let risks: [Risk]? }
        let env: Env = try await request("GET", "/v1/risks")
        return env.risks ?? []
    }

    struct DecisionRequest: Encodable {
        let label: String
        let oneTimeCostCents: Int64
        let monthlyCostCents: Int64
        let allowCloud: Bool
        enum CodingKeys: String, CodingKey {
            case label
            case oneTimeCostCents = "one_time_cost_cents"
            case monthlyCostCents = "monthly_cost_cents"
            case allowCloud = "allow_cloud"
        }
    }

    func evaluateDecision(_ req: DecisionRequest) async throws -> DecisionResponse {
        try await request("POST", "/v1/decision", body: req)
    }

    func monthlyReview(year: Int? = nil, month: Int? = nil, allowCloud: Bool = false) async throws -> MonthlyReview {
        var query: [URLQueryItem] = []
        if let year { query.append(URLQueryItem(name: "year", value: String(year))) }
        if let month { query.append(URLQueryItem(name: "month", value: String(month))) }
        if allowCloud { query.append(URLQueryItem(name: "allow_cloud", value: "true")) }
        return try await request("GET", "/v1/monthly-review", query: query)
    }

    // MARK: - La profondeur (P6)

    func realEstate() async throws -> [PropertyStatus] {
        struct Env: Decodable { let properties: [PropertyStatus]? }
        let env: Env = try await request("GET", "/v1/real-estate")
        return env.properties ?? []
    }

    struct PropertyRequest: Encodable {
        let purchasePriceCents: Int64
        let purchaseDate: String
        let monthlyRentCents: Int64
        let monthlyChargesCents: Int64
        let propertyTaxYearlyCents: Int64
        let liabilityID: String
        let monthlyLoanPaymentCents: Int64
        enum CodingKeys: String, CodingKey {
            case purchasePriceCents = "purchase_price_cents"
            case purchaseDate = "purchase_date"
            case monthlyRentCents = "monthly_rent_cents"
            case monthlyChargesCents = "monthly_charges_cents"
            case propertyTaxYearlyCents = "property_tax_yearly_cents"
            case liabilityID = "liability_id"
            case monthlyLoanPaymentCents = "monthly_loan_payment_cents"
        }
    }

    func upsertProperty(assetID: String, _ req: PropertyRequest) async throws {
        let _: PropertyDetails = try await request("PUT", "/v1/assets/\(assetID)/property", body: req)
    }

    func investments() async throws -> (items: [InvestmentStatus], total: Cents) {
        struct Env: Decodable {
            let investments: [InvestmentStatus]?
            let totalCents: Cents
            enum CodingKeys: String, CodingKey {
                case investments
                case totalCents = "total_cents"
            }
        }
        let env: Env = try await request("GET", "/v1/investments")
        return (env.investments ?? [], env.totalCents)
    }

    func objects() async throws -> [ObjectStatus] {
        struct Env: Decodable { let objects: [ObjectStatus]? }
        let env: Env = try await request("GET", "/v1/objects")
        return env.objects ?? []
    }

    struct ObjectRequest: Encodable {
        let category: String
        let brand: String
        let purchasePriceCents: Int64
        let purchaseDate: String
        let insured: Bool
        enum CodingKeys: String, CodingKey {
            case category, brand, insured
            case purchasePriceCents = "purchase_price_cents"
            case purchaseDate = "purchase_date"
        }
    }

    func upsertObject(assetID: String, _ req: ObjectRequest) async throws {
        let _: ObjectDetails = try await request("PUT", "/v1/assets/\(assetID)/object", body: req)
    }

    func timeline() async throws -> [TimelineEvent] {
        struct Env: Decodable { let events: [TimelineEvent]? }
        let env: Env = try await request("GET", "/v1/timeline")
        return env.events ?? []
    }

    func documents() async throws -> (items: [VaultDocument], vaultConfigured: Bool) {
        struct Env: Decodable {
            let documents: [VaultDocument]?
            let vaultConfigured: Bool
            enum CodingKeys: String, CodingKey {
                case documents
                case vaultConfigured = "vault_configured"
            }
        }
        let env: Env = try await request("GET", "/v1/documents/")
        return (env.documents ?? [], env.vaultConfigured)
    }

    struct DocumentRequest: Encodable {
        let name: String
        let kind: String
        let mime: String
        let assetID: String
        let contentBase64: String
        enum CodingKeys: String, CodingKey {
            case name, kind, mime
            case assetID = "asset_id"
            case contentBase64 = "content_base64"
        }
    }

    func createDocument(_ req: DocumentRequest) async throws -> VaultDocument {
        try await request("POST", "/v1/documents/", body: req)
    }

    /// Télécharge le contenu déchiffré d'un document (octets bruts).
    func documentContent(id: String) async throws -> Data {
        guard let token = tokenProvider() else { throw APIError.notAuthenticated }
        var req = URLRequest(url: baseURL.appending(path: "/v1/documents/\(id)/content"))
        req.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        let (data, response) = try await URLSession.shared.data(for: req)
        guard let http = response as? HTTPURLResponse else { throw APIError.invalidResponse }
        guard (200..<300).contains(http.statusCode) else {
            if http.statusCode == 401 { throw APIError.notAuthenticated }
            throw APIError.badStatus(http.statusCode, message: nil)
        }
        return data
    }

    func deleteDocument(id: String) async throws {
        let _: EmptyResponse = try await request("DELETE", "/v1/documents/\(id)")
    }

    func contacts() async throws -> [Contact] {
        struct Env: Decodable { let contacts: [Contact]? }
        let env: Env = try await request("GET", "/v1/contacts/")
        return env.contacts ?? []
    }

    struct ContactRequest: Encodable {
        let name: String
        let role: String
        let phone: String
        let email: String
        let note: String
    }

    func createContact(_ req: ContactRequest) async throws -> Contact {
        try await request("POST", "/v1/contacts/", body: req)
    }

    func deleteContact(id: String) async throws {
        let _: EmptyResponse = try await request("DELETE", "/v1/contacts/\(id)")
    }

    func transmission() async throws -> TransmissionSummary {
        try await request("GET", "/v1/transmission")
    }

    // MARK: - Le confort (P7)

    struct ScenarioParams: Encodable {
        let label: String
        let monthlySavingsCents: Int64
        let monthlyExpensesCents: Int64
        let annualReturnBps: Int
        let oneTimeCostCents: Int64
        enum CodingKeys: String, CodingKey {
            case label
            case monthlySavingsCents = "monthly_savings_cents"
            case monthlyExpensesCents = "monthly_expenses_cents"
            case annualReturnBps = "annual_return_bps"
            case oneTimeCostCents = "one_time_cost_cents"
        }
    }

    func compareScenarios(a: ScenarioParams, b: ScenarioParams, months: Int = 240) async throws -> ScenarioComparison {
        struct Req: Encodable {
            let a: ScenarioParams
            let b: ScenarioParams
            let months: Int
        }
        return try await request("POST", "/v1/scenarios/compare", body: Req(a: a, b: b, months: months))
    }

    func companies() async throws -> [CompanyStatus] {
        struct Env: Decodable { let companies: [CompanyStatus]? }
        let env: Env = try await request("GET", "/v1/company")
        return env.companies ?? []
    }

    struct CompanyRequest: Encodable {
        let siren: String
        let ownershipBps: Int
        let ccaCents: Int64
        let annualDividendsCents: Int64
        let monthlySalaryCents: Int64
        enum CodingKeys: String, CodingKey {
            case siren
            case ownershipBps = "ownership_bps"
            case ccaCents = "cca_cents"
            case annualDividendsCents = "annual_dividends_cents"
            case monthlySalaryCents = "monthly_salary_cents"
        }
    }

    func upsertCompany(assetID: String, _ req: CompanyRequest) async throws {
        let _: CompanyDetails = try await request("PUT", "/v1/assets/\(assetID)/company", body: req)
    }

    func bankStatus() async throws -> BankStatus {
        try await request("GET", "/v1/bank/status")
    }

    func bankInstitutions(country: String = "fr") async throws -> [BankInstitution] {
        struct Env: Decodable { let institutions: [BankInstitution]? }
        let env: Env = try await request("GET", "/v1/bank/institutions",
                                         query: [URLQueryItem(name: "country", value: country)])
        return env.institutions ?? []
    }

    struct BankConnectResponse: Decodable {
        let link: BankLink
        let consentLink: String
        enum CodingKeys: String, CodingKey {
            case link
            case consentLink = "consent_link"
        }
    }

    func bankConnect(institutionID: String, institutionName: String, assetID: String) async throws -> BankConnectResponse {
        struct Req: Encodable {
            let institutionID: String
            let institutionName: String
            let assetID: String
            enum CodingKeys: String, CodingKey {
                case institutionID = "institution_id"
                case institutionName = "institution_name"
                case assetID = "asset_id"
            }
        }
        return try await request("POST", "/v1/bank/connect",
                                 body: Req(institutionID: institutionID,
                                           institutionName: institutionName,
                                           assetID: assetID))
    }

    func bankSync() async throws -> [BankSyncResult] {
        struct Env: Decodable { let results: [BankSyncResult]? }
        struct Empty: Encodable {}
        let env: Env = try await request("POST", "/v1/bank/sync", body: Empty())
        return env.results ?? []
    }

    func bankDisconnect(id: String) async throws {
        let _: EmptyResponse = try await request("DELETE", "/v1/bank/links/\(id)")
    }

    // MARK: - Espace partagé (EF-007) & devises (EF-008)

    func spaces() async throws -> [Space] {
        struct Env: Decodable { let spaces: [Space]? }
        let env: Env = try await request("GET", "/v1/spaces/")
        return env.spaces ?? []
    }

    func createSpace(name: String) async throws -> Space {
        struct Req: Encodable { let name: String }
        return try await request("POST", "/v1/spaces/", body: Req(name: name))
    }

    func spaceDetail(id: String) async throws -> SpaceDetail {
        try await request("GET", "/v1/spaces/\(id)")
    }

    func addSpaceMember(spaceID: String, profileID: String) async throws {
        struct Req: Encodable {
            let profileID: String
            enum CodingKeys: String, CodingKey { case profileID = "profile_id" }
        }
        let _: EmptyResponse = try await request("POST", "/v1/spaces/\(spaceID)/members",
                                                 body: Req(profileID: profileID))
    }

    func removeSpaceMember(spaceID: String, profileID: String) async throws {
        let _: EmptyResponse = try await request("DELETE", "/v1/spaces/\(spaceID)/members/\(profileID)")
    }

    /// Marque (spaceID non nil) ou retire (nil) une dépense commune.
    func setTransactionSpace(transactionID: String, spaceID: String?) async throws {
        struct Req: Encodable {
            let spaceID: String
            enum CodingKeys: String, CodingKey { case spaceID = "space_id" }
        }
        let _: EmptyResponse = try await request("PUT", "/v1/transactions/\(transactionID)/space",
                                                 body: Req(spaceID: spaceID ?? ""))
    }

    func fxRates() async throws -> (rates: [FXRate], unrated: [String]) {
        struct Env: Decodable {
            let rates: [FXRate]?
            let unrated: [String]?
        }
        let env: Env = try await request("GET", "/v1/fx/")
        return (env.rates ?? [], env.unrated ?? [])
    }

    func upsertFXRate(currency: String, rateMicro: Int64) async throws {
        struct Req: Encodable {
            let rateMicro: Int64
            enum CodingKeys: String, CodingKey { case rateMicro = "rate_micro" }
        }
        let _: FXRate = try await request("PUT", "/v1/fx/\(currency)", body: Req(rateMicro: rateMicro))
    }

    func deleteFXRate(currency: String) async throws {
        let _: EmptyResponse = try await request("DELETE", "/v1/fx/\(currency)")
    }

    // MARK: - Split multi-catégories (EF-024)

    struct SplitPart: Encodable {
        let amountCents: Int64
        let categoryID: String
        let label: String
        enum CodingKeys: String, CodingKey {
            case label
            case amountCents = "amount_cents"
            case categoryID = "category_id"
        }
    }

    func splitTransaction(id: String, parts: [SplitPart]) async throws -> [Transaction] {
        struct Req: Encodable { let parts: [SplitPart] }
        struct Env: Decodable { let transactions: [Transaction]? }
        let env: Env = try await request("POST", "/v1/transactions/\(id)/split", body: Req(parts: parts))
        return env.transactions ?? []
    }

    // MARK: - Journal d'accès & export (audit)

    func accessLog() async throws -> [AccessEvent] {
        struct Env: Decodable { let events: [AccessEvent]? }
        let env: Env = try await request("GET", "/v1/access-log")
        return env.events ?? []
    }

    /// Réinitialise TOUTES les données du profil (confirmation = nom exact).
    func resetData(confirmName: String) async throws {
        let _: EmptyResponse = try await request(
            "DELETE", "/v1/me/data",
            query: [URLQueryItem(name: "confirm", value: confirmName)]
        )
    }

    /// Supprime définitivement le profil (confirmation = nom exact).
    func deleteProfile(confirmName: String) async throws {
        let _: EmptyResponse = try await request(
            "DELETE", "/v1/me",
            query: [URLQueryItem(name: "confirm", value: confirmName)]
        )
    }

    /// Télécharge l'export complet (EF-006) — ZIP en octets bruts.
    func exportData() async throws -> Data {
        guard let token = tokenProvider() else { throw APIError.notAuthenticated }
        var req = URLRequest(url: baseURL.appending(path: "/v1/export"))
        req.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        let (data, response) = try await URLSession.shared.data(for: req)
        guard let http = response as? HTTPURLResponse,
              (200..<300).contains(http.statusCode) else {
            throw APIError.invalidResponse
        }
        return data
    }
}
