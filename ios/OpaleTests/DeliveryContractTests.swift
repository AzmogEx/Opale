import XCTest
@testable import Opale

final class DeliveryContractTests: XCTestCase {
    @MainActor func testSyntheticAPIContractsDecodeAndMutate() async throws {
        guard let value = ProcessInfo.processInfo.environment["OPALE_TEST_API_URL"], let base = URL(string: value) else { throw XCTSkip("Requires disposable delivery API") }
        let anonymous = APIClient(baseURL: base) { nil }
        struct Demo: Decodable { let profile: Profile; let token: String }
        let demo: Demo = try await anonymous.request("POST", "/v1/profiles/demo", authenticated: false)
        let api = APIClient(baseURL: base) { demo.token }
        let assets = try await api.listAssets()
        XCTAssertFalse(assets.isEmpty)
        let originalWorth = try await api.netWorth()
        XCTAssertNotNil(originalWorth.complete)
        let unknownValue = try await api.createAsset(name: "iOS valeur inconnue", kind: .other)
        let incompleteWorth = try await api.netWorth()
        XCTAssertEqual(incompleteWorth.complete, false)
        XCTAssertEqual(incompleteWorth.missingValuations, (originalWorth.missingValuations ?? 0) + 1)
        try await api.deleteAsset(id: unknownValue.id)
        _ = try await api.netWorthHistory(months: 12)
        _ = try await api.cashflow(days: 1)
        _ = try await api.cashflow(days: 90)
        _ = try await api.snapshots()
        _ = try await api.wrapped(year: 2026)
        _ = try await api.taxEstimate(incomeCents: 4_000_000, partsTenths: 10, perCents: 100_000, perCeilingCents: 50_000)
        _ = try await api.investments()
        _ = try await api.listGoals()
        _ = try await api.bankStatus()
        struct Accounts: Decodable { let accounts: [BankAccount] }
        let _: Accounts = try await api.request("GET", "/v1/bank/accounts")
        _ = try await api.allocation()
        _ = try await api.customAlerts()
        struct Compare: Encodable {
            let kind = "cash_credit"
            let horizon_months = 120
            let loan_months = 120
            let initial_cash_cents: Int64 = 10_000_000
            let monthly_budget_cents: Int64 = 200_000
            let asset_price_cents: Int64 = 2_000_000
            let down_payment_cents: Int64 = 500_000
            let loan_rate_bps = 350
            let investment_return_bps = 400
        }
        let comparison: DecisionComparisonResult = try await api.request("POST", "/v1/decisions/compare", body: Compare())
        XCTAssertFalse(comparison.sensitivity.isEmpty)
        XCTAssertEqual(comparison.timeline?.map(\.months), [0, 60, 120])
        XCTAssertEqual(Set(comparison.scenarios?.map(\.name) ?? []), Set(["prudent", "normal", "ambitieux"]))
        XCTAssertNotNil(comparison.recommendation)
        XCTAssertNotNil(comparison.risks)
        struct CalendarEnvelope: Decodable { let rules: [CalendarRule]; let occurrences: [CalendarOccurrence] }
        let _: CalendarEnvelope = try await api.request("GET", "/v1/calendar")
        if let asset = assets.first {
            _ = try await api.assetValuations(assetID: asset.id)
            let _: QuotesToolView.Metadata = try await api.request("GET", "/v1/assets/\(asset.id)/quote")
        }
        // The same request types used by the forms are exercised against the real API.
        let yesterday = Calendar.opale.date(byAdding: .day, value: -1, to: .now)!.opaleDayString
        let requestID = UUID().uuidString
        let account = try await api.createAsset(name: "iOS contract account", kind: .checking, initialValue: 100_000, initialAsOf: yesterday, requestID: requestID)
        let duplicate = try await api.createAsset(name: "iOS contract account", kind: .checking, initialValue: 100_000, initialAsOf: yesterday, requestID: requestID)
        XCTAssertEqual(account.id, duplicate.id)
        let transaction = try await api.createTransaction(.init(assetID: account.id, amountCents: -12_345, occurredOn: Date.now.opaleDayString, label: "iOS mutation", categoryID: "", note: ""))
        let reloaded = try await api.listAssets().first { $0.id == account.id }
        XCTAssertEqual(reloaded?.currentValue?.raw, 87_655)
        var patch = APIClient.PatchTransactionRequest()
        patch.amountCents = -10_000
        _ = try await api.updateTransaction(id: transaction.id, patch)
        let afterEdit = try await api.listAssets().first { $0.id == account.id }
        XCTAssertEqual(afterEdit?.currentValue?.raw, 90_000)
        try await api.deleteTransaction(id: transaction.id)
        let liability = try await api.createLiability(name: "iOS principal", kind: .consumerLoan, initialValue: 100_000, initialAsOf: yesterday)
        let repayment = try await api.createTransaction(.init(flowKind: "loan_principal", linkedLiabilityID: liability.id, assetID: account.id, amountCents: -10_000, occurredOn: Date.now.opaleDayString, label: "Remboursement principal", categoryID: "", note: ""))
        XCTAssertEqual(repayment.linkedLiabilityID, liability.id)
        let debt = try await api.listLiabilities().first { $0.id == liability.id }
        XCTAssertEqual(debt?.latestValue?.raw, 90_000)
        try await api.deleteTransaction(id: repayment.id)
        let rule = CalendarRule(id: "", asset_id: account.id, label: "iOS échéance", amount_cents: Cents(-123), date: Date.now.opaleDayString, frequency: "quarterly", end_date: nil, active: true, merchant_key: "")
        let savedRule: CalendarRule = try await api.request("POST", "/v1/calendar", body: rule)
        XCTAssertFalse(savedRule.id.isEmpty)
        XCTAssertEqual(savedRule.frequency, "quarterly")
        let _: APIClient.EmptyResponse = try await api.request("DELETE", "/v1/calendar/\(savedRule.id)")
        let investment = try await api.createAsset(name: "iOS placement", kind: .pea, initialValue: 100_000, initialAsOf: yesterday)
        _ = try await api.addAssetValuation(assetID: investment.id, valueCents: 150_000, asOf: Date.now.opaleDayString)
        let flow = InvestmentFlow(id: "", asset_id: investment.id, kind: "contribution", amount_cents: Cents(50_000), occurred_on: Date.now.opaleDayString, note: "Apport exact")
        let _: InvestmentFlow = try await api.request("POST", "/v1/assets/\(investment.id)/investment/flows", body: flow)
        struct Coverage: Encodable { let complete = true }
        let _: APIClient.EmptyResponse = try await api.request("PUT", "/v1/assets/\(investment.id)/investment/coverage", body: Coverage())
        let performance: InvestmentDetail = try await api.request("GET", "/v1/assets/\(investment.id)/investment")
        XCTAssertTrue(performance.performance.known)
        XCTAssertEqual(performance.performance.gain_cents.raw, 0)
        let contact = try await api.createContact(.init(name: "iOS proche", role: "trusted", phone: "", email: "", note: ""))
        try await api.updateContact(id: contact.id, request: .init(name: "iOS proche modifié", role: "trusted", phone: "", email: "", note: ""))
        let doc = try await api.createDocument(.init(name: "iOS-contract.txt", kind: "other", mime: "text/plain", assetID: account.id, contentBase64: Data("Synthetic document".utf8).base64EncodedString()))
        try await api.updateDocument(id: doc.id, name: "iOS-renamed.txt", kind: "other", assetID: account.id)
        let downloaded = try await api.documentContent(id: doc.id)
        XCTAssertEqual(downloaded, Data("Synthetic document".utf8))
        let _: APIClient.EmptyResponse = try await api.request("PUT", "/v1/beneficiaries", body: Beneficiary(id: "", contact_id: contact.id, document_id: doc.id, share_bps: 3333, note: ""))
        struct Beneficiaries: Decodable { let beneficiaries: [Beneficiary] }
        let beneficiaries: Beneficiaries = try await api.request("GET", "/v1/beneficiaries")
        XCTAssertEqual(beneficiaries.beneficiaries.first { $0.contact_id == contact.id && $0.document_id == doc.id }?.share_bps, 3333)

    }
}
