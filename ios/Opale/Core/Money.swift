import Foundation

/// Montant en centimes (entier signé) — miroir du package Go `money`.
///
/// RÈGLE D'OR (ENF-007) : l'argent n'est JAMAIS manipulé en Double/Float.
/// Les conversions d'affichage passent par `Decimal` (base 10, exacte).
struct Cents: Hashable, Codable, Sendable, Comparable, AdditiveArithmetic {
    var raw: Int64

    init(_ raw: Int64) { self.raw = raw }

    init(from decoder: Decoder) throws {
        raw = try decoder.singleValueContainer().decode(Int64.self)
    }

    func encode(to encoder: Encoder) throws {
        var c = encoder.singleValueContainer()
        try c.encode(raw)
    }

    static let zero = Cents(0)
    static func < (lhs: Cents, rhs: Cents) -> Bool { lhs.raw < rhs.raw }
    static func + (lhs: Cents, rhs: Cents) -> Cents { Cents(lhs.raw + rhs.raw) }
    static func - (lhs: Cents, rhs: Cents) -> Cents { Cents(lhs.raw - rhs.raw) }

    /// Valeur décimale exacte en euros (pour l'affichage uniquement).
    var decimalEuros: Decimal { Decimal(raw) / 100 }

    /// Valeur en euros pour le POSITIONNEMENT GRAPHIQUE uniquement (Swift
    /// Charts exige un Double). Jamais utilisé pour un calcul financier.
    var chartValue: Double { Double(raw) / 100 }

    /// Convertit une saisie décimale ("123,45", "1 000.5", "42300") en centimes,
    /// sans passer par un Double — miroir du `money.Parse` du backend Go.
    static func parse(_ input: String) -> Cents? {
        parse(input.replacingOccurrences(of: "€", with: ""), currency: "EUR")
    }
}

/// Formatage monétaire d'Opale (EUR, locale française).
enum MoneyFormat {
    private static let full: NumberFormatter = {
        let f = NumberFormatter()
        f.numberStyle = .currency
        f.currencyCode = "EUR"
        f.locale = Locale(identifier: "fr_FR")
        f.maximumFractionDigits = 2
        f.minimumFractionDigits = 2
        return f
    }()

    private static let whole: NumberFormatter = {
        let f = NumberFormatter()
        f.numberStyle = .currency
        f.currencyCode = "EUR"
        f.locale = Locale(identifier: "fr_FR")
        f.maximumFractionDigits = 0
        return f
    }()

    /// "48 300,00 €"
    static func euros(_ cents: Cents) -> String {
        full.string(from: cents.decimalEuros as NSDecimalNumber) ?? "\(cents.decimalEuros) €"
    }

    /// "48 300 €" — pour les grands chiffres (patrimoine net).
    static func eurosWhole(_ cents: Cents) -> String {
        whole.string(from: cents.decimalEuros as NSDecimalNumber) ?? "\(cents.decimalEuros) €"
    }

    /// "+2 140 €" / "−540 €" — variation signée.
    static func signedEurosWhole(_ cents: Cents) -> String {
        let magnitude = Decimal(cents.raw.magnitude) / 100
        let s = whole.string(from: magnitude as NSDecimalNumber) ?? "—"
        if cents.raw > 0 { return "+" + s }
        if cents.raw < 0 { return "−" + s }
        return s
    }
}


extension MoneyFormat {
    static func amount(_ amount: Cents, currency: String, whole: Bool = false) -> String {
        let formatter = NumberFormatter()
        formatter.locale = Locale(identifier: "fr_FR")
        formatter.numberStyle = .currency
        formatter.currencyCode = currency
        let scale = formatter.maximumFractionDigits
        if whole { formatter.maximumFractionDigits = 0; formatter.minimumFractionDigits = 0 }
        var divisor = Decimal(1)
        for _ in 0..<scale { divisor *= 10 }
        return formatter.string(from: (Decimal(amount.raw) / divisor) as NSDecimalNumber) ?? "—"
    }
}


extension MoneyFormat {
    static func exponent(_ currency: String) -> Int {
        let formatter = NumberFormatter(); formatter.numberStyle = .currency; formatter.currencyCode = currency
        return formatter.maximumFractionDigits
    }
    static func input(_ amount: Cents, currency: String) -> String {
        var divisor = Decimal(1)
        for _ in 0..<exponent(currency) { divisor *= 10 }
        return NSDecimalNumber(decimal: Decimal(amount.raw) / divisor).stringValue
    }
}
extension Cents {
    static func parse(_ input: String, currency: String) -> Cents? {
        let normalized = input.filter { !$0.isWhitespace }.replacingOccurrences(of: ",", with: ".")
        guard normalized.range(of: #"^[+-]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)$"#, options: .regularExpression) != nil,
              let decimal = Decimal(string: normalized, locale: Locale(identifier: "en_US_POSIX")) else { return nil }
        let exponent = MoneyFormat.exponent(currency)
        guard normalized.split(separator: ".", omittingEmptySubsequences: false).count <= 2,
              (normalized.split(separator: ".", omittingEmptySubsequences: false).last?.count ?? 0) <= exponent || !normalized.contains(".") else { return nil }
        var multiplier = Decimal(1)
        for _ in 0..<exponent { multiplier *= 10 }
        let scaled = decimal * multiplier
        guard scaled <= Decimal(Int64.max), scaled >= Decimal(Int64.min) else { return nil }
        return Cents(NSDecimalNumber(decimal: scaled).int64Value)
    }
}
