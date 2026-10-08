import SwiftUI
import UserNotifications

struct PushSettingsView: View {
    @Environment(SessionStore.self) private var session
    @State private var state = "Vérification…"
    @State private var error: String?
    @State private var busy = false
    var body: some View {
        Form {
            Section("Notifications de ce profil") {
                Text(state)
                Text("Les notifications restent génériques : ouvre Opale et déverrouille-la pour consulter les montants.").font(.caption)
                Button("Activer les notifications") { Task { await enable() } }.disabled(busy)
                Button("Désactiver pour ce profil", role: .destructive) { Task { await disable() } }.disabled(busy)
                Link("Réglages iOS", destination: URL(string: UIApplication.openSettingsURLString)!)
            }
            ToolError(message: error)
        }.navigationTitle("Notifications").task { await refresh() }
        .onReceive(NotificationCenter.default.publisher(for: .opalePushToken)) { _ in guard session.preferenceInt("push.enabled", default: 0) == 1 else { return }; Task { do { try await PushRegistration.sync(api: session.api); state = "Appareil enregistré pour ce profil" } catch { self.error = error.localizedDescription } } }
    }
    private func refresh() async {
        let settings = await UNUserNotificationCenter.current().notificationSettings()
        state = settings.authorizationStatus == .authorized ? "Autorisées sur cet appareil" : settings.authorizationStatus == .provisional ? "Livraison silencieuse autorisée" : "Non autorisées"
        if session.preferenceInt("push.enabled", default: 0) == 0 { state += " · désactivées pour ce profil" }
        error = UserDefaults.standard.string(forKey: "push.registrationError")
    }
    private func enable() async { busy = true; defer { busy = false }; do { try await PushRegistration.enable(); try await PushRegistration.sync(api: session.api); session.setPreference(1, "push.enabled"); error = nil; await refresh() } catch { self.error = error.localizedDescription } }
    private func disable() async { busy = true; defer { busy = false }; do { try await PushRegistration.revoke(api: session.api); session.setPreference(0, "push.enabled"); state = "Désactivées pour ce profil"; error = nil } catch { self.error = error.localizedDescription } }
}
