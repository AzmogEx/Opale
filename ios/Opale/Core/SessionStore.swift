import Foundation
import Observation

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

    /// URL du backend — modifiable depuis l'écran de connexion (homelab).
    var baseURLString: String {
        didSet { UserDefaults.standard.set(baseURLString, forKey: "opale.baseURL") }
    }

    /// Mode discret (EF-004) : flouter tous les montants d'un geste.
    var discreetMode = false

    private nonisolated static let tokenKey = "session.token"

    init() {
        baseURLString = UserDefaults.standard.string(forKey: "opale.baseURL")
            ?? "http://localhost:8080"
        // Tests UI : démarrage à l'état déconnecté, déterministe.
        if CommandLine.arguments.contains("--reset-session") {
            Keychain.delete(Self.tokenKey)
        }
    }

    /// Client API construit sur l'URL courante ; lit le jeton du trousseau.
    var api: APIClient {
        let url = URL(string: baseURLString) ?? URL(string: "http://localhost:8080")!
        return APIClient(baseURL: url) { Keychain.get(Self.tokenKey) }
    }

    /// Restaure la session au lancement : jeton en trousseau + `GET /me`.
    func bootstrap() async {
        guard case .loading = state else { return }
        guard Keychain.get(Self.tokenKey) != nil else {
            state = .loggedOut
            return
        }
        do {
            let profile: Profile = try await api.me()
            state = .loggedIn(profile)
        } catch {
            Keychain.delete(Self.tokenKey)
            state = .loggedOut
        }
    }

    func login(profileID: String, pin: String) async throws {
        let res = try await api.login(profileID: profileID, pin: pin)
        Keychain.set(res.token, forKey: Self.tokenKey)
        state = .loggedIn(res.profile)
    }

    func createProfileAndLogin(name: String, pin: String) async throws {
        let profile = try await api.createProfile(name: name, pin: pin)
        try await login(profileID: profile.id, pin: pin)
    }

    func logout() async {
        try? await api.logout()
        Keychain.delete(Self.tokenKey)
        WidgetBridge.clear() // le widget ne doit pas survivre à la session
        state = .loggedOut
    }
}

