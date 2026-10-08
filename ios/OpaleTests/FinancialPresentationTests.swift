import XCTest
@testable import Opale

final class FinancialPresentationTests: XCTestCase {
    @MainActor func testExactParsingAndOverflow() {
        XCTAssertEqual(Cents.parse("12 345,67")?.raw, 1_234_567)
        XCTAssertEqual(Cents.parse("-0,01")?.raw, -1)
        XCTAssertNil(Cents.parse("9223372036854775807"))
        XCTAssertNil(Cents.parse("1.234"))
        XCTAssertNil(Cents.parse("--1"))
    }
    @MainActor func testCurrencyMinorUnitsAndMalformedInput() {
        XCTAssertEqual(Cents.parse("123", currency: "JPY")?.raw, 123)
        XCTAssertNil(Cents.parse("1,5", currency: "JPY"))
        XCTAssertEqual(Cents.parse("1,234", currency: "KWD")?.raw, 1234)
        XCTAssertEqual(Cents.parse("1,23", currency: "EUR")?.raw, 123)
        XCTAssertNil(Cents.parse("1-2", currency: "EUR"))
        XCTAssertNil(Cents.parse("+", currency: "EUR"))
        XCTAssertNil(Cents.parse("92233720368547758.08", currency: "EUR"))
        XCTAssertEqual(MoneyFormat.input(Cents(1234), currency: "KWD"), "1.234")
    }
    @MainActor func testFrenchCivilDatesDoNotShiftAtMonthBoundaries() throws {
        let formatter = ISO8601DateFormatter()
        let march = try XCTUnwrap(formatter.date(from: "2026-03-31T22:30:00Z"))
        XCTAssertEqual(march.opaleDayString, "2026-04-01")
        let winter = try XCTUnwrap(formatter.date(from: "2026-12-31T23:30:00Z"))
        XCTAssertEqual(winter.opaleDayString, "2027-01-01")
        for day in ["2024-02-29", "2026-03-29", "2026-10-25"] {
            XCTAssertEqual(try XCTUnwrap(Date.fromOpaleDay(day)).opaleDayString, day)
        }
        XCTAssertNil(Date.fromOpaleDay("2026-02-29"))
        XCTAssertNil(Date.fromOpaleDay("2026-13-01"))
        let civil = try XCTUnwrap(Date.fromOpaleDay("2026-01-01"))
        XCTAssertEqual(civil.opaleFormatted(.dateTime.day().month(.wide).year()), "1 janvier 2026")
    }
    @MainActor func testDecoderAcceptsCivilAndInstantDates() throws {
        struct Item: Decodable { let date: Date }
        for value in ["2026-06-01", "2026-06-01T00:00:00Z", "2026-06-01T00:00:00.123Z"] {
            let item = try APIClient.makeDecoder().decode(Item.self, from: Data("{\"date\":\"\(value)\"}".utf8))
            XCTAssertEqual(item.date.opaleDayString, "2026-06-01")
        }
    }
    @MainActor func testCachedDataRemainsSeparatedByProfileAndServer() {
        let a = "test-profile-a-" + UUID().uuidString
        let b = "test-profile-b-" + UUID().uuidString
        DiskCache.save([Cents(123)], key: a)
        DiskCache.save([Cents(456)], key: b)
        XCTAssertEqual(DiskCache.load([Cents].self, key: a)?.value, [Cents(123)])
        XCTAssertEqual(DiskCache.load([Cents].self, key: b)?.value, [Cents(456)])
        XCTAssertNil(DiskCache.load([Cents].self, key: "different-server-" + a))
    }
    @MainActor func testColdStartAndInactivityLock() {
        let token = Keychain.get("session.token")
        let enabled = UserDefaults.standard.object(forKey: "lock.enabled")
        defer {
            if let token { Keychain.set(token, forKey: "session.token") } else { Keychain.delete("session.token") }
            if let enabled { UserDefaults.standard.set(enabled, forKey: "lock.enabled") } else { UserDefaults.standard.removeObject(forKey: "lock.enabled") }
        }
        XCTAssertTrue(Keychain.set("synthetic-test-token", forKey: "session.token"), "A locally signed simulator app is required for Keychain integration.")
        UserDefaults.standard.removeObject(forKey: "lock.enabled")
        let lock = AppLock()
        XCTAssertTrue(lock.enabled)
        XCTAssertTrue(lock.locked)
        lock.authenticatedByPIN()
        XCTAssertFalse(lock.locked)
        lock.checkInactivity(now: .now.addingTimeInterval(301))
        XCTAssertTrue(lock.locked)
    }

    @MainActor func testTransactionSuggestionRejectsInventedCategoryAndMalformedLabel() {
        let proposal = LocalAI.TransactionProposal(categoryID: "invented", merchantLabel: "Boulangerie", confident: true)
        XCTAssertNil(LocalAI.validate(proposal, allowedIDs: ["food"])?.categoryID)
        XCTAssertEqual(LocalAI.validate(proposal, allowedIDs: ["food"])?.label, "Boulangerie")
        let valid = LocalAI.TransactionProposal(categoryID: "food", merchantLabel: "Boulangerie", confident: true)
        XCTAssertEqual(LocalAI.validate(valid, allowedIDs: ["food"])?.categoryID, "food")
        let invalid = LocalAI.TransactionProposal(categoryID: "food", merchantLabel: "<script>", confident: true)
        XCTAssertNil(LocalAI.validate(invalid, allowedIDs: ["food"]))
    }
    @MainActor func testEmptyCashForecastDecodesAsEmptyList() throws {
        let data = Data(#"{"start_cash_cents":123,"end_cash_cents":123,"until":"2026-10-09","upcoming":null}"#.utf8)
        let cash = try APIClient.makeDecoder().decode(CashProjection.self, from: data)
        XCTAssertEqual(cash.startCash.raw, 123)
        XCTAssertTrue(cash.upcoming.isEmpty)
    }

}
