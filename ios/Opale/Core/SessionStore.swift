import Foundation
import Observation
import UserNotifications

/// Store de session — source de vérité de l'authentification (EF-001/EF-002).
///
/// Détient le client API, le profil connecté et le jeton (dans le trousseau).
/// Injecté à la racine via `.environment(session)`.
@MainActor
@Observable
final class SessionStore {
    enum State {
        case loading
        case loggedOut
        case loggedIn(Profile)
    }

    private(set) var state: State = .loading

    /// Backend imposé par l'application, indépendant des anciennes préférences.
    let baseURLString: String

    /// Mode discret (EF-004) : flouter tous les montants d'un geste.
    var discreetMode = false { didSet {
        WidgetBridge.setDiscreet(discreetMode)
        if profileID != "anonyme" { setPreference(discreetMode ? 1 : 0, "discreet") }
    } }
    private(set) var isOffline = false
    private var generation = UUID()
    private let networkSession: URLSession
    private let clearCachedData: () -> Void
    var refreshID = UUID()
    var selectedTab = "home"
    var showAlerts = false
    func openDestination(_ destination: String) {
        if destination == "alerts" { selectedTab = "home"; showAlerts = true }
        else if ["home", "flows", "wealth", "projection", "assistant"].contains(destination) { selectedTab = destination }
    }
    var profileKey: String { baseURLString + "|" + profileID }
    func changed() { refreshID = UUID() }

    /// Identifiant du profil connecté (clé du cache hors-ligne).
    var profileID: String {
        if case .loggedIn(let profile) = state { return profile.id }
        return "anonyme"
    }

    /// Nom du profil connecté (« — » hors session).
    var profileName: String {
        if case .loggedIn(let profile) = state { return profile.name }
        return "—"
    }

    private nonisolated static let tokenKey = "session.token"
    private nonisolated static let profileKeychainKey = "session.profile"

    init(networkSession: URLSession = .shared, clearCachedData: @escaping () -> Void = { WidgetBridge.clear(); DiskCache.clear() }) {
        self.networkSession = networkSession
        self.clearCachedData = clearCachedData
        baseURLString = AppBackend.url.absoluteString
        UserDefaults.standard.removeObject(forKey: "opale.baseURL")
        // Tests UI : démarrage à l'état déconnecté, déterministe.
        if CommandLine.arguments.contains("--reset-session") {
            Keychain.delete(Self.tokenKey)
        }
    }

    /// Client API construit sur l'URL courante ; lit le jeton du trousseau.
    var api: APIClient {
        let url = AppBackend.url
        let token = Keychain.sessionToken(forServer: baseURLString)
        return APIClient(baseURL: url, urlSession: networkSession) { token }
    }

    /// Restaure la session au lancement : jeton en trousseau + `GET /me`.
    func bootstrap() async {
        guard case .loading = state else { return }
        guard Keychain.sessionToken(forServer: baseURLString) != nil else {
            state = .loggedOut
            return
        }
        let revision = generation
        do {
            let profile: Profile = try await api.me()
            guard revision == generation else { return }
            remember(profile)
            isOffline = false
            state = .loggedIn(profile)
            discreetMode = preferenceInt("discreet", default: 0) == 1
        } catch {
            guard revision == generation else { return }
            if case APIError.notAuthenticated = error {
                invalidateSession()
            } else if let saved = Keychain.get(Self.profileKeychainKey),
                      let data = saved.data(using: .utf8),
                      let cached = try? JSONDecoder().decode(SavedProfile.self, from: data),
                      cached.server == baseURLString {
                isOffline = true
                state = .loggedIn(cached.profile)
                discreetMode = preferenceInt("discreet", default: 0) == 1
            } else {
                state = .loggedOut
            }
        }
    }

    private struct SavedProfile: Codable { let server: String; let profile: Profile }
    private func remember(_ profile: Profile) {
        if let data = try? JSONEncoder().encode(SavedProfile(server: baseURLString, profile: profile)),
           let value = String(data: data, encoding: .utf8) {
            Keychain.set(value, forKey: Self.profileKeychainKey)
        }
    }

    func invalidateSession() {
        generation = UUID()
        Keychain.delete(Self.tokenKey)
        Keychain.delete(Self.profileKeychainKey)
        clearCachedData()
        UNUserNotificationCenter.current().removeAllDeliveredNotifications()
        UNUserNotificationCenter.current().removeAllPendingNotificationRequests()
        state = .loggedOut
        selectedTab = "home"
        showAlerts = false
        isOffline = false
    }

    func revalidate() async {
        guard case .loggedIn = state else { return }
        let revision = generation
        do {
            let profile = try await api.me()
            guard revision == generation else { return }
            remember(profile)
            state = .loggedIn(profile)
            discreetMode = preferenceInt("discreet", default: 0) == 1
            isOffline = false
        } catch {
            guard revision == generation else { return }
            if case APIError.notAuthenticated = error { invalidateSession() }
            else { isOffline = true }
        }
    }

    func login(profileID: String, pin: String) async throws {
        let res = try await api.login(profileID: profileID, pin: pin)
        generation = UUID()
        guard Keychain.set(res.token, forKey: Self.tokenKey) else { throw APIError.badStatus(503, message: "Le trousseau sécurisé est indisponible. Relance Opale sur un appareil déverrouillé.") }
        remember(res.profile)
        isOffline = false
        state = .loggedIn(res.profile)
        discreetMode = preferenceInt("discreet", default: 0) == 1
    }

    func startDemo() async throws {
        struct Demo: Decodable { let profile: Profile; let token: String }
        let result: Demo = try await api.request("POST", "/v1/profiles/demo", authenticated: false)
        generation = UUID()
        guard Keychain.set(result.token, forKey: Self.tokenKey) else { throw APIError.badStatus(503, message: "Le trousseau sécurisé est indisponible. Relance Opale sur un appareil déverrouillé.") }
        remember(result.profile)
        isOffline = false
        state = .loggedIn(result.profile)
        discreetMode = preferenceInt("discreet", default: 0) == 1
    }

    func createProfileAndLogin(name: String, pin: String) async throws {
        let profile = try await api.createProfile(name: name, pin: pin)
        try await login(profileID: profile.id, pin: pin)
    }

    func logout() async {
        try? await PushRegistration.revoke(api: api)
        try? await api.logout()
        invalidateSession()
    }
}



extension SessionStore {
    func preferenceInt(_ name: String, default fallback: Int) -> Int {
        UserDefaults.standard.object(forKey: "profile.\(profileKey).\(name)") as? Int ?? fallback
    }
    func setPreference(_ value: Int, _ name: String) { UserDefaults.standard.set(value, forKey: "profile.\(profileKey).\(name)") }
}
