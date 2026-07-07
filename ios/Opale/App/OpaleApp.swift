import SwiftUI

/// Point d'entrée d'Opale.
///
/// Architecture MV : pas de view models — les vues lisent des stores
/// `@Observable` injectés par l'environnement (`SessionStore`).
@main
struct OpaleApp: App {
    @State private var session = SessionStore()
    @State private var lock = AppLock()
    @Environment(\.scenePhase) private var scenePhase

    var body: some Scene {
        WindowGroup {
            RootView()
                .environment(session)
                .environment(lock)
                .environment(\.discreetMode, session.discreetMode)
                .tint(OpaleTheme.accent)
                .fontDesign(.rounded) // typo signature — chaleureuse, lisible
                // L'app est française : dates et nombres en français, quel
                // que soit le réglage de langue de l'appareil.
                .environment(\.locale, Locale(identifier: "fr_FR"))
                // Verrouillage (ENF-004) : masque le contenu dès qu'on quitte.
                .overlay {
                    if lock.locked {
                        LockScreenView(lock: lock)
                    }
                }
                .onChange(of: scenePhase) { _, phase in
                    switch phase {
                    case .background:
                        if case .loggedIn = session.state { lock.lockIfEnabled() }
                        NotificationManager.scheduleRefresh()
                    case .active:
                        // Le widget a pu changer le mode discret (AppIntent).
                        session.discreetMode = WidgetBridge.discreet()
                        Task { await lock.unlock() }
                    default:
                        break
                    }
                }
                .task { NotificationManager.bootstrap() }
        }
    }
}
