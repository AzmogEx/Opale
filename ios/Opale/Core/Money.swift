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
        var s = input.trimmingCharacters(in: .whitespaces)
            .replacingOccurrences(of: " ", with: "")   // espace fine (fr)
            .replacingOccurrences(of: " ", with: "")
            .replacingOccurrences(of: "€", with: "")
            .replacingOccurrences(of: ",", with: ".")
        guard !s.isEmpty else { return nil }

        let negative = s.hasPrefix("-")
        if negative { s.removeFirst() }
        if s.hasPrefix("+") { s.removeFirst() }

        let parts = s.split(separator: ".", omittingEmptySubsequences: false)
        guard parts.count <= 2, let whole = Int64(parts[0].isEmpty ? "0" : parts[0]) else { return nil }

        var frac: Int64 = 0
        if parts.count == 2 {
            var f = String(parts[1])
            guard f.count <= 2, f.allSatisfy(\.isNumber) else { return nil }
            if f.count == 1 { f += "0" }
            if f.isEmpty { f = "0" }
            guard let parsed = Int64(f) else { return nil }
            frac = parsed
        }

        let cents = whole * 100 + frac
        return Cents(negative ? -cents : cents)
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
        let s = eurosWhole(Cents(abs(cents.raw)))
        if cents.raw > 0 { return "+" + s }
        if cents.raw < 0 { return "−" + s }
        return s
    }
}
