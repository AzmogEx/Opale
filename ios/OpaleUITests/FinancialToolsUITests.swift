import XCTest

/// End-to-end journeys against the disposable localhost database only.
final class FinancialToolsUITests: XCTestCase {
    private let baseURL = URL(string: "http://localhost:58088")!
    private let pin = "8164"

    @MainActor func testContractsPriceConfirmationIncomeAndInvestmentExplorer() async throws {
        continueAfterFailure = false
        let app = XCUIApplication()
        let name = "Recette outils " + UUID().uuidString.prefix(8)
        app.launchArguments = ["--reset-session", "--base-url", baseURL.absoluteString]
        app.launch()
        if app.buttons["Passer"].waitForExistence(timeout: 3) { app.buttons["Passer"].tap() }
        reveal(app.buttons["Nouveau profil"], in: app)
        app.buttons["Nouveau profil"].tap()
        fill(app.textFields["Prénom"], text: String(name), in: app)
        fill(app.secureTextFields["PIN"], text: pin, in: app)
        fill(app.secureTextFields["Confirme le PIN"], text: pin, in: app)
        app.buttons["Créer"].tap()
        XCTAssertTrue(app.buttons["journey.continue"].waitForExistence(timeout: 15))
        let profiles = try await json("GET", "/v1/profiles")
        let items = try XCTUnwrap(profiles["profiles"] as? [[String: Any]])
        let id = try XCTUnwrap(items.first { ($0["name"] as? String) == String(name) }?["id"] as? String)
        let login = try await json("POST", "/v1/auth/login", body: ["profile_id": id, "pin": pin])
        let token = try XCTUnwrap(login["token"] as? String)
        cleanup(name: String(name), token: token)
        XCTAssertTrue(app.buttons["Réglages"].waitForExistence(timeout: 15))
        let account = try await json("POST", "/v1/assets", token: token, body: ["name": "Compte de recette", "kind": "checking", "currency": "EUR"])
        let assetID = try XCTUnwrap(account["id"] as? String)

        app.buttons["Réglages"].tap()
        app.buttons["Contrats et abonnements"].tap()
        XCTAssertTrue(app.buttons["contracts.add"].waitForExistence(timeout: 10))
        app.buttons["contracts.add"].tap()
        fill(app.textFields["contract.name"], text: "Musique recette", in: app)
        fill(app.textFields["contract.amount"], text: "10,99", in: app)
        capture(app, "Contrat-creation")
        XCTAssertTrue(app.buttons["contract.save"].isEnabled)
        app.buttons["contract.save"].tap()
        let row = app.buttons["contract.row.Musique recette"]
        XCTAssertTrue(row.waitForExistence(timeout: 15))
        let list = try await json("GET", "/v1/contracts", token: token)
        var contract = try XCTUnwrap((list["contracts"] as? [[String: Any]])?.first)
        XCTAssertEqual((contract["amount_cents"] as? NSNumber)?.int64Value, 1099)
        contract["asset_id"] = assetID; contract["merchant_key"] = "MUSIC"
        let contractID = try XCTUnwrap(contract["id"] as? String)
        _ = try await json("PUT", "/v1/contracts/\(contractID)", token: token, body: contract)
        let today = Date.now.formatted(.iso8601.year().month().day().dateSeparator(.dash))
        _ = try await json("POST", "/v1/transactions", token: token, body: ["asset_id": assetID, "amount_cents": -1299, "occurred_on": today, "label": "MUSIC"])
        reveal(row, in: app); row.tap()
        let accept = app.buttons["contract.acceptPrice"]
        XCTAssertTrue(accept.waitForExistence(timeout: 15))
        reveal(accept, in: app)
        capture(app, "Contrat-hausse-a-confirmer")
        accept.tap()
        app.alerts.buttons["Confirmer"].tap()
        XCTAssertTrue(app.staticTexts["Historique des tarifs"].waitForExistence(timeout: 10) || app.navigationBars["Musique recette"].exists)
        let updated = try await json("GET", "/v1/contracts", token: token)
        let confirmed = try XCTUnwrap((updated["contracts"] as? [[String: Any]])?.first)
        XCTAssertEqual((confirmed["amount_cents"] as? NSNumber)?.int64Value, 1299)
        let prices = try await json("GET", "/v1/contracts/\(contractID)/prices", token: token)
        XCTAssertEqual((prices["prices"] as? [[String: Any]])?.count, 2)
        app.navigationBars.buttons["Contrats"].firstMatch.tap()
        app.navigationBars.buttons["Réglages"].firstMatch.tap()

        app.buttons["Revenus variables"].tap()
        XCTAssertTrue(app.buttons["incomes.add"].waitForExistence(timeout: 10))
        app.buttons["incomes.add"].tap()
        fill(app.textFields["income.name"], text: "Mission recette", in: app)
        fill(app.textFields["income.usual"], text: "1000", in: app)
        fill(app.textFields["income.high"], text: "2000", in: app)
        capture(app, "Revenu-fourchette")
        XCTAssertTrue(app.buttons["income.save"].isEnabled)
        app.buttons["income.save"].tap()
        let incomeRow = app.buttons["income.row.Mission recette"]
        XCTAssertTrue(incomeRow.waitForExistence(timeout: 15)); reveal(incomeRow, in: app)
        capture(app, "Revenus-variables")
        let incomes = try await json("GET", "/v1/incomes/variable", token: token)
        let income = try XCTUnwrap((incomes["incomes"] as? [[String: Any]])?.first)
        XCTAssertEqual((income["low_cents"] as? NSNumber)?.int64Value, 0)
        XCTAssertEqual((income["usual_cents"] as? NSNumber)?.int64Value, 100000)
        XCTAssertEqual(income["forecast"] as? String, "off")
        app.navigationBars.buttons["Réglages"].firstMatch.tap()
        app.buttons["Explorer les investissements"].tap()
        let highRisk = app.segmentedControls["invest.risk"].buttons["Élevé"]
        XCTAssertTrue(highRisk.waitForExistence(timeout: 10)); highRisk.tap()
        let idea = app.buttons["invest.idea.world-pea"]
        reveal(idea, in: app)
        XCTAssertTrue(idea.exists); idea.tap()
        XCTAssertTrue(app.switches["invest.save"].waitForExistence(timeout: 10))
        app.switches["invest.save"].coordinate(withNormalizedOffset: CGVector(dx: 0.92, dy: 0.5)).tap()
        capture(app, "Investissement-fiche-WPEA")
        let source = app.descendants(matching: .any).matching(NSPredicate(format: "label == %@", "Émetteur · Fiche officielle")).firstMatch
        reveal(source, in: app)
        XCTAssertTrue(source.exists)
        capture(app, "Investissement-sources-et-risques")
        let txs = try await json("GET", "/v1/transactions", token: token)
        XCTAssertEqual((txs["transactions"] as? [[String: Any]])?.count, 1, "Only the synthetic bank payment may exist.")
    }
    @MainActor private func fill(_ field: XCUIElement, text: String, in app: XCUIApplication) {
        reveal(field, in: app); XCTAssertTrue(field.waitForExistence(timeout: 5)); field.tap(); field.typeText(text)
    }
    @MainActor private func reveal(_ element: XCUIElement, in app: XCUIApplication) {
        for _ in 0..<12 where !element.isHittable { app.swipeUp() }
    }
    @MainActor private func capture(_ app: XCUIApplication, _ name: String) {
        let attachment = XCTAttachment(screenshot: app.screenshot()); attachment.name = name; attachment.lifetime = .keepAlways; add(attachment)
    }
    @MainActor private func json(_ method: String, _ path: String, token: String? = nil, body: [String: Any]? = nil) async throws -> [String: Any] {
        var request = URLRequest(url: baseURL.appending(path: path)); request.httpMethod = method; request.timeoutInterval = 15
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        if let token { request.setValue("Bearer " + token, forHTTPHeaderField: "Authorization") }
        if let body { request.httpBody = try JSONSerialization.data(withJSONObject: body) }
        let (data, response) = try await URLSession.shared.data(for: request)
        let http = try XCTUnwrap(response as? HTTPURLResponse)
        XCTAssertTrue((200..<300).contains(http.statusCode), "\(method) \(path) → \(http.statusCode) \(String(decoding: data, as: UTF8.self))")
        return try XCTUnwrap(JSONSerialization.jsonObject(with: data) as? [String: Any])
    }
    @MainActor private func cleanup(name: String, token: String) {
        var url = URLComponents(url: baseURL.appending(path: "/v1/me"), resolvingAgainstBaseURL: false)!
        url.queryItems = [URLQueryItem(name: "confirm", value: name)]
        let target = url.url!
        addTeardownBlock {
            var request = URLRequest(url: target); request.httpMethod = "DELETE"; request.setValue("Bearer " + token, forHTTPHeaderField: "Authorization")
            let (_, response) = try await URLSession.shared.data(for: request)
            XCTAssertEqual((response as? HTTPURLResponse)?.statusCode, 204)
        }
    }
}
