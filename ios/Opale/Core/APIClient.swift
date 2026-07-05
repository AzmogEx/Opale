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

    func createAsset(name: String, kind: AssetKind, note: String = "") async throws -> Asset {
        try await request(
            "POST", "/v1/assets/",
            body: UpsertAssetRequest(name: name, kind: kind.rawValue, note: note)
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
}
