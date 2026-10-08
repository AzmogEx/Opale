import Foundation
import UIKit
import UserNotifications

struct PushRoute: Sendable {
    let profileID: String
    let destination: String
    nonisolated func destination(for currentProfile: String) -> String? {
        guard profileID == currentProfile, ["home", "flows", "wealth", "projection", "assistant", "alerts"].contains(destination) else { return nil }
        return destination
    }
}
extension Notification.Name {
    static let opalePushToken = Notification.Name("opale.push.token")
    static let opaleOpenDestination = Notification.Name("opale.push.destination")
}

/// APNs belongs to the device; registration on the server belongs to the authenticated profile.
final class OpaleAppDelegate: NSObject, UIApplicationDelegate, UNUserNotificationCenterDelegate {
    func application(_ application: UIApplication, didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]? = nil) -> Bool {
        UNUserNotificationCenter.current().delegate = self
        return true
    }
    func application(_ application: UIApplication, didRegisterForRemoteNotificationsWithDeviceToken deviceToken: Data) {
        let token = deviceToken.map { String(format: "%02x", $0) }.joined()
        Keychain.set(token, forKey: "push.deviceToken")
        UserDefaults.standard.removeObject(forKey: "push.registrationError")
        NotificationCenter.default.post(name: .opalePushToken, object: token)
    }
    func application(_ application: UIApplication, didFailToRegisterForRemoteNotificationsWithError error: Error) {
        UserDefaults.standard.set("Enregistrement APNs indisponible : \(error.localizedDescription)", forKey: "push.registrationError")
    }
    nonisolated func userNotificationCenter(_ center: UNUserNotificationCenter, didReceive response: UNNotificationResponse) async {
        let destination = response.notification.request.content.userInfo["destination"] as? String ?? "home"
        guard let profileID = response.notification.request.content.userInfo["profile_id"] as? String else { return }
        let route = PushRoute(profileID: profileID, destination: destination)
        await MainActor.run { NotificationCenter.default.post(name: .opaleOpenDestination, object: route) }
    }
    nonisolated func userNotificationCenter(_ center: UNUserNotificationCenter, willPresent notification: UNNotification) async -> UNNotificationPresentationOptions { [.badge] }
}

enum PushRegistration {
    static func enable() async throws {
        let granted = try await UNUserNotificationCenter.current().requestAuthorization(options: [.alert, .badge, .sound])
        guard granted else { throw APIError.badStatus(403, message: "Notifications refusées. Tu peux les autoriser dans Réglages iOS.") }
        UIApplication.shared.registerForRemoteNotifications()
    }
    static func sync(api: APIClient) async throws {
        guard let token = Keychain.get("push.deviceToken") else { return }
        try await api.registerPushToken(token)
    }
    static func revoke(api: APIClient) async throws {
        guard let token = Keychain.get("push.deviceToken") else { return }
        struct Body: Encodable { let token: String }
        let _: APIClient.EmptyResponse = try await api.request("DELETE", "/v1/push/register", body: Body(token: token))
    }
}
