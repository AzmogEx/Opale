import XCTest

/// Manual journey against the disposable localhost database; no personal profiles.
final class FinancialSetupUITests: XCTestCase {
    private let baseURL = URL(string: "http://localhost:58088")!
    private let pin = "8164"

    @MainActor func testManualJourneyExplainsBudgetWithoutInventingTransactions() async throws {
        continueAfterFailure = false
        let app = XCUIApplication()
        let name = "Recette parcours " + UUID().uuidString.prefix(8)
        createProfile(app, name: String(name))
        let token = try await tokenForProfile(String(name))
        cleanup(name: String(name), token: token)
        capture(app, "Parcours-accueil")
        app.buttons["journey.continue"].tap()
        XCTAssertTrue(app.buttons["journey.addAccount"].waitForExistence(timeout: 10))
        app.buttons["journey.addAccount"].tap()
        fill(app.textFields["Nom (ex. Compte courant BNP)"], "Compte recette", app)
        fill(app.textFields["Ex. 12 500,00"], "2000", app)
        app.navigationBars.buttons["Ajouter"].tap()
        XCTAssertTrue(app.buttons.matching(NSPredicate(format: "label CONTAINS %@", "Compte recette")).firstMatch.waitForExistence(timeout: 15))
        next(app, expecting: "journey.addSalary")
        app.buttons["journey.addSalary"].tap()
        fill(app.textFields["fixedIncome.amount"], "3200", app)
        capture(app, "Parcours-salaire-simple")
        XCTAssertTrue(app.buttons["fixedIncome.save"].isEnabled)
        app.buttons["fixedIncome.save"].tap()
        XCTAssertTrue(app.buttons.matching(NSPredicate(format: "label CONTAINS %@", "Salaire net")).firstMatch.waitForExistence(timeout: 15))
        next(app, expecting: "journey.addExpense")
        app.buttons["journey.addExpense"].tap()
        fill(app.textFields["contract.name"], "Loyer recette", app)
        fill(app.textFields["contract.amount"], "850", app)
        app.buttons["contract.save"].tap()
        XCTAssertTrue(app.buttons.matching(NSPredicate(format: "label CONTAINS %@", "Loyer recette")).firstMatch.waitForExistence(timeout: 15))
        next(app, expecting: "journey.addSubscription")
        app.buttons["journey.addSubscription"].tap()
        fill(app.textFields["contract.name"], "Musique recette", app)
        fill(app.textFields["contract.amount"], "10,99", app)
        app.buttons["contract.save"].tap()
        XCTAssertTrue(app.buttons.matching(NSPredicate(format: "label CONTAINS %@", "Musique recette")).firstMatch.waitForExistence(timeout: 15))
        validate(app)
        XCTAssertTrue(app.buttons["Ajouter un bien ou placement"].waitForExistence(timeout: 15))
        validate(app)
        fill(app.textFields["journey.dailyBudget"], "500", app)
        validate(app)
        let remaining = app.staticTexts.matching(NSPredicate(format: "label CONTAINS %@", "839,01")).firstMatch
        reveal(remaining, app)
        XCTAssertTrue(remaining.waitForExistence(timeout: 15), "3200 − 850 − 10,99 − 500 = 1839,01 €")
        capture(app, "Parcours-bilan-explique")
        validate(app)
        XCTAssertTrue(app.buttons["journey.continue"].waitForExistence(timeout: 15))
        let progress = try await json("GET", "/v1/journey", token: token)
        XCTAssertEqual((progress["reviewed"] as? [String])?.count, 7)
        XCTAssertEqual((progress["daily_budget_cents"] as? NSNumber)?.int64Value, 50000)
        let transactions = try await json("GET", "/v1/transactions", token: token)
        XCTAssertEqual((transactions["transactions"] as? [[String: Any]])?.count, 0)
        let calendar = try await json("GET", "/v1/calendar", token: token)
        XCTAssertEqual((calendar["rules"] as? [[String: Any]])?.count, 3)
        let net = try await json("GET", "/v1/net-worth", token: token)
        XCTAssertEqual((net["net_cents"] as? NSNumber)?.int64Value, 200000)

        app.tabBars.buttons["Assistant"].tap()
        XCTAssertTrue(app.buttons["assistant.settings"].waitForExistence(timeout: 10))
        app.buttons["assistant.settings"].tap()
        XCTAssertTrue(app.navigationBars["Choisir mon IA"].waitForExistence(timeout: 10))
        capture(app, "IA-choix-PC-cloud")
        app.buttons["Fermer"].tap()
        let start = app.buttons["Par où commencer ?"]
        reveal(start, app); start.tap()
        XCTAssertTrue(app.buttons["Reprendre mon parcours"].waitForExistence(timeout: 15))
        app.buttons["Reprendre mon parcours"].tap()
        XCTAssertTrue(app.navigationBars["Mon parcours"].waitForExistence(timeout: 10))
    }

    @MainActor func testBudgetDraftAndProgressResumeAfterRelaunch() async throws {
        continueAfterFailure = false
        let app = XCUIApplication()
        let name = "Recette reprise " + UUID().uuidString.prefix(8)
        createProfile(app, name: String(name))
        let token = try await tokenForProfile(String(name))
        cleanup(name: String(name), token: token)
        app.buttons["journey.continue"].tap()
        for _ in 0..<5 {
            let skip = app.buttons["journey.skip"]
            reveal(skip, app); XCTAssertTrue(skip.waitForExistence(timeout: 10)); skip.tap()
        }
        fill(app.textFields["journey.dailyBudget"], "735,42", app)
        let before = try await json("GET", "/v1/journey", token: token)
        XCTAssertEqual((before["skipped"] as? [String])?.count, 5)
        XCTAssertTrue(before["daily_budget_cents"] is NSNull)
        capture(app, "Parcours-brouillon-budget")
        app.terminate()
        app.launchArguments = ["--reset-session", "--base-url", baseURL.absoluteString]
        app.launch()
        let profile = app.buttons[String(name)]
        reveal(profile, app); XCTAssertTrue(profile.waitForExistence(timeout: 15)); profile.tap()
        fill(app.secureTextFields["Code PIN"], pin, app)
        app.buttons["Déverrouiller"].tap()
        XCTAssertTrue(app.buttons["journey.continue"].waitForExistence(timeout: 15)); app.buttons["journey.continue"].tap()
        let budget = app.textFields["journey.dailyBudget"]
        XCTAssertTrue(budget.waitForExistence(timeout: 15)); XCTAssertEqual(budget.value as? String, "735,42")
        capture(app, "Parcours-repris-apres-fermeture")
        validate(app)
        let after = try await json("GET", "/v1/journey", token: token)
        XCTAssertEqual((after["daily_budget_cents"] as? NSNumber)?.int64Value, 73542)
        let pending = app.staticTexts.matching(NSPredicate(format: "label CONTAINS %@", "À compléter")).firstMatch
        reveal(pending, app); XCTAssertTrue(pending.exists, "Skipped income and charges must not produce a false remaining budget.")
    }

    @MainActor private func createProfile(_ app: XCUIApplication, name: String) {
        app.launchArguments = ["--reset-session", "--base-url", baseURL.absoluteString]
        app.launch()
        if app.buttons["Passer"].waitForExistence(timeout: 3) { app.buttons["Passer"].tap() }
        let create = app.buttons["Nouveau profil"]
        reveal(create, app); XCTAssertTrue(create.waitForExistence(timeout: 15)); create.tap()
        fill(app.textFields["Prénom"], name, app)
        fill(app.secureTextFields["PIN"], pin, app)
        fill(app.secureTextFields["Confirme le PIN"], pin, app)
        app.buttons["Créer"].tap()
        XCTAssertTrue(app.buttons["journey.continue"].waitForExistence(timeout: 20))
    }
    @MainActor private func fill(_ field: XCUIElement, _ value: String, _ app: XCUIApplication) {
        reveal(field, app); XCTAssertTrue(field.waitForExistence(timeout: 5)); field.tap(); field.typeText(value)
    }
    @MainActor private func reveal(_ element: XCUIElement, _ app: XCUIApplication) {
        _ = element.waitForExistence(timeout: 5)
        for _ in 0..<10 {
            if element.exists && element.isHittable { return }
            app.swipeUp()
        }
    }
    @MainActor private func validate(_ app: XCUIApplication) {
        if app.keyboards.firstMatch.exists && app.buttons["OK"].isHittable { app.buttons["OK"].tap() }
        let validate = app.buttons["journey.validate"]
        reveal(validate, app); XCTAssertTrue(validate.waitForExistence(timeout: 10)); XCTAssertTrue(validate.isEnabled); validate.tap()
    }
    @MainActor private func next(_ app: XCUIApplication, expecting identifier: String) {
        validate(app); XCTAssertTrue(app.buttons[identifier].waitForExistence(timeout: 15))
    }
    @MainActor private func capture(_ app: XCUIApplication, _ title: String) {
        let attachment = XCTAttachment(screenshot: app.screenshot()); attachment.name = title; attachment.lifetime = .keepAlways; add(attachment)
    }
    @MainActor private func tokenForProfile(_ name: String) async throws -> String {
        let list = try await json("GET", "/v1/profiles")
        let id = try XCTUnwrap((list["profiles"] as? [[String: Any]])?.first { ($0["name"] as? String) == name }?["id"] as? String)
        let result = try await json("POST", "/v1/auth/login", body: ["profile_id": id, "pin": pin])
        return try XCTUnwrap(result["token"] as? String)
    }
    @MainActor private func json(_ method: String, _ path: String, token: String? = nil, body: [String: Any]? = nil) async throws -> [String: Any] {
        var request = URLRequest(url: baseURL.appending(path: path)); request.httpMethod = method; request.timeoutInterval = 15
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        if let token { request.setValue("Bearer " + token, forHTTPHeaderField: "Authorization") }
        if let body { request.httpBody = try JSONSerialization.data(withJSONObject: body) }
        let (data, response) = try await URLSession.shared.data(for: request)
        XCTAssertTrue((200..<300).contains((response as? HTTPURLResponse)?.statusCode ?? 0), "\(method) \(path): \(String(decoding: data, as: UTF8.self))")
        return try XCTUnwrap(JSONSerialization.jsonObject(with: data) as? [String: Any])
    }
    @MainActor private func cleanup(name: String, token: String) {
        var components = URLComponents(url: baseURL.appending(path: "/v1/me"), resolvingAgainstBaseURL: false)!
        components.queryItems = [URLQueryItem(name: "confirm", value: name)]
        let target = components.url!
        addTeardownBlock {
            var request = URLRequest(url: target); request.httpMethod = "DELETE"; request.setValue("Bearer " + token, forHTTPHeaderField: "Authorization")
            let (_, response) = try await URLSession.shared.data(for: request)
            XCTAssertEqual((response as? HTTPURLResponse)?.statusCode, 204)
        }
    }
}
