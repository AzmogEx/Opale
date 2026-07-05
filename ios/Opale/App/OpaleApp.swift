import SwiftUI

/// Point d'entrée d'Opale.
///
/// Architecture MV : pas de view models — les vues lisent des stores
/// `@Observable` injectés par l'environnement (`SessionStore`).
@main
struct OpaleApp: App {
    @State private var session = SessionStore()

    var body: some Scene {
        WindowGroup {
            RootView()
                .environment(session)
                .environment(\.discreetMode, session.discreetMode)
                .tint(OpaleTheme.accent)
        }
    }
}
