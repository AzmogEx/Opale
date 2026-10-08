import XCTest

/// Against the disposable delivery API only; no personal profiles are used.
final class DeliveryTests: XCTestCase {
    @MainActor func testDemoAndAllPilotToolsAreReachable() throws {
        let app = XCUIApplication()
        app.launchArguments = ["--reset-session", "--base-url", "http://localhost:58088"]
        app.launch()
        if app.buttons["Passer"].waitForExistence(timeout: 3) { app.buttons["Passer"].tap() }
        let demo = app.buttons["Essayer la démonstration"]
        XCTAssertTrue(demo.waitForExistence(timeout: 15))
        demo.tap()
        let settings = app.buttons["Réglages"]
        XCTAssertTrue(settings.waitForExistence(timeout: 20))
        settings.tap()
        let pilot = app.buttons["Pilote automatique"]
        for _ in 0..<4 where !pilot.isHittable { app.swipeUp() }
        XCTAssertTrue(pilot.waitForExistence(timeout: 5))
        pilot.tap()
        for title in ["Fiscalité & PER", "Crédit & capacité", "Allocation cible", "Alertes personnalisées", "Bilan annuel", "Cours automatiques", "Catégories & règles", "Historique mensuel"] {
            let item = app.buttons[title]
            XCTAssertTrue(item.waitForExistence(timeout: 5), title)
        }
        app.buttons["Bilan annuel"].tap()
        XCTAssertTrue(app.staticTexts["Revenus"].waitForExistence(timeout: 10))
        let screenshot = XCTAttachment(screenshot: app.screenshot())
        screenshot.name = "Wrapped-demo"
        screenshot.lifetime = .keepAlways
        add(screenshot)
    }
    @MainActor func testColdStartHidesContentUntilUnlock() throws {
        let app = XCUIApplication()
        app.launchArguments = ["--reset-session", "--base-url", "http://localhost:58088"]
        app.launch()
        if app.buttons["Passer"].waitForExistence(timeout: 3) { app.buttons["Passer"].tap() }
        let demo = app.buttons["Essayer la démonstration"]
        XCTAssertTrue(demo.waitForExistence(timeout: 15))
        demo.tap()
        XCTAssertTrue(app.buttons["Réglages"].waitForExistence(timeout: 15))
        app.terminate()
        app.launchArguments = ["--base-url", "http://localhost:58088"]
        app.launch()
        XCTAssertTrue(app.staticTexts["Opale est verrouillée"].waitForExistence(timeout: 15))
        XCTAssertFalse(app.buttons["Réglages"].exists)
        XCTAssertFalse(app.tabBars.buttons["Patrimoine"].exists)
        let screenshot = XCTAttachment(screenshot: app.screenshot())
        screenshot.name = "Cold-start-locked"
        screenshot.lifetime = .keepAlways
        add(screenshot)
        app.buttons["Changer de profil"].tap()
        XCTAssertTrue(app.buttons["Essayer la démonstration"].waitForExistence(timeout: 10))
    }

    @MainActor func testManualAssetAndDiscreetMode() throws {
        let app = XCUIApplication()
        app.launchArguments = ["--reset-session", "--base-url", "http://localhost:58088"]
        app.launch()
        if app.buttons["Passer"].waitForExistence(timeout: 3) { app.buttons["Passer"].tap() }
        XCTAssertTrue(app.buttons["Essayer la démonstration"].waitForExistence(timeout: 15))
        app.buttons["Essayer la démonstration"].tap()
        XCTAssertTrue(app.buttons["Masquer les montants"].waitForExistence(timeout: 15))
        app.buttons["Masquer les montants"].tap()
        XCTAssertTrue(app.staticTexts["Montant masqué"].firstMatch.waitForExistence(timeout: 5))
        app.buttons["Afficher les montants"].tap()
        app.tabBars.buttons["Patrimoine"].tap()
        XCTAssertTrue(app.buttons["Ajouter au patrimoine"].waitForExistence(timeout: 10))
        app.buttons["Ajouter au patrimoine"].tap()
        app.buttons["Nouvel actif"].tap()
        let name = "Recette UI " + UUID().uuidString.prefix(5)
        let field = app.textFields["Nom (ex. Compte courant BNP)"]
        XCTAssertTrue(field.waitForExistence(timeout: 5))
        field.tap(); field.typeText(String(name))
        let value = app.textFields["Ex. 12 500,00"]
        value.tap(); value.typeText("1234,56")
        app.buttons["Ajouter"].tap()
        let holding = app.buttons.containing(.staticText, identifier: String(name)).firstMatch
        for _ in 0..<8 where !holding.isHittable { app.swipeUp() }
        XCTAssertTrue(holding.waitForExistence(timeout: 10))
        holding.tap()
        XCTAssertTrue(app.buttons["Ajouter une valorisation"].waitForExistence(timeout: 10))
        XCTAssertTrue(app.staticTexts["1\u{202F}234,56 €"].exists || app.staticTexts.matching(NSPredicate(format: "label CONTAINS %@", "234,56")).firstMatch.exists)
        let screenshot = XCTAttachment(screenshot: app.screenshot())
        screenshot.name = "Created-asset-history"
        screenshot.lifetime = .keepAlways
        add(screenshot)
    }

    @MainActor func testLargeTextAndDiscreetPilot() throws {
        let app = XCUIApplication()
        app.launchArguments = ["--reset-session", "--base-url", "http://localhost:58088", "-UIPreferredContentSizeCategoryName", "UICTContentSizeCategoryAccessibilityXXXL"]
        app.launch()
        if app.buttons["Passer"].waitForExistence(timeout: 3) { app.buttons["Passer"].tap() }
        for _ in 0..<8 where !app.buttons["Essayer la démonstration"].isHittable { app.swipeUp() }
        XCTAssertTrue(app.buttons["Essayer la démonstration"].waitForExistence(timeout: 15))
        app.buttons["Essayer la démonstration"].tap()
        XCTAssertTrue(app.buttons["Masquer les montants"].waitForExistence(timeout: 15))
        let home = XCTAttachment(screenshot: app.screenshot())
        home.name = "Home-accessibility-XXXL"; home.lifetime = .keepAlways; add(home)
        app.buttons["Masquer les montants"].tap()
        XCTAssertTrue(app.staticTexts["Montant masqué"].firstMatch.waitForExistence(timeout: 5))
        XCTAssertFalse(app.otherElements["net-worth-chart"].exists)
        app.buttons["Réglages"].tap()
        let pilot = app.buttons["Pilote automatique"]
        for _ in 0..<6 where !pilot.isHittable { app.swipeUp() }
        pilot.tap()
        let wrapped = app.buttons["Bilan annuel"]
        for _ in 0..<6 where !wrapped.isHittable { app.swipeUp() }
        XCTAssertTrue(wrapped.exists)
        wrapped.tap()
        XCTAssertTrue(app.staticTexts["Revenus"].waitForExistence(timeout: 10))
        XCTAssertTrue(app.staticTexts["Montant masqué"].firstMatch.exists)
        XCTAssertFalse(app.staticTexts.matching(NSPredicate(format: "label CONTAINS %@", "€")).firstMatch.exists)
        let screenshot = XCTAttachment(screenshot: app.screenshot())
        screenshot.name = "Wrapped-discreet-accessibility-XXXL"
        screenshot.lifetime = .keepAlways
        add(screenshot)
    }

}
