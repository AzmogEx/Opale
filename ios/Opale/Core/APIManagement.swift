import Foundation

struct CategorizationRule: Codable, Identifiable {
    let id: String
    var merchantKey: String
    var categoryID: String
    enum CodingKeys: String, CodingKey { case id; case merchantKey = "merchant_key"; case categoryID = "category_id" }
}
extension APIClient {
    struct MetadataRequest: Encodable { let name: String; let note: String; let archived: Bool }
    func updateHolding(id: String, liability: Bool, name: String, note: String, archived: Bool) async throws {
        let _: EmptyResponse = try await request("PATCH", "/v1/\(liability ? "liabilities" : "assets")/\(id)/", body: MetadataRequest(name: name, note: note, archived: archived))
    }
    func updateValuation(id: String, amount: Int64, date: String) async throws {
        let _: EmptyResponse = try await request("PATCH", "/v1/valuations/\(id)", body: AddValuationRequest(valueCents: amount, asOf: date))
    }
    func deleteValuation(id: String) async throws { let _: EmptyResponse = try await request("DELETE", "/v1/valuations/\(id)") }
    func updateGoal(id: String, request body: CreateGoalRequest) async throws { let _: EmptyResponse = try await request("PATCH", "/v1/goals/\(id)", body: body) }
    func updateContact(id: String, request body: ContactRequest) async throws { let _: EmptyResponse = try await request("PATCH", "/v1/contacts/\(id)", body: body) }
    func updateDocument(id: String, name: String, kind: String, assetID: String) async throws {
        struct Body: Encodable { let name: String; let kind: String; let asset_id: String }
        let _: EmptyResponse = try await request("PATCH", "/v1/documents/\(id)", body: Body(name: name, kind: kind, asset_id: assetID))
    }
    func saveCategory(id: String?, name: String, icon: String) async throws {
        struct Body: Encodable { let name: String; let icon: String }
        let _: EmptyResponse = try await request(id == nil ? "POST" : "PATCH", "/v1/categories" + (id.map { "/\($0)" } ?? ""), body: Body(name: name, icon: icon))
    }
    func deleteCategory(id: String) async throws { let _: EmptyResponse = try await request("DELETE", "/v1/categories/\(id)") }
    func categorizationRules() async throws -> [CategorizationRule] {
        struct Envelope: Decodable { let rules: [CategorizationRule] }
        let result: Envelope = try await request("GET", "/v1/rules"); return result.rules
    }
    func saveRule(id: String?, merchant: String, categoryID: String) async throws {
        struct Body: Encodable { let merchant_key: String; let category_id: String }
        let _: EmptyResponse = try await request(id == nil ? "POST" : "PATCH", "/v1/rules" + (id.map { "/\($0)" } ?? ""), body: Body(merchant_key: merchant, category_id: categoryID))
    }
    func deleteRule(id: String) async throws { let _: EmptyResponse = try await request("DELETE", "/v1/rules/\(id)") }
}
