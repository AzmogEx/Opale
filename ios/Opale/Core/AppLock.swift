import SwiftUI
import LocalAuthentication

/// Verrouillage de l'app (ENF-004) : Face ID / Touch ID / code appareil.
///
/// Comportement : quand il est activé, l'app se verrouille au passage en
/// ARRIÈRE-PLAN (retour à l'écran d'accueil) et se déverrouille par
/// biométrie au retour. L'activation elle-même demande Face ID — le retour
/// visuel est immédiat, on sait que ça marche.
@MainActor
@Observable
final class AppLock {
	/// Réglage persistant (activé par défaut). Passer par `setEnabled` pour l'UI.
	private(set) var enabled: Bool
	/// L'app est actuellement verrouillée.
	private(set) var locked = false
    private var lastInteraction = Date.now
    func recordInteraction() { lastInteraction = .now }
    func checkInactivity(now: Date = .now) { if enabled, now.timeIntervalSince(lastInteraction) >= 300 { locked = true } }
	/// Une demande biométrique est en cours (évite les doubles prompts —
	/// le prompt Face ID rend l'app `.inactive`, il ne doit pas se relancer).
	private var authenticating = false

	/// La biométrie est-elle disponible sur cet appareil ?
	let biometryAvailable: Bool
	let biometryLabel: String

	init() {
		enabled = UserDefaults.standard.object(forKey: "lock.enabled") as? Bool ?? true
		locked = (UserDefaults.standard.object(forKey: "lock.enabled") as? Bool ?? true) && Keychain.get("session.token") != nil
		let context = LAContext()
		biometryAvailable = context.canEvaluatePolicy(.deviceOwnerAuthentication, error: nil)
		biometryLabel = switch context.biometryType {
		case .faceID: "Face ID"
		case .touchID: "Touch ID"
		default: "Code de l'appareil"
		}
	}

	/// Active/désactive le verrouillage — avec authentification IMMÉDIATE
	/// dans les deux sens : activer prouve que ça marche, désactiver exige
	/// d'être le propriétaire. Renvoie false si l'authentification échoue.
	@discardableResult
	func setEnabled(_ wanted: Bool) async -> Bool {
		guard wanted != enabled else { return true }
		let reason = wanted ? "Activer le verrouillage d'Opale" : "Désactiver le verrouillage d'Opale"
		guard await authenticate(reason: reason) else { return false }
		enabled = wanted
		UserDefaults.standard.set(wanted, forKey: "lock.enabled")
		return true
	}

	/// Verrouille immédiatement (bouton « Verrouiller maintenant »).
	func lockNow(force: Bool = false) {
		guard enabled || force else { return }
		locked = true
	}

	/// À appeler quand l'app passe en arrière-plan.
	func lockIfEnabled() {
		if enabled { locked = true }
	}

	/// Demande la biométrie pour déverrouiller (au retour au premier plan).
	func unlock() async {
		guard locked, !authenticating else { return }
		if await authenticate(reason: "Déverrouiller Opale") {
			SoundPlayer.play(.unlock)
			withAnimation(.easeOut(duration: 0.3)) { locked = false; recordInteraction() }
		}
	}

    func authenticatedByPIN() { locked = false; recordInteraction() }

	private func authenticate(reason: String) async -> Bool {
		guard biometryAvailable else { return false }
		authenticating = true
		defer { authenticating = false }
		let context = LAContext()
		context.localizedCancelTitle = "Annuler"
		return (try? await context.evaluatePolicy(.deviceOwnerAuthentication,
		                                          localizedReason: reason)) ?? false
	}
}

/// Écran de verrouillage — masque tout le contenu (mode discret ultime).
struct LockScreenView: View {
	let lock: AppLock
    @Environment(SessionStore.self) private var session
    @State private var pin = ""
    @State private var error: String?
    @State private var busy = false

	var body: some View {
		ZStack {
			OpaleTheme.iridescent
				.opacity(0.25)
				.ignoresSafeArea()
			Rectangle()
				.fill(Color(uiColor: .systemBackground))
				.ignoresSafeArea()

			VStack(spacing: 16) {
				Image(systemName: "lock.fill")
					.font(.system(size: 44))
					.foregroundStyle(OpaleTheme.iridescent)
				Text("Opale est verrouillée")
					.font(.headline)
                if !lock.biometryAvailable {
                    SecureField("Code PIN du profil", text: $pin).keyboardType(.numberPad).textFieldStyle(.roundedBorder).frame(maxWidth: 250)
                    Button("Déverrouiller avec le PIN") { Task { await unlockPIN() } }.disabled(busy || pin.isEmpty)
                    if let error { Text(error).font(.caption).foregroundStyle(.red) }
                    Text("Une connexion au serveur est nécessaire sans code de sécurité de l’appareil.").font(.caption).multilineTextAlignment(.center).padding(.horizontal)
                }
				Button {
					Task { await lock.unlock() }
				} label: {
					Label("Déverrouiller", systemImage: "faceid")
						.padding(.horizontal, 8)
				}
				.buttonStyle(.glassProminent)
                .disabled(!lock.biometryAvailable)
                Button("Changer de profil") { session.invalidateSession(); lock.authenticatedByPIN() }
			}
		}
	}
    private func unlockPIN() async {
        guard case .loggedIn(let profile) = session.state else { session.invalidateSession(); lock.authenticatedByPIN(); return }
        busy = true; defer { busy = false }
        do { try await session.login(profileID: profile.id, pin: pin); pin = ""; lock.authenticatedByPIN() }
        catch { self.error = error.localizedDescription }
    }

}
