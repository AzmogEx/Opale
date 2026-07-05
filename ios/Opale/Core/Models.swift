import Foundation

// Modèles miroirs de l'API Opale (backend Go). Les clés JSON sont en
// snake_case — chaque type déclare ses CodingKeys explicitement.

// MARK: - Profil & session

struct Profile: Identifiable, Codable, Hashable, Sendable {
    let id: String
    var name: String
    var privacyDefault: String

    enum CodingKeys: String, CodingKey {
        case id, name
        case privacyDefault = "privacy_default"
    }
}

struct LoginResponse: Codable, Sendable {
    let token: String
    let profile: Profile

    enum CodingKeys: String, CodingKey { case token, profile }
}

// MARK: - Patrimoine

enum AssetKind: String, Codable, CaseIterable, Identifiable, Sendable {
    case checking, savings
    case lifeInsurance = "life_insurance"
    case pea, cto, crypto
    case realEstate = "real_estate"
    case preciousMetal = "precious_metal"
    case vehicle, valuable
    case companyShare = "company_share"
    case other

    var id: String { rawValue }

    /// Libellé français.
    var label: String {
        switch self {
        case .checking: "Compte courant"
        case .savings: "Livret / épargne"
        case .lifeInsurance: "Assurance-vie"
        case .pea: "PEA"
        case .cto: "Compte-titres"
        case .crypto: "Crypto"
        case .realEstate: "Immobilier"
        case .preciousMetal: "Or / métaux"
        case .vehicle: "Véhicule"
        case .valuable: "Objet de valeur"
        case .companyShare: "Parts de société"
        case .other: "Autre"
        }
    }

    var systemImage: String {
        switch self {
        case .checking: "creditcard"
        case .savings: "banknote"
        case .lifeInsurance: "shield.lefthalf.filled"
        case .pea, .cto: "chart.pie"
        case .crypto: "bitcoinsign.circle"
        case .realEstate: "house"
        case .preciousMetal: "sparkle"
        case .vehicle: "car"
        case .valuable: "crown"
        case .companyShare: "briefcase"
        case .other: "square.grid.2x2"
        }
    }
}

enum LiabilityKind: String, Codable, CaseIterable, Identifiable, Sendable {
    case mortgage
    case autoLoan = "auto_loan"
    case consumerLoan = "consumer_loan"
    case other

    var id: String { rawValue }

    var label: String {
        switch self {
        case .mortgage: "Crédit immobilier"
        case .autoLoan: "Crédit auto"
        case .consumerLoan: "Crédit conso"
        case .other: "Autre dette"
        }
    }

    var systemImage: String {
        switch self {
        case .mortgage: "house.lodge"
        case .autoLoan: "car.rear"
        case .consumerLoan: "cart"
        case .other: "minus.circle"
        }
    }
}

struct Asset: Identifiable, Codable, Hashable, Sendable {
    let id: String
    var name: String
    var kind: AssetKind
    var currency: String
    var note: String
    var archived: Bool
    var latestValue: Cents?

    enum CodingKeys: String, CodingKey {
        case id, name, kind, currency, note, archived
        case latestValue = "latest_value_cents"
    }
}

struct Liability: Identifiable, Codable, Hashable, Sendable {
    let id: String
    var name: String
    var kind: LiabilityKind
    var currency: String
    var note: String
    var archived: Bool
    var latestValue: Cents?

    enum CodingKeys: String, CodingKey {
        case id, name, kind, currency, note, archived
        case latestValue = "latest_value_cents"
    }
}

struct Valuation: Identifiable, Codable, Hashable, Sendable {
    let id: String
    var value: Cents
    var asOf: Date
    var note: String

    enum CodingKeys: String, CodingKey {
        case id, note
        case value = "value_cents"
        case asOf = "as_of"
    }
}

// MARK: - Patrimoine net (CA-1 : calcul côté backend, jamais côté client)

struct NetWorth: Codable, Hashable, Sendable {
    var assetsTotal: Cents
    var liabilitiesTotal: Cents
    var net: Cents
    var currency: String

    enum CodingKeys: String, CodingKey {
        case assetsTotal = "assets_total_cents"
        case liabilitiesTotal = "liabilities_total_cents"
        case net = "net_cents"
        case currency
    }
}

struct NetWorthPoint: Codable, Hashable, Identifiable, Sendable {
    var asOf: Date
    var assetsTotal: Cents
    var liabilitiesTotal: Cents
    var net: Cents

    var id: Date { asOf }

    enum CodingKeys: String, CodingKey {
        case asOf = "as_of"
        case assetsTotal = "assets_total_cents"
        case liabilitiesTotal = "liabilities_total_cents"
        case net = "net_cents"
    }
}

struct NetWorthHistory: Codable, Hashable, Sendable {
    var points: [NetWorthPoint]
    var currency: String
}

// MARK: - Enveloppes de listes renvoyées par l'API

struct AssetsEnvelope: Codable, Sendable { let assets: [Asset] }
struct LiabilitiesEnvelope: Codable, Sendable { let liabilities: [Liability] }
struct ValuationsEnvelope: Codable, Sendable { let valuations: [Valuation] }
struct ProfilesEnvelope: Codable, Sendable { let profiles: [Profile] }
