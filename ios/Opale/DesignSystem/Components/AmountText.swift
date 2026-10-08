import SwiftUI

/// Montant animé — le composant signature d'Opale (EF-011).
///
/// - Les changements de valeur « roulent » chiffre par chiffre
///   (`contentTransition(.numericText)`) avec un ressort doux.
/// - Respecte le mode discret (EF-004) : montant flouté.
/// - Respecte « réduire les animations » (accessibilité).
struct AmountText: View {
    var cents: Cents
    var style: Style = .full
    var currency = "EUR"

    enum Style {
        /// "48 300,00 €"
        case full
        /// "48 300 €" — grands chiffres (héro).
        case whole
        /// "+2 140 €" — variation signée, colorée gain/perte.
        case signedDelta
    }

    @Environment(\.discreetMode) private var discreetMode
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    private var text: String {
        if currency != "EUR" { return MoneyFormat.amount(cents, currency: currency, whole: style == .whole) }
        return switch style {
        case .full: MoneyFormat.euros(cents)
        case .whole: MoneyFormat.eurosWhole(cents)
        case .signedDelta: MoneyFormat.signedEurosWhole(cents)
        }
    }

    var body: some View {
        // Replace the source string too: parent accessibility containers can
        // derive their value from Text before applying its accessibility label.
        Text(discreetMode ? "••••" : text)
            .contentTransition(.numericText(value: Double(cents.raw)))
            .animation(reduceMotion ? nil : .spring(duration: 0.6), value: cents)
            .monospacedDigit()
            .foregroundStyle(style == .signedDelta ? AnyShapeStyle(OpaleTheme.delta(cents)) : AnyShapeStyle(.primary))
            .blur(radius: discreetMode ? 8 : 0)
            .accessibilityLabel(discreetMode ? "Montant masqué" : text)
    }
}

#Preview("Montants") {
    VStack(alignment: .leading, spacing: 16) {
        AmountText(cents: Cents(4_830_000), style: .whole)
            .font(.system(size: 44, weight: .bold, design: .rounded))
        AmountText(cents: Cents(4_830_000), style: .full)
        AmountText(cents: Cents(214_000), style: .signedDelta)
        AmountText(cents: Cents(-54_000), style: .signedDelta)
        AmountText(cents: Cents(4_830_000), style: .whole)
            .environment(\.discreetMode, true)
    }
    .padding()
}
