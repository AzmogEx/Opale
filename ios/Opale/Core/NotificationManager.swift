import Foundation
import UserNotifications
import BackgroundTasks

/// Notifications locales (EF-053, prolongement) : les alertes critiques du
/// moteur (solde négatif prévu, enveloppe dépassée) arrivent même app fermée,
/// via un rafraîchissement en arrière-plan. Autorisation « provisionnelle » :
/// livraison silencieuse dans le centre de notifications, sans pop-up
/// intrusive — l'utilisateur promeut ou coupe depuis Réglages.
nonisolated enum NotificationManager {
    static let refreshTaskID = "app.opale.ios.refresh"

    /// À appeler au lancement : autorisation provisionnelle + enregistrement
    /// de la tâche d'arrière-plan.
    nonisolated static func bootstrap() {
        // Permission is requested explicitly in Notifications settings.

        BGTaskScheduler.shared.register(forTaskWithIdentifier: refreshTaskID, using: nil) { task in
            guard let refresh = task as? BGAppRefreshTask else {
                task.setTaskCompleted(success: false)
                return
            }
            handleRefresh(refresh)
        }
    }

    /// Planifie le prochain rafraîchissement (~4 h, à la discrétion d'iOS).
    nonisolated static func scheduleRefresh() {
        let request = BGAppRefreshTaskRequest(identifier: refreshTaskID)
        request.earliestBeginDate = Date(timeIntervalSinceNow: 4 * 3600)
        try? BGTaskScheduler.shared.submit(request)
    }

    /// En arrière-plan : interroge le moteur et notifie les alertes critiques.
    nonisolated private static func handleRefresh(_ task: BGAppRefreshTask) {
        scheduleRefresh() // toujours réarmer

        // BGAppRefreshTask n'est pas Sendable mais setTaskCompleted /
        // expirationHandler sont thread-safe (contrat BackgroundTasks).
        nonisolated(unsafe) let task = task
        let work = Task {
            defer { task.setTaskCompleted(success: true) }
            let base = UserDefaults.standard.string(forKey: "opale.baseURL") ?? "http://localhost:8080"
            guard let url = URL(string: base), let token = Keychain.sessionToken(forServer: base) else { return }
            let api = await MainActor.run { APIClient(baseURL: url) { token } }
            guard let profile = try? await api.me(),
                  UserDefaults.standard.integer(forKey: "profile.\(base)|\(profile.id).push.enabled") == 1,
                  let alerts = try? await api.alerts(),
                  Keychain.get("session.token") == token, !Task.isCancelled else { return }
            for alert in alerts where alert.severity == "critical" {
                await postNotification(title: alert.title, body: alert.detail, id: alert.id, profileID: profile.id)
            }
        }
        task.expirationHandler = { work.cancel() }
    }

    /// Publie une notification locale (dédupliquée par identifiant).
    static func postNotification(title: String, body: String, id: String, profileID: String) async {
        let content = UNMutableNotificationContent()
        content.title = "Opale · point à vérifier"
        content.body = "Une alerte est disponible. Ouvre l’app pour la consulter."
        content.userInfo = ["destination": "alerts", "profile_id": profileID]
        content.sound = nil // provisionnel : silencieux par design
        let request = UNNotificationRequest(identifier: id, content: content, trigger: nil)
        try? await UNUserNotificationCenter.current().add(request)
    }
}
