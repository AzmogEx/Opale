import SwiftUI

/// Point d'entrée d'Opale.
///
/// Architecture MV : pas de view models — les vues lisent des stores
/// `@Observable` injectés par l'environnement (`SessionStore`).
@main
struct OpaleApp: App {
    @UIApplicationDelegateAdaptor(OpaleAppDelegate.self) private var appDelegate
    @State private var session = SessionStore()
    @State private var lock = AppLock()
    @State private var pendingPush: PushRoute?
    @AppStorage("appearance.mode") private var appearance = "system"
    @AppStorage("appearance.accent") private var accent = "opale"
    @Environment(\.scenePhase) private var scenePhase

    var body: some Scene {
        WindowGroup {
            Group {
                if lock.locked { LockScreenView(lock: lock) }
                else { RootView() }
            }
                .id(lock.locked)
                .environment(session)
                .environment(lock)
                .environment(\.discreetMode, session.discreetMode)
                .tint(OpaleTheme.color(for: accent))
                .preferredColorScheme(appearance == "dark" ? .dark : appearance == "light" ? .light : nil)
                .background(ActivityObserver().allowsHitTesting(false))
                .onReceive(NotificationCenter.default.publisher(for: .opaleInteraction)) { _ in lock.recordInteraction() }
                .onReceive(NotificationCenter.default.publisher(for: UITextField.textDidChangeNotification)) { _ in lock.recordInteraction() }
                .onReceive(NotificationCenter.default.publisher(for: UITextView.textDidChangeNotification)) { _ in lock.recordInteraction() }
                .task {
                    while !Task.isCancelled {
                        do { try await Task.sleep(for: .seconds(30)) } catch { return }
                        if scenePhase == .active, case .loggedIn = session.state { lock.checkInactivity() }
                    }
                }
                .labeledContentStyle(AdaptiveLabeledContentStyle())
                .fontDesign(.rounded) // typo signature — chaleureuse, lisible
                // L'app est française : dates et nombres en français, quel
                // que soit le réglage de langue de l'appareil.
                .environment(\.locale, Locale(identifier: "fr_FR"))
                .environment(\.calendar, Calendar.opale)
                .environment(\.timeZone, TimeZone(identifier: "Europe/Paris")!)
                // Verrouillage (ENF-004) : masque le contenu dès qu'on quitte.
                .accessibilityHidden(scenePhase != .active)
                .overlay {
                    if scenePhase != .active {
                        Color(uiColor: .systemBackground).ignoresSafeArea()
                            .overlay(Image(systemName: "lock.shield.fill").font(.largeTitle))
                    }
                }
                .environment(session)
                .environment(lock)
                .onChange(of: scenePhase) { _, phase in
                    switch phase {
                    case .background:
                        if case .loggedIn = session.state { lock.lockIfEnabled() }
                        NotificationManager.scheduleRefresh()
                    case .active:
                        // Le widget a pu changer le mode discret (AppIntent).
                        session.discreetMode = WidgetBridge.discreet()
                        Task { await session.revalidate() }
                    default:
                        break
                    }
                }
                .onChange(of: session.isOffline) { _, offline in if offline { lock.lockNow(force: true) } }
                .onReceive(NotificationCenter.default.publisher(for: .opaleSessionExpired)) { notice in
                    if let failedToken = notice.object as? String, failedToken == Keychain.get("session.token") { session.invalidateSession() }
                }
                .onChange(of: session.profileID) { _, id in
                    if let route = pendingPush, id != "anonyme" { if let destination = route.destination(for: id) { session.openDestination(destination) }; pendingPush = nil }
                    if id != "anonyme", session.preferenceInt("push.enabled", default: 0) == 1 { UIApplication.shared.registerForRemoteNotifications() }
                }
                .onReceive(NotificationCenter.default.publisher(for: .opalePushToken)) { _ in
                    if case .loggedIn = session.state, session.preferenceInt("push.enabled", default: 0) == 1 { Task { try? await PushRegistration.sync(api: session.api) } }
                }
                 .onReceive(NotificationCenter.default.publisher(for: .opaleOpenDestination)) { notice in
                    guard let route = notice.object as? PushRoute else { return }
                    if session.profileID == "anonyme" { pendingPush = route }
                    else if let destination = route.destination(for: session.profileID) { session.openDestination(destination) }
                }
                .onOpenURL { url in if url.scheme == "opale" { session.openDestination(url.host ?? "home") } }
                .task { NotificationManager.bootstrap(); await session.bootstrap() }
        }
    }
}
