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

        // Le montant vérifié en E2E (109 800 € depuis P6) doit apparaître.
        let amount = app.staticTexts
            .matching(NSPredicate(format: "label CONTAINS '109' AND label CONTAINS '800'"))
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

        // 10. Assistant (P5, EF-050/061) : le radar de risques s'affiche et
        //     une question reçoit une réponse (repli moteur, IA hors ligne).
        tabBar.buttons["Assistant"].tap()
        let radar = app.descendants(matching: .any)
            .matching(NSPredicate(format: "label CONTAINS[c] 'radar de risques'"))
            .firstMatch
        XCTAssertTrue(radar.waitForExistence(timeout: 10), "Le radar de risques doit s'afficher")
        let revenuUnique = app.descendants(matching: .any)
            .matching(NSPredicate(format: "label CONTAINS[c] 'revenu unique'"))
            .firstMatch
        XCTAssertTrue(revenuUnique.exists, "Le risque « revenu unique » doit être détecté")

        let suggestion = app.buttons["Comment va mon épargne ?"]
        XCTAssertTrue(suggestion.waitForExistence(timeout: 5), "Les suggestions doivent s'afficher")
        suggestion.tap()
        // Sans IA configurée, la réponse est le repli déterministe du moteur.
        let reponse = app.descendants(matching: .any)
            .matching(NSPredicate(format: "label CONTAINS[c] 'calculé par le moteur'"))
            .firstMatch
        XCTAssertTrue(reponse.waitForExistence(timeout: 15), "La réponse (repli moteur) doit s'afficher")

        // 11. La profondeur (P6, EF-033/034) : les centres du Patrimoine.
        //     (label EXACT : la ligne d'actif « Studio Lyon 3e » contient
        //     aussi « Immobilier » comme sous-titre de type.)
        tabBar.buttons["Patrimoine"].tap()
        let immobilier = app.buttons["Immobilier"]
        XCTAssertTrue(immobilier.waitForExistence(timeout: 10), "Le centre Immobilier doit s'afficher")
        immobilier.tap()
        let rendement = app.descendants(matching: .any)
            .matching(NSPredicate(format: "label CONTAINS[c] 'rendement'"))
            .firstMatch
        XCTAssertTrue(rendement.waitForExistence(timeout: 10), "Les indicateurs immobiliers doivent s'afficher")
        // Retour à la racine : re-taper l'onglet courant dépile la navigation.
        tabBar.buttons["Patrimoine"].tap()

        let placements = app.buttons["Placements"]
        XCTAssertTrue(placements.waitForExistence(timeout: 10), "Le centre Placements doit s'afficher")
        placements.tap()
        // Contenu propre à l'écran Placements (insensible casse + accents).
        let repartition = app.descendants(matching: .any)
            .matching(NSPredicate(format: "label CONTAINS[cd] 'repartition' OR label CONTAINS[cd] 'performance'"))
            .firstMatch
        // Le tap peut se perdre dans l'animation de dépilement : un re-essai
        // SEULEMENT si on est toujours sur la grille des centres.
        if !repartition.waitForExistence(timeout: 4), placements.exists {
            placements.tap()
        }
        XCTAssertTrue(repartition.waitForExistence(timeout: 10), "La répartition des placements doit s'afficher")
    }
}
