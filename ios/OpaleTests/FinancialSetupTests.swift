import XCTest
@testable import Opale

final class FinancialSetupTests: XCTestCase {
    @MainActor func testMonthlyEstimateRoundsOnlyAfterAllAnnualizedAmounts() {
        var draft = FinancialSetupDraft()
        draft.income.amount = "100,00"
        draft.expenses = [FinancialSetupEntry(label: "Assurance", amount: "1,00", frequency: "quarterly")]
        draft.subscriptions = [FinancialSetupEntry(label: "Service", amount: "1,00", frequency: "yearly")]
        let summary = draft.summary
        XCTAssertEqual(summary.income?.raw, 10_000)
        XCTAssertEqual(summary.expenses?.raw, 33)
        XCTAssertEqual(summary.subscriptions?.raw, 8)
        // 100 − 1/3 − 1/12 = 99,5833… ; arrondir chaque charge avant donne à tort 99,59.
        XCTAssertEqual(summary.remaining?.raw, 9_958)
        XCTAssertTrue(summary.isComplete)
        XCTAssertFalse(summary.variableBudgetProvided)
    }

    @MainActor func testSmallAnnualSubscriptionsAreSummedBeforeRounding() {
        var draft = FinancialSetupDraft()
        draft.income.amount = "1 234,56"
        draft.subscriptions = (0..<6).map { FinancialSetupEntry(label: "Service \($0)", amount: "1,00", frequency: "yearly") }
        draft.variableBudget = "200,01"
        XCTAssertTrue(draft.summary.variableBudgetProvided)
        XCTAssertEqual(draft.summary.subscriptions?.raw, 50)
        XCTAssertEqual(draft.summary.remaining?.raw, 103_405)
        draft.expenses = (0..<3).map { FinancialSetupEntry(label: "Charge \($0)", amount: "0,01", frequency: "quarterly") }
        XCTAssertEqual(draft.summary.expenses?.raw, 1)
        XCTAssertEqual(draft.summary.remaining?.raw, 103_404)
    }

    @MainActor func testIncompleteOrInvalidInputsNeverCreateAPlausibleRemainingBudget() {
        var draft = FinancialSetupDraft()
        XCTAssertNil(draft.summary.remaining)
        XCTAssertFalse(draft.summary.isComplete)
        draft.income.amount = "2500"
        draft.expenses = [FinancialSetupEntry(label: "Loyer", amount: "")]
        XCTAssertNil(draft.summary.expenses)
        XCTAssertNil(draft.summary.remaining)
        draft.expenses[0].amount = "500,001"
        XCTAssertNil(draft.summary.remaining)
        draft.expenses[0].amount = "500"
        draft.expenses[0].frequency = "invalid"
        XCTAssertNil(draft.summary.remaining)
        draft.expenses = []
        draft.variableBudget = "-20"
        XCTAssertNil(draft.summary.remaining)
        draft.variableBudget = ""
        draft.income.enabled = false
        XCTAssertNil(draft.summary.income)
        XCTAssertNil(draft.summary.remaining)
        XCTAssertTrue(draft.summary.isComplete)
    }

    @MainActor func testServerLimitsAndFrenchAmountsAreValidatedExactly() {
        XCTAssertEqual(FinancialSetupMoney.positive("9 999 999 999,99")?.raw, 999_999_999_999)
        XCTAssertNil(FinancialSetupMoney.positive("10 000 000 000"))
        XCTAssertNil(FinancialSetupMoney.positive("0"))
        XCTAssertNil(FinancialSetupMoney.positive("-1"))
        XCTAssertNil(FinancialSetupMoney.positive("2,345"))
        XCTAssertNil(FinancialSetupMoney.parse("1e3"))
        XCTAssertEqual(FinancialSetupMoney.parse("-1 234,56")?.raw, -123_456)
        XCTAssertEqual(FinancialSetupMoney.optionalNonnegative("")?.raw, 0)
    }

    @MainActor func testPageValidationAllowsProgressButFinalValidationChecksAllPages() throws {
        let today = try XCTUnwrap(Date.fromOpaleDay("2026-10-09"))
        var draft = FinancialSetupDraft()
        draft.income.amount = "2500"
        draft.income.nextDate = "2026-10-28"
        draft.account.name = ""
        XCTAssertTrue(draft.validationErrors(forStep: 0, on: today).isEmpty)
        XCTAssertFalse(draft.validationErrors(forStep: 1, on: today).isEmpty)
        XCTAssertFalse(draft.validationErrors(on: today).isEmpty)
        draft.account.name = "Mon compte"
        draft.account.balance = "-100"
        draft.account.balanceDate = "2026-10-08"
        XCTAssertTrue(draft.validationErrors(on: today).isEmpty)
        draft.account.balanceDate = "2026-10-10"
        XCTAssertFalse(draft.validationErrors(on: today).isEmpty)
        draft.account.existingAssetID = "account-already-valued"
        draft.account.balance = "ignored"
        draft.account.name = ""
        XCTAssertTrue(draft.validationErrors(on: today).isEmpty)
        draft.account.enabled = false
        XCTAssertFalse(draft.validationErrors(on: today).isEmpty)
    }

    @MainActor func testInvalidDatesDuplicateRowsAndMissingLabelsAreRejected() throws {
        let today = try XCTUnwrap(Date.fromOpaleDay("2026-10-09"))
        var draft = FinancialSetupDraft()
        draft.income.enabled = false
        draft.expenses = [FinancialSetupEntry(label: "", amount: "12", date: "2026-02-30")]
        XCTAssertFalse(draft.validationErrors(forStep: 2, on: today).isEmpty)
        draft.expenses[0].label = "Assurance"
        draft.expenses[0].date = "2026-10-08"
        XCTAssertFalse(draft.validationErrors(forStep: 2, on: today).isEmpty)
        draft.expenses[0].date = "2026-10-09"
        XCTAssertTrue(draft.validationErrors(forStep: 2, on: today).isEmpty)
        draft.subscriptions = draft.expenses
        XCTAssertFalse(draft.validationErrors(on: today).isEmpty)
        draft.subscriptions = []
        draft.expenses[0].date = "2036-10-10"
        XCTAssertFalse(draft.validationErrors(forStep: 2, on: today).isEmpty)
    }

    @MainActor func testProtectedDraftCacheIsIsolatedByProfileAndServer() {
        let id = UUID().uuidString
        let first = "https://one.invalid|" + id
        let secondProfile = "https://one.invalid|" + UUID().uuidString
        let secondServer = "https://two.invalid|" + id
        defer { [first, secondProfile, secondServer].forEach { FinancialSetupCache.clear(profileKey: $0) } }
        var a = FinancialSetupDraft()
        a.income.amount = "1234,56"
        var b = FinancialSetupDraft()
        b.income.amount = "999"
        FinancialSetupCache.save(draft: a, step: 3, revision: 4, profileKey: first)
        FinancialSetupCache.save(draft: b, step: 1, revision: 8, profileKey: secondProfile)
        XCTAssertEqual(FinancialSetupCache.load(profileKey: first), FinancialSetupCachedDraft(draft: a, step: 3, revision: 4))
        XCTAssertEqual(FinancialSetupCache.load(profileKey: secondProfile)?.draft, b)
        XCTAssertNil(FinancialSetupCache.load(profileKey: secondServer))
        FinancialSetupCache.clear(profileKey: first)
        XCTAssertNil(FinancialSetupCache.load(profileKey: first))
        XCTAssertEqual(FinancialSetupCache.load(profileKey: secondProfile)?.draft, b)
    }
}
