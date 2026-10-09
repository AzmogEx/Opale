import XCTest

/// Recette isolée : uniquement l'API synthétique localhost:58088.
/// Chaque test crée un nouveau profil par l'interface, puis supprime ce seul profil.
final class FinancialSetupUITests: XCTestCase {
    private let baseURL = URL(string: "http://localhost:58088")!
    private let pin = "8164"

    @MainActor func testNewProfileCompletesSetupWithoutInventingTransactions() async throws {
        continueAfterFailure = false
        let app = XCUIApplication()
        let name = "Recette formulaire " + UUID().uuidString.prefix(8)
        createProfile(app, name: String(name))
        let token = try await tokenForNewProfile(named: String(name))
        registerCleanup(name: String(name), token: token)

        XCTAssertTrue(app.textFields["setup.income.amount"].waitForExistence(timeout: 15), "Le formulaire s'ouvre après la création du profil.")
        fill(app, field: app.textFields["setup.income.amount"], text: "3200")
        capture(app, name: "Setup-01-revenus")
        next(app, expecting: "Ton compte principal")
        fill(app, field: app.textFields["setup.account.balance"], text: "2000")
        capture(app, name: "Setup-02-compte")
        next(app, expecting: "Tes charges fixes")

        app.buttons["setup.add.expense"].tap()
        app.buttons["Loyer"].tap()
        fill(app, field: entryAmount(app, kind: "expense"), text: "850")
        capture(app, name: "Setup-03-charges")
        next(app, expecting: "Tes abonnements")
        app.buttons["setup.add.subscription"].tap()
        app.buttons["Spotify"].tap()
        fill(app, field: entryAmount(app, kind: "subscription"), text: "10,99")
        capture(app, name: "Setup-04-abonnements")
        next(app, expecting: "Ton budget et ton objectif")

        fill(app, field: app.textFields["setup.variable"], text: "500")
        dismissKeyboard(app)
        reveal(app.switches["setup.goal.enabled"], in: app)
        // SwiftUI expose toute la ligne comme Switch ; viser le contrôle à
        // droite évite de toucher seulement le libellé dans le Form iOS 26.
        app.switches["setup.goal.enabled"].coordinate(withNormalizedOffset: CGVector(dx: 0.92, dy: 0.5)).tap()
        XCTAssertTrue(app.textFields["setup.goal.name"].waitForExistence(timeout: 5))
        fill(app, field: app.textFields["setup.goal.name"], text: "Réserve de sécurité")
        fill(app, field: app.textFields["setup.goal.target"], text: "3000")
        capture(app, name: "Setup-05-budget-objectif")
        next(app, expecting: "Ton point de départ")
        XCTAssertTrue(app.staticTexts.matching(NSPredicate(format: "label CONTAINS %@", "839,01")).firstMatch.waitForExistence(timeout: 5), "3 200 − 850 − 10,99 − 500 = 1 839,01 €.")
        capture(app, name: "Setup-06-recapitulatif")
        app.buttons["setup.next"].tap()
        XCTAssertTrue(app.buttons["Réglages"].waitForExistence(timeout: 20))
        XCTAssertFalse(app.buttons["setup.next"].exists)

        let setup = try await json("GET", "/v1/onboarding", token: token)
        XCTAssertEqual(setup["status"] as? String, "completed")
        XCTAssertEqual(setup["should_prompt"] as? Bool, false)
        let result = try XCTUnwrap(setup["result"] as? [String: Any])
        XCTAssertEqual((result["expense_rule_ids"] as? [String])?.count, 1)
        XCTAssertEqual((result["subscription_rule_ids"] as? [String])?.count, 1)
        XCTAssertFalse((result["goal_id"] as? String ?? "").isEmpty)
        let transactions = try await json("GET", "/v1/transactions/", token: token)
        XCTAssertEqual((transactions["transactions"] as? [[String: Any]])?.count, 0, "Les déclarations ne doivent créer aucune opération réalisée.")
        let assets = try await json("GET", "/v1/assets/", token: token)
        XCTAssertEqual((assets["assets"] as? [[String: Any]])?.count, 1)
        let calendar = try await json("GET", "/v1/calendar", token: token)
        let rules = try XCTUnwrap(calendar["rules"] as? [[String: Any]])
        XCTAssertEqual(rules.count, 3)
        XCTAssertEqual(rules.compactMap { ($0["amount_cents"] as? NSNumber)?.int64Value }.sorted(), [-85_000, -1_099, 320_000])
        capture(app, name: "Setup-07-accueil-configure")

        // Le formulaire terminé est consultable depuis Réglages, sans repartir à zéro.
        openSetupFromSettings(app)
        XCTAssertTrue(app.staticTexts["Ton point de départ est enregistré"].waitForExistence(timeout: 10))
        XCTAssertFalse(app.buttons["setup.next"].exists)
    }

    @MainActor func testDraftResumesFromSettingsAndAfterRelaunch() async throws {
        continueAfterFailure = false
        let app = XCUIApplication()
        let name = "Recette reprise " + UUID().uuidString.prefix(8)
        createProfile(app, name: String(name))
        let token = try await tokenForNewProfile(named: String(name))
        registerCleanup(name: String(name), token: token)
        fill(app, field: app.textFields["setup.income.amount"], text: "2800")
        next(app, expecting: "Ton compte principal")
        fill(app, field: app.textFields["setup.account.balance"], text: "900")
        dismissKeyboard(app)
        app.buttons["setup.later"].tap()
        XCTAssertTrue(app.buttons["Réglages"].waitForExistence(timeout: 15))
        let postponed = try await json("GET", "/v1/onboarding", token: token)
        XCTAssertEqual(postponed["status"] as? String, "skipped")
        XCTAssertEqual(postponed["step"] as? Int, 1)

        openSetupFromSettings(app)
        let balance = app.textFields["setup.account.balance"]
        XCTAssertTrue(balance.waitForExistence(timeout: 10))
        XCTAssertEqual(balance.value as? String, "900")
        capture(app, name: "Setup-reprise-reglages")
        next(app, expecting: "Tes charges fixes")
        app.buttons["setup.add.expense"].tap()
        app.buttons["Loyer"].tap()
        fill(app, field: entryAmount(app, kind: "expense"), text: "735,42")
        dismissKeyboard(app)
        capture(app, name: "Setup-brouillon-local-avant-relance")
        let serverBefore = try await json("GET", "/v1/onboarding", token: token)
        let beforeDraft = try XCTUnwrap(serverBefore["draft"] as? [String: Any])
        XCTAssertEqual((beforeDraft["expenses"] as? [[String: Any]])?.count, 0, "La page en cours n'a pas encore été envoyée ; la reprise doit lire le brouillon local.")

        app.terminate()
        app.launchArguments = ["--base-url", baseURL.absoluteString]
        app.launch()
        XCTAssertTrue(app.staticTexts["Opale est verrouillée"].waitForExistence(timeout: 15))
        XCTAssertFalse(entryAmount(app, kind: "expense").exists)
        capture(app, name: "Setup-brouillon-protege-au-redemarrage")
        // La biométrie du simulateur dépend de son enrôlement. Réauthentifier
        // le même profil par PIN via le flag de test qui retire seulement le
        // jeton Keychain ; aucun effacement du cache ni logout n'est effectué.
        app.terminate()
        launchLoggedOut(app)
        let profileButton = app.buttons[String(name)]
        reveal(profileButton, in: app)
        XCTAssertTrue(profileButton.waitForExistence(timeout: 10))
        profileButton.tap()
        let pinField = app.secureTextFields["Code PIN"]
        XCTAssertTrue(pinField.waitForExistence(timeout: 5))
        pinField.tap()
        pinField.typeText(pin)
        app.buttons["Déverrouiller"].tap()
        let resumedAmount = entryAmount(app, kind: "expense")
        XCTAssertTrue(resumedAmount.waitForExistence(timeout: 15))
        XCTAssertEqual(resumedAmount.value as? String, "735,42")
        capture(app, name: "Setup-brouillon-repris-apres-relance")
        app.buttons["setup.later"].tap()
        XCTAssertTrue(app.buttons["Réglages"].waitForExistence(timeout: 15))
        let after = try await json("GET", "/v1/onboarding", token: token)
        let afterDraft = try XCTUnwrap(after["draft"] as? [String: Any])
        let entries = try XCTUnwrap(afterDraft["expenses"] as? [[String: Any]])
        XCTAssertEqual(entries.count, 1)
        XCTAssertEqual(entries.first?["amount"] as? String, "735,42")
        let transactions = try await json("GET", "/v1/transactions/", token: token)
        XCTAssertEqual((transactions["transactions"] as? [[String: Any]])?.count, 0)
    }

    // MARK: - Interface

    @MainActor private func launchLoggedOut(_ app: XCUIApplication) {
        app.launchArguments = ["--reset-session", "--base-url", baseURL.absoluteString]
        app.launch()
        if app.buttons["Passer"].waitForExistence(timeout: 3) { app.buttons["Passer"].tap() }
        XCTAssertTrue(app.navigationBars["Opale"].waitForExistence(timeout: 15))
    }

    @MainActor private func createProfile(_ app: XCUIApplication, name: String) {
        launchLoggedOut(app)
        let create = app.buttons["Nouveau profil"]
        reveal(create, in: app)
        XCTAssertTrue(create.waitForExistence(timeout: 10))
        create.tap()
        fill(app, field: app.textFields["Prénom"], text: name)
        fill(app, field: app.secureTextFields["PIN"], text: pin)
        fill(app, field: app.secureTextFields["Confirme le PIN"], text: pin)
        app.buttons["Créer"].tap()
        XCTAssertTrue(app.textFields["setup.income.amount"].waitForExistence(timeout: 15))
    }

    @MainActor private func fill(_ app: XCUIApplication, field: XCUIElement, text: String) {
        reveal(field, in: app)
        XCTAssertTrue(field.waitForExistence(timeout: 5))
        field.tap()
        field.typeText(text)
    }

    @MainActor private func next(_ app: XCUIApplication, expecting title: String) {
        dismissKeyboard(app)
        let next = app.buttons["setup.next"]
        XCTAssertTrue(next.waitForExistence(timeout: 5))
        next.tap()
        XCTAssertTrue(app.staticTexts[title].waitForExistence(timeout: 15), title)
    }

    @MainActor private func dismissKeyboard(_ app: XCUIApplication) {
        if app.keyboards.firstMatch.exists, app.buttons["OK"].isHittable { app.buttons["OK"].tap() }
    }

    @MainActor private func entryAmount(_ app: XCUIApplication, kind: String) -> XCUIElement {
        app.textFields.matching(NSPredicate(format: "identifier BEGINSWITH %@", "setup.\(kind).amount.")).firstMatch
    }

    @MainActor private func reveal(_ element: XCUIElement, in app: XCUIApplication) {
        for _ in 0..<10 where !element.isHittable { app.swipeUp() }
    }

    @MainActor private func openSetupFromSettings(_ app: XCUIApplication) {
        app.buttons["Réglages"].tap()
        let setup = app.buttons["Ma situation de départ"]
        reveal(setup, in: app)
        XCTAssertTrue(setup.waitForExistence(timeout: 10))
        setup.tap()
    }

    @MainActor private func capture(_ app: XCUIApplication, name: String) {
        let screenshot = XCTAttachment(screenshot: app.screenshot())
        screenshot.name = name
        screenshot.lifetime = .keepAlways
        add(screenshot)
    }

    // MARK: - Vérifications serveur et nettoyage du seul profil de recette

    @MainActor private func tokenForNewProfile(named name: String) async throws -> String {
        let profiles = try await json("GET", "/v1/profiles")
        let items = try XCTUnwrap(profiles["profiles"] as? [[String: Any]])
        let id = try XCTUnwrap(items.first { ($0["name"] as? String) == name }?["id"] as? String)
        let login = try await json("POST", "/v1/auth/login", body: ["profile_id": id, "pin": pin])
        return try XCTUnwrap(login["token"] as? String)
    }

    @MainActor private func json(_ method: String, _ path: String, token: String? = nil, body: [String: String]? = nil) async throws -> [String: Any] {
        var request = URLRequest(url: baseURL.appending(path: path))
        request.httpMethod = method
        request.timeoutInterval = 15
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        if let token { request.setValue("Bearer " + token, forHTTPHeaderField: "Authorization") }
        if let body { request.httpBody = try JSONSerialization.data(withJSONObject: body) }
        let (data, response) = try await URLSession.shared.data(for: request)
        let http = try XCTUnwrap(response as? HTTPURLResponse)
        XCTAssertTrue((200..<300).contains(http.statusCode), "API synthétique : \(method) \(path) → \(http.statusCode)")
        return try XCTUnwrap(JSONSerialization.jsonObject(with: data) as? [String: Any])
    }

    @MainActor private func registerCleanup(name: String, token: String) {
        var components = URLComponents(url: baseURL.appending(path: "/v1/me"), resolvingAgainstBaseURL: false)!
        components.queryItems = [URLQueryItem(name: "confirm", value: name)]
        let url = components.url!
        addTeardownBlock {
            var request = URLRequest(url: url)
            request.httpMethod = "DELETE"
            request.timeoutInterval = 15
            request.setValue("Bearer " + token, forHTTPHeaderField: "Authorization")
            let (_, response) = try await URLSession.shared.data(for: request)
            let http = try XCTUnwrap(response as? HTTPURLResponse)
            XCTAssertTrue((200..<300).contains(http.statusCode), "Suppression du profil synthétique créé par ce test.")
        }
    }
}
