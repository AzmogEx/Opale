import SwiftUI

/// Thème d'Opale — identité inspirée de la pierre : reflets nacrés, irisés.
/// (Palette de base P1 ; le système de thèmes complet arrive avec EF-005.)
enum OpaleTheme {
    /// Couleur d'accent signature (bleu-vert opale).
    static let accent = Color(red: 0.35, green: 0.72, blue: 0.71)

    /// Dégradé irisé signature, utilisé pour le chiffre du patrimoine
    /// et les surfaces « héro ».
    static let iridescent = LinearGradient(
        colors: [
            Color(red: 0.42, green: 0.78, blue: 0.75),
            Color(red: 0.51, green: 0.62, blue: 0.87),
            Color(red: 0.78, green: 0.57, blue: 0.79),
        ],
        startPoint: .topLeading,
        endPoint: .bottomTrailing
    )

    /// Vert « gain » / rouge « perte » pour les variations.
    static let gain = Color(red: 0.24, green: 0.71, blue: 0.54)
    static let loss = Color(red: 0.88, green: 0.36, blue: 0.38)

    /// Couleur de variation selon le signe (en centimes).
    static func delta(_ cents: Cents) -> Color {
        if cents.raw > 0 { return gain }
        if cents.raw < 0 { return loss }
        return .secondary
    }
}
