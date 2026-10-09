import Foundation

enum InvestmentRisk: String, Codable, CaseIterable, Identifiable, Sendable {
    case low, medium, high
    var id: String { rawValue }
    var label: String { switch self { case .low: "Faible"; case .medium: "Moyen"; case .high: "Élevé" } }
    var guidance: String {
        switch self {
        case .low: "Pour une réserve ou un projet proche, privilégie la disponibilité et les garanties applicables. L’inflation peut réduire le pouvoir d’achat."
        case .medium: "Pour un horizon de plusieurs années, compare des obligations et des supports mixtes diversifiés. Leur valeur peut baisser et leur capital n’est pas garanti."
        case .high: "Pour un horizon long et une capacité à supporter des pertes, étudie les actions diversifiées. Actions individuelles, immobilier et crypto ajoutent des risques de concentration ou de liquidité."
        }
    }
}
enum InvestmentFamily: String, Codable, CaseIterable, Identifiable, Sendable {
    case savings, bonds, etf, stock, crypto, property
    var id: String { rawValue }
    var label: String { switch self { case .savings: "Épargne"; case .bonds: "Obligations"; case .etf: "ETF"; case .stock: "Actions"; case .crypto: "Crypto"; case .property: "Immobilier" } }
    var icon: String { switch self { case .savings: "banknote"; case .bonds: "building.columns"; case .etf: "chart.pie"; case .stock: "chart.line.uptrend.xyaxis"; case .crypto: "bitcoinsign.circle"; case .property: "house" } }
}
struct InvestmentSource: Codable, Hashable, Sendable { var title: String; var url: String }
struct InvestmentIdea: Codable, Identifiable, Sendable {
    var id: String
    var name: String
    var identifier: String
    var risk: InvestmentRisk
    var families: [InvestmentFamily]
    var geographies: [String]
    var envelopes: [String]
    var franceOnly: Bool = false
    var horizonYears: Int
    var rationale: String
    var risks: String
    var costs: String
    var liquidity: String
    var check: String
    var sources: [InvestmentSource]
    func matches(risk: InvestmentRisk, country: String, family: String, envelope: String, residence: String, horizon: Int?, query: String) -> Bool {
        guard self.risk == risk, country.isEmpty || geographies.contains(country), family.isEmpty || families.contains(where: { $0.rawValue == family }), envelope.isEmpty || envelopes.contains(envelope), !franceOnly || residence == "France" else { return false }
        if ["PEA", "Assurance-vie", "Livret"].contains(envelope), residence != "France" { return false }
        if let horizon, horizon < horizonYears { return false }
        return query.isEmpty || (name + " " + identifier + " " + geographies.joined(separator: " ")).localizedStandardContains(query)
    }
}

/// Editorial examples, reviewed on this date. No live price, ranking or expected return.
enum InvestmentCatalog {
    static let checkedOn = "2026-10-09"
    static let geographies = ["Monde", "France", "Europe", "Pays-Bas", "États-Unis", "Japon", "Émergents"]
    static let residences = ["France", "Belgique", "Suisse", "Luxembourg", "Allemagne", "Autre"]
    static let peaSource = InvestmentSource(title: "Service Public · PEA", url: "https://www.service-public.gouv.fr/particuliers/vosdroits/F2385")
    static let etfSource = InvestmentSource(title: "AMF · Comprendre les ETF", url: "https://www.amf-france.org/fr/espace-epargnants/comprendre-les-produits-financiers/placements-collectifs/trackers-etf")
    static let cryptoSource = InvestmentSource(title: "AMF · Précautions sur les crypto-actifs", url: "https://www.amf-france.org/fr/espace-epargnants/proteger-son-epargne/crypto-actifs-bitcoin-etc/investir-en-crypto-actifs-les-precautions-pratiques")
    static let ideas: [InvestmentIdea] = [
        InvestmentIdea(id: "regulated-savings", name: "Livrets réglementés", identifier: "Livret A · LDDS · LEP selon éligibilité", risk: .low, families: [.savings], geographies: ["France"], envelopes: ["Livret"], franceOnly: true, horizonYears: 0,
            rationale: "Une piste pour la réserve de sécurité et les dépenses proches. Compare les livrets auxquels tu as droit avant de bloquer ton argent ailleurs.",
            risks: "L’inflation peut dépasser la rémunération. Les plafonds, conditions de résidence et de revenus diffèrent selon le livret.",
            costs: "Compare les conditions de la banque. Vérifie les taux et plafonds en vigueur ; aucun taux n’est figé ici.", liquidity: "Épargne disponible, sous réserve des modalités bancaires.", check: "Vérifier l’éligibilité, le plafond restant et la garantie de l’État applicable.",
            sources: [.init(title: "Ministère de l’Économie · Garanties des dépôts", url: "https://www.economie.gouv.fr/facileco/garantie-des-depots-et-des-titres"), .init(title: "Service Public · Livret A", url: "https://www.service-public.gouv.fr/particuliers/vosdroits/F2365")]),
        InvestmentIdea(id: "term-deposit", name: "Compte à terme en euros", identifier: "Offres bancaires à comparer", risk: .low, families: [.savings], geographies: ["France", "Europe"], envelopes: ["Banque"], horizonYears: 1,
            rationale: "Pour une somme dont la date d’utilisation est connue, comparer une rémunération contractuelle à celle d’une épargne disponible.",
            risks: "Argent immobilisé, pénalités possibles en cas de retrait anticipé. Risque bancaire au-delà de la garantie applicable, inflation et change si ton objectif est dans une autre devise.",
            costs: "Comparer le taux net de fiscalité et les pénalités de sortie.", liquidity: "Selon la durée et les clauses du contrat.", check: "En France, vérifier le plafond de garantie de 100 000 € par déposant et établissement, en cumulant les dépôts concernés. Vérifier le régime local hors de France.",
            sources: [.init(title: "Service Public · Compte à terme", url: "https://www.service-public.gouv.fr/particuliers/vosdroits/F2372"), .init(title: "Ministère de l’Économie · Garantie des dépôts", url: "https://www.economie.gouv.fr/facileco/garantie-des-depots-et-des-titres")]),
        InvestmentIdea(id: "euro-fund", name: "Fonds en euros", identifier: "Dans une assurance-vie française", risk: .low, families: [.savings], geographies: ["France"], envelopes: ["Assurance-vie"], franceOnly: true, horizonYears: 3,
            rationale: "Pour comparer une épargne dont la valeur est garantie par l’assureur, en tenant compte de l’ancienneté et des frais du contrat.",
            risks: "Garantie portée par l’assureur, selon le contrat. Ne pas confondre avec les unités de compte ou l’euro-croissance. Les bonus peuvent imposer une part risquée.",
            costs: "Frais de versement, de gestion et conditions des bonus : comparer le rendement net de ces frais et de la fiscalité.", liquidity: "Rachat possible selon le contrat ; le versement n’est pas instantané.", check: "Lire les conditions de garantie, notamment le traitement des frais, et les contraintes de versement.",
            sources: [.init(title: "Service Public · Fonctionnement de l’assurance-vie", url: "https://www.service-public.gouv.fr/particuliers/vosdroits/F15274")]),
        InvestmentIdea(id: "short-euro-bonds", name: "iShares € Govt Bond 1–3yr UCITS ETF", identifier: "IE00B14X4Q57 · part distribuante", risk: .medium, families: [.etf, .bonds], geographies: ["Europe"], envelopes: ["CTO"], horizonYears: 3,
            rationale: "Un exemple d’exposition à des obligations d’États de la zone euro de durée courte, à comparer à un dépôt garanti selon le besoin de liquidité.",
            risks: "Capital non garanti : risques de taux, de crédit souverain et de marché. La durée courte réduit la sensibilité aux taux sans supprimer les pertes.",
            costs: "Frais du fonds, courtage, écart achat/vente et fiscalité des distributions. Lire le DIC actuel.", liquidity: "Négocié en Bourse ; prix variable et liquidité de marché.", check: "Vérifier le DIC, la duration, la qualité des émetteurs et la part exacte. Un rendement obligataire affiché n’est pas une promesse.",
            sources: [.init(title: "iShares · Fiche officielle", url: "https://www.ishares.com/uk/professionals/en/products/251733/ishares-euro-government-bond-13yr-ucits-etf"), etfSource]),
        InvestmentIdea(id: "lifestrategy40", name: "Vanguard LifeStrategy 40% Equity UCITS ETF", identifier: "IE00BMVB5M21 · EUR capitalisant", risk: .medium, families: [.etf, .bonds], geographies: ["Monde"], envelopes: ["CTO"], horizonYears: 5,
            rationale: "Un exemple de support mixte avec environ 40 % d’actions et 60 % d’obligations, pour étudier une diversification et un rééquilibrage dans un même fonds.",
            risks: "Actions et obligations peuvent baisser ensemble. Pas de garantie en capital. La poche obligataire vise une couverture en euros, sans supprimer tous les risques.",
            costs: "Frais courants du fonds et du courtier, spread et fiscalité. Vérifier le DIC et la disponibilité chez ton courtier.", liquidity: "ETF coté : vente au prix du marché.", check: "Comparer l’allocation à ton horizon et à ta capacité de perte. La catégorie Moyen est un repère Opale, pas une garantie ni le score du DIC.",
            sources: [.init(title: "Vanguard · Fiche officielle", url: "https://www.fr.vanguard/professionnel/produits/etf/lifestrategy/9492/lifestrategy-40-equity-ucits-etf-eur-accumulating"), etfSource]),
        equityETF("all-world", "Vanguard FTSE All-World UCITS ETF", "IE00BK5BQT80 · capitalisant", ["Monde"], ["CTO"], "Actions de marchés développés et émergents dans un même support : une piste pour une exposition mondiale diversifiée.", "Poids important des grands marchés, risque actions et change. Le pays de domiciliation du fonds n’est pas la zone d’investissement.", "https://www.vanguard.co.uk/professional/product/etf/equity/9679/ftse-all-world-ucits-etf-usd-accumulating"),
        equityETF("world-pea", "iShares MSCI World Swap PEA UCITS ETF", "IE0002XZSHO1 · WPEA · capitalisant", ["Monde"], ["PEA", "CTO"], "Un exemple d’ETF éligible au PEA pour les marchés développés. Le MSCI World n’inclut pas les marchés émergents.", "Risque actions, change et contrepartie lié à la réplication synthétique. L’éligibilité PEA ne garantit pas le capital.", "https://www.blackrock.com/fr/intermediaries/products/335178/"),
        equityETF("sp500", "iShares Core S&P 500 UCITS ETF", "IE00B5BMR087 · capitalisant", ["États-Unis"], ["CTO"], "Exposition aux grandes entreprises américaines. À comparer à un ETF mondial si tu veux assumer une concentration sur les États-Unis.", "Concentration sur un pays et ses grandes valeurs, change dollar/euro. Ajouter ce fonds à un World peut renforcer des positions déjà présentes.", "https://www.ishares.com/nl/professionele-belegger/nl/producten/253743/CSPX"),
        equityETF("europe", "iShares Core MSCI Europe UCITS ETF", "IE00B4K48X80 · capitalisant", ["Europe"], ["CTO"], "Une piste pour une exposition aux actions européennes. La zone Europe comprend aussi des marchés hors zone euro.", "Risque actions, secteurs dominants et devises non euro. L’éligibilité au PEA de cette part n’est pas supposée.", "https://www.ishares.com/de/privatanleger/de/produkte/251861/SMEA"),
        equityETF("japan", "iShares Core MSCI Japan IMI UCITS ETF", "IE00B4L5YX21 · capitalisant", ["Japon"], ["CTO"], "Exposition aux entreprises japonaises de différentes tailles, pour étudier une diversification géographique ciblée.", "Concentration sur le Japon, risque actions et yen. La cotation en euros ne supprime pas automatiquement le risque de change.", "https://www.ishares.com/uk/professionals/en/products/251867/sjpa"),
        equityETF("emerging", "iShares Core MSCI EM IMI UCITS ETF", "IE00BKM4GZ66 · capitalisant", ["Émergents"], ["CTO"], "Une exposition aux actions des marchés émergents, grandes, moyennes et petites capitalisations, à comparer au contenu de ton ETF mondial.", "Risques politiques, de change, de liquidité et de gouvernance. La composition par pays peut évoluer.", "https://www.ishares.com/uk/professionals/en/products/264659/ishares-core-msci-em-imi-ucits-etf"),
        stock("schneider", "Schneider Electric", "FR0000121972 · SU · Paris", "France", "Équipements électriques et gestion de l’énergie : une entreprise à analyser pour une exposition industrielle.", "Cycle industriel, concurrence et valorisation. Comparer croissance, marges, dette et flux de trésorerie avant toute décision.", "https://www.se.com/ww/fr/about-us/investor-relations/share-information/share-price/", pea: true),
        stock("asml", "ASML", "NL0010273215 · Amsterdam", "Pays-Bas", "Équipements de fabrication des semi-conducteurs : une piste d’analyse sur la chaîne technologique.", "Cycle des semi-conducteurs, restrictions d’exportation, clients concentrés et prix payé pour la croissance.", "https://www.asml.com/en/investors/shares", pea: true),
        stock("microsoft", "Microsoft", "MSFT · Nasdaq", "États-Unis", "Logiciels et services cloud : une entreprise à comparer à son poids déjà présent dans les ETF mondiaux.", "Valorisation, dépenses d’investissement, concurrence et réglementation. Risque de change en dollars.", "https://www.microsoft.com/en-us/investor/sec-filings"),
        stock("toyota", "Toyota Motor", "7203 · Tokyo", "Japon", "Un exemple d’entreprise automobile japonaise à étudier pour une exposition industrielle différente.", "Cycle automobile, transition technologique, tarifs douaniers et change yen. Les modalités d’achat varient selon la place boursière.", "https://global.toyota/en/ir/stock/outline/"),
        crypto("bitcoin", "Bitcoin", "BTC", "Actif numérique à offre encadrée par son protocole. À étudier uniquement pour une part dont la perte totale serait supportable."),
        crypto("ethereum", "Ethereum", "ETH", "Réseau d’applications décentralisées : analyser séparément le jeton, les usages, les coûts et les risques de conservation. Le staking ajoute d’autres risques."),
        InvestmentIdea(id: "rental", name: "Immobilier locatif direct", identifier: "Projet à chiffrer bien par bien", risk: .high, families: [.property], geographies: ["France"], envelopes: ["Direct"], franceOnly: true, horizonYears: 10,
            rationale: "Un projet tangible à étudier à partir du rendement net et du cash-flow, après toutes les dépenses et le crédit.",
            risks: "Vacance, impayés, travaux, baisse de valeur, concentration sur un bien et effet de levier. La fiscalité et les règles de location dépendent du lieu et du régime.",
            costs: "Acquisition, financement, copropriété, entretien, gestion, assurance, taxe foncière et fiscalité des loyers.", liquidity: "Faible : vente longue et frais de transaction élevés.", check: "Vérifier le DPE, la demande locative, les règles locales et un scénario avec plusieurs mois de vacance et des travaux.",
            sources: [.init(title: "ANIL · Projet d’investissement locatif", url: "https://www.anil.org/votre-besoin/gerer-un-bien/bailleur/investissement-locatif/")]),
        InvestmentIdea(id: "scpi", name: "SCPI immobilières", identifier: "Choisir selon les actifs et le DIC", risk: .high, families: [.property], geographies: ["France", "Europe"], envelopes: ["Direct", "Assurance-vie"], franceOnly: true, horizonYears: 10,
            rationale: "Pour étudier une exposition immobilière gérée et répartie entre plusieurs biens, sans gérer soi-même chaque locataire.",
            risks: "Capital et revenus non garantis. Baisse de prix des parts, vacance et revente potentiellement bloquée. Les zones d’investissement dépendent de chaque SCPI.",
            costs: "Frais de souscription, de gestion et de cession, plus les frais du contrat si détention en assurance-vie.", liquidity: "Faible : le délai de revente dépend des acheteurs et du marché, sans garantie de sortie.", check: "Lire le DIC, les rapports, les retraits en attente, l’endettement et la répartition réelle des immeubles. Vérifier la fiscalité de chaque pays.",
            sources: [.init(title: "AMF · Comprendre les SCPI", url: "https://www.amf-france.org/fr/espace-epargnants/comprendre-les-produits-financiers/placements-collectifs/scpi-un-autre-moyen-dinvestir-dans-limmobilier")])
    ]
    private static func equityETF(_ id: String, _ name: String, _ identifier: String, _ countries: [String], _ envelopes: [String], _ rationale: String, _ risks: String, _ url: String) -> InvestmentIdea {
        InvestmentIdea(id: id, name: name, identifier: identifier, risk: .high, families: [.etf], geographies: countries, envelopes: envelopes, horizonYears: 8, rationale: rationale, risks: risks + " Capital non garanti.", costs: "Comparer frais courants, courtage, garde, spread, suivi de l’indice et fiscalité. Consulter le DIC de la part exacte.", liquidity: "ETF coté, au prix du marché. Une forte baisse reste possible même sur longue durée.", check: "Comparer l’indice, les recouvrements avec tes placements, la réplication et la disponibilité chez ton courtier. Aucune estimation de rendement n’est fournie.", sources: [.init(title: "Émetteur · Fiche officielle", url: url), etfSource] + (envelopes.contains("PEA") ? [peaSource] : []))
    }
    private static func stock(_ id: String, _ name: String, _ identifier: String, _ country: String, _ rationale: String, _ risks: String, _ url: String, pea: Bool = false) -> InvestmentIdea {
        InvestmentIdea(id: id, name: name, identifier: identifier, risk: .high, families: [.stock], geographies: [country], envelopes: pea ? ["CTO", "PEA"] : ["CTO"], horizonYears: 8, rationale: rationale, risks: risks + " Une action individuelle peut perdre l’essentiel de sa valeur.", costs: "Courtage, garde, change, taxes de transaction et fiscalité des dividendes / plus-values selon ta résidence.", liquidity: "Cotée en Bourse ; volatilité et horaires propres au marché.", check: "C’est une piste d’analyse, sans jugement sur le prix actuel. Lire les résultats récents et comparer à un ETF diversifié. Confirmer l’éligibilité de la ligne et l’accès au marché auprès du courtier.", sources: [.init(title: "Entreprise · Relations investisseurs", url: url)] + (pea ? [peaSource] : []))
    }
    private static func crypto(_ id: String, _ name: String, _ identifier: String, _ rationale: String) -> InvestmentIdea {
        InvestmentIdea(id: id, name: name, identifier: identifier, risk: .high, families: [.crypto], geographies: ["Monde"], envelopes: ["Direct"], horizonYears: 8, rationale: rationale, risks: "Très forte volatilité et perte totale possibles. Risques de plateforme, de piratage, de clés perdues et de réglementation. Une durée longue ne garantit aucun résultat.", costs: "Frais de plateforme, spread, réseau, conservation et fiscalité. Ne pas supposer un rendement fixe.", liquidity: "Échanges souvent continus, mais retraits ou plateformes peuvent être suspendus.", check: "Vérifier le statut réglementaire du prestataire et maîtriser la conservation. Éviter le levier et distinguer achat, prêt et staking.", sources: [cryptoSource])
    }
}
