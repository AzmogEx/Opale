import XCTest

/// Test de fumée P1 : parcours complet contre le backend local.
///
/// Prérequis : l'API Opale tourne sur http://localhost:8080 avec le profil
/// « Adam » (PIN 1234) et des valorisations (voir vérification P0).
final class SmokeTests: XCTestCase {

    @MainActor
    func testLoginPuisPatrimoineNetVisible() throws {
        let app = XCUIApplication()
        app.launchArguments = ["--reset-session"]
        app.launch()

        // 1. Porte de profils : « Adam » chargé depuis l'API.
        let adam = app.buttons["Adam"]
        XCTAssertTrue(adam.waitForExistence(timeout: 10), "Le profil Adam doit venir du backend")
        adam.tap()

        // 2. Connexion par PIN.
        let pinField = app.secureTextFields["Code PIN"]
        XCTAssertTrue(pinField.waitForExistence(timeout: 5))
        pinField.tap()
        pinField.typeText("1234")
        app.buttons["Déverrouiller"].tap()

        // 3. Accueil : le patrimoine net s'affiche (EF-010).
        // (.textCase(.uppercase) transforme le libellé → recherche insensible à la casse)
        let heroLabel = app.staticTexts
            .matching(NSPredicate(format: "label CONTAINS[c] 'patrimoine net'"))
            .firstMatch
        XCTAssertTrue(heroLabel.waitForExistence(timeout: 10), "Le héro Patrimoine net doit s'afficher")

        // Le montant vérifié en P0 (48 300 €) doit apparaître.
        let amount = app.staticTexts
            .matching(NSPredicate(format: "label CONTAINS '48' AND label CONTAINS '300'"))
            .firstMatch
        XCTAssertTrue(amount.waitForExistence(timeout: 5), "Le montant du patrimoine doit être visible")

        // 4. Les 5 onglets existent (EF-003) et Patrimoine liste les actifs.
        let tabBar = app.tabBars.firstMatch
        for tab in ["Accueil", "Flux", "Patrimoine", "Projection", "Assistant"] {
            XCTAssertTrue(tabBar.buttons[tab].exists, "Onglet manquant : \(tab)")
        }
        tabBar.buttons["Patrimoine"].tap()
        XCTAssertTrue(
            app.staticTexts["Compte courant BNP"].waitForExistence(timeout: 10),
            "L'actif créé en P0 doit être listé"
        )

        // 5. Projection (P2, EF-040/041) : la date d'indépendance et les
        //    hypothèses s'affichent, calculées par le moteur backend.
        tabBar.buttons["Projection"].tap()
        let independence = app.staticTexts
            .matching(NSPredicate(format: "label CONTAINS[c] 'indépendance'"))
            .firstMatch
        XCTAssertTrue(independence.waitForExistence(timeout: 10), "Le héro Indépendance doit s'afficher")
        let freedom = app.staticTexts
            .matching(NSPredicate(format: "label CONTAINS[c] 'libre en' OR label CONTAINS[c] 'déjà libre' OR label CONTAINS[c] 'hors d'"))
            .firstMatch
        XCTAssertTrue(freedom.waitForExistence(timeout: 10), "Le verdict de liberté doit s'afficher")
        XCTAssertTrue(app.sliders.firstMatch.exists, "Les curseurs d'hypothèses doivent exister")

        // 6. Flux (P3, EF-020→022) : les mouvements importés en CSV s'affichent,
        //    catégorisés, avec le résumé mensuel.
        tabBar.buttons["Flux"].tap()
        // Liste paresseuse : on cible une ligne du HAUT (les plus récentes),
        // sans présumer du type d'élément exposé par SwiftUI.
        let zzz = app.descendants(matching: .any)
            .matching(NSPredicate(format: "label CONTAINS 'Zzz Boutique Mystere'"))
            .firstMatch
        XCTAssertTrue(zzz.waitForExistence(timeout: 10), "Les transactions importées doivent s'afficher")
        let loisirs = app.descendants(matching: .any)
            .matching(NSPredicate(format: "label CONTAINS 'Loisirs'"))
            .firstMatch
        XCTAssertTrue(loisirs.exists, "La catégorie apprise (Loisirs) doit être visible")
        let revenus = app.staticTexts
            .matching(NSPredicate(format: "label CONTAINS[c] 'revenus'"))
            .firstMatch
        XCTAssertTrue(revenus.exists, "Le résumé mensuel doit s'afficher")

        // 7. Enveloppes (P4, EF-028) : la jauge Restaurants (dépassée) s'affiche.
        app.buttons["Enveloppes"].tap()
        let restaurants = app.descendants(matching: .any)
            .matching(NSPredicate(format: "label CONTAINS 'Restaurants'"))
            .firstMatch
        XCTAssertTrue(restaurants.waitForExistence(timeout: 10), "L'enveloppe Restaurants doit s'afficher")

        // 8. À venir (P4, EF-025/027) : le cash projeté s'affiche.
        app.buttons["À venir"].tap()
        let cashPrevu = app.descendants(matching: .any)
            .matching(NSPredicate(format: "label CONTAINS[c] 'cash prévu'"))
            .firstMatch
        XCTAssertTrue(cashPrevu.waitForExistence(timeout: 10), "Le cashflow futur doit s'afficher")

        // 9. Accueil (P4, EF-015/053) : le score de santé et l'alerte d'enveloppe.
        tabBar.buttons["Accueil"].tap()
        let sante = app.descendants(matching: .any)
            .matching(NSPredicate(format: "label CONTAINS[c] 'santé financière'"))
            .firstMatch
        XCTAssertTrue(sante.waitForExistence(timeout: 10), "La carte santé financière doit s'afficher")
        let alerte = app.descendants(matching: .any)
            .matching(NSPredicate(format: "label CONTAINS[c] 'dépassée'"))
            .firstMatch
        XCTAssertTrue(alerte.exists, "L'alerte d'enveloppe dépassée doit s'afficher")
    }
}
