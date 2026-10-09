import XCTest
@testable import Opale

final class FinancialJourneyTests: XCTestCase {
    @MainActor private func snapshot() -> FinancialJourneySnapshot {
        FinancialJourneySnapshot(state: FinancialJourneyState(), assets: [], liabilities: [], rules: [], contracts: [], incomes: [], goals: [], netWorth: NetWorth(assetsTotal: .zero, liabilitiesTotal: .zero, net: .zero, currency: "EUR", complete: true), alerts: [])
    }
    @MainActor func testOrderedProgressDistinguishesSkippedAndReviewedSteps() throws {
        var state = FinancialJourneyState()
        XCTAssertEqual(state.next, .accounts)
        state.mark(.accounts, skipped: false)
        state.mark(.income, skipped: true)
        XCTAssertEqual(state.next, .expenses)
        XCTAssertEqual(state.reviewed, ["accounts"])
        XCTAssertEqual(state.skipped, ["income"])
        state.mark(.income, skipped: false)
        XCTAssertTrue(state.skipped.isEmpty)
        for step in JourneyStep.allCases { state.mark(step, skipped: false) }
        XCTAssertTrue(state.finished)
        XCTAssertEqual(state.completedCount, 7)
        XCTAssertEqual(try JSONDecoder().decode(FinancialJourneyState.self, from: JSONEncoder().encode(state)), state)
    }
    @MainActor func testDeclaredBudgetIsExactAndRequiresExplicitReview() throws {
        var data = snapshot()
        var salary = VariableIncome(); salary.low = Cents(320000); salary.usual = salary.low; salary.high = salary.low; salary.forecast = "usual"
        var rent = FinancialContract(); rent.category = "housing"; rent.amount = Cents(85000)
        var subscription = FinancialContract(); subscription.amount = Cents(1099)
        data.incomes = [salary]; data.contracts = [rent, subscription]; data.state.dailyBudget = Cents(50000)
        XCTAssertNil(data.currencyRows[0].remaining)
        data.state.reviewed = ["income", "expenses", "subscriptions", "budget"]
        XCTAssertEqual(data.currencyRows[0].remaining?.raw, 183901)
        data.state.mark(.expenses, skipped: true)
        XCTAssertNil(data.currencyRows[0].remaining, "Skipped data is never assumed complete.")
    }
    @MainActor func testNoDoubleCountingManagedRulesOneOffsEndedSeriesOrForeignCurrencies() throws {
        var data = snapshot()
        data.assets = [Asset(id: "eur", name: "EUR", kind: .checking, currency: "EUR", note: "", archived: false), Asset(id: "jpy", name: "JPY", kind: .checking, currency: "JPY", note: "", archived: false)]
        let manual = CalendarRule(id: "salary", asset_id: "eur", label: "Salary", amount_cents: Cents(100000), date: "2026-01-01", frequency: "monthly", active: true, merchant_key: "")
        var ended = manual; ended.id = "old"; ended.end_date = "2020-01-01"
        var managed = manual; managed.id = "managed"; managed.managed_by = "income"
        var weekly = manual; weekly.id = "weekly"; weekly.amount_cents = Cents(-300); weekly.frequency = "weekly"
        data.rules = [manual, ended, managed, weekly]
        var variable = VariableIncome(); variable.low = Cents(60000); variable.usual = Cents(120000); variable.high = Cents(180000); variable.forecast = "prudent"; variable.frequency = "quarterly"
        var once = variable; once.frequency = "once"
        var excluded = variable; excluded.forecast = "off"
        var foreign = variable; foreign.currency = "JPY"
        data.incomes = [variable, once, excluded, foreign]
        data.state.dailyBudget = .zero
        data.state.reviewed = ["income", "expenses", "subscriptions", "budget"]
        let euro = try XCTUnwrap(data.currencyRows.first { $0.currency == "EUR" })
        XCTAssertEqual(euro.income?.raw, 120000)
        XCTAssertEqual(euro.charges?.raw, 1300)
        XCTAssertEqual(euro.remaining?.raw, 118700)
        let yen = try XCTUnwrap(data.currencyRows.first { $0.currency == "JPY" })
        XCTAssertEqual(yen.income?.raw, 20000)
        XCTAssertNil(yen.remaining, "No everyday budget has been provided in JPY.")
    }
    @MainActor func testAnnualRoundingOccursAfterSumAndExtremeAmountDoesNotCrash() throws {
        var data = snapshot()
        var annual = FinancialContract(); annual.frequency = "yearly"; annual.amount = Cents(100)
        data.contracts = Array(repeating: annual, count: 6)
        XCTAssertEqual(data.currencyRows[0].subscriptions?.raw, 50)
        data.assets = [Asset(id: "cash", name: "cash", kind: .checking, currency: "EUR", note: "", archived: false)]
        data.rules = [CalendarRule(id: "extreme", asset_id: "cash", label: "Extreme", amount_cents: Cents(Int64.min), date: "2026-01-01", frequency: "weekly", active: true, merchant_key: "")]
        XCTAssertNil(data.currencyRows[0].charges)
    }
    @MainActor func testStatusFromPreviousServerRemainsReadable() throws {
        let status = try JSONDecoder().decode(AssistantStatus.self, from: Data(#"{"homelab_available":false,"cloud_configured":false}"#.utf8))
        XCTAssertNil(status.homelabConfigured)
        XCTAssertNil(status.cloudAllowed)
    }
}
