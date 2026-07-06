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
	/// Réglage persistant (opt-in). Passer par `setEnabled` pour l'UI.
	private(set) var enabled: Bool
	/// L'app est actuellement verrouillée.
	private(set) var locked = false
	/// Une demande biométrique est en cours (évite les doubles prompts —
	/// le prompt Face ID rend l'app `.inactive`, il ne doit pas se relancer).
	private var authenticating = false

	/// La biométrie est-elle disponible sur cet appareil ?
	let biometryAvailable: Bool
	let biometryLabel: String

	init() {
		enabled = UserDefaults.standard.bool(forKey: "lock.enabled")
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
	func lockNow() {
		guard enabled else { return }
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
			withAnimation(.easeOut(duration: 0.3)) { locked = false }
		}
	}

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

	var body: some View {
		ZStack {
			OpaleTheme.iridescent
				.opacity(0.25)
				.ignoresSafeArea()
			Rectangle()
				.fill(.ultraThinMaterial)
				.ignoresSafeArea()

			VStack(spacing: 16) {
				Image(systemName: "lock.fill")
					.font(.system(size: 44))
					.foregroundStyle(OpaleTheme.iridescent)
				Text("Opale est verrouillée")
					.font(.headline)
				Button {
					Task { await lock.unlock() }
				} label: {
					Label("Déverrouiller", systemImage: "faceid")
						.padding(.horizontal, 8)
				}
				.buttonStyle(.glassProminent)
			}
		}
		// Tente le déverrouillage dès l'apparition (retour au premier plan).
		.task { await lock.unlock() }
	}
}
