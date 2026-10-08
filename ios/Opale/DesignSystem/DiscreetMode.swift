import SwiftUI

extension EnvironmentValues {
    /// Mode discret (EF-004) : quand actif, tous les montants sont floutés.
    /// Propagé depuis `SessionStore` à la racine de l'app.
    @Entry var discreetMode: Bool = false
}


/// Masks the complete sensitive region, including labels embedded in charts and narratives.
private struct SensitiveRegion: ViewModifier {
    @Environment(\.discreetMode) private var discreet
    func body(content: Content) -> some View {
        content.blur(radius: discreet ? 9 : 0)
            .accessibilityHidden(discreet)
            .overlay { if discreet { Text("Montants masqués").font(.caption).accessibilityLabel("Montants masqués") } }
    }
}
extension View { func sensitive() -> some View { modifier(SensitiveRegion()) } }
