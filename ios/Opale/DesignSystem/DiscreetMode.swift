import SwiftUI

extension EnvironmentValues {
    /// Mode discret (EF-004) : quand actif, tous les montants sont floutés.
    /// Propagé depuis `SessionStore` à la racine de l'app.
    @Entry var discreetMode: Bool = false
}
