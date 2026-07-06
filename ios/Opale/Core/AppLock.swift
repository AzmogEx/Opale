import SwiftUI
import LocalAuthentication

/// Verrouillage de l'app (ENF-004) : Face ID / Touch ID / code appareil.
///
/// Opt-in (réglage dans le menu Assistant). Quand il est actif, l'app se
/// verrouille au passage en arrière-plan et se déverrouille par biométrie
/// au retour. Le lancement à froid après connexion PIN n'est PAS verrouillé
/// (le PIN vient d'être saisi) — ce qui préserve aussi les tests UI.
@MainActor
@Observable
final class AppLock {
    /// Réglage persistant (opt-in).
    var enabled: Bool {
        didSet { UserDefaults.standard.set(enabled, forKey: "lock.enabled") }
    }
    /// L'app est actuellement verrouillée.
    private(set) var locked = false
    /// La biométrie est-elle disponible sur cet appareil ?
    let biometryAvailable: Bool
    let biometryLabel: String

    init() {
        enabled = UserDefaults.standard.bool(forKey: "lock.enabled")
        let context = LAContext()
        var error: NSError?
        biometryAvailable = context.canEvaluatePolicy(.deviceOwnerAuthentication, error: &error)
        biometryLabel = switch context.biometryType {
        case .faceID: "Face ID"
        case .touchID: "Touch ID"
        default: "Code de l'appareil"
        }
    }

    /// À appeler quand l'app passe en arrière-plan.
    func lockIfEnabled() {
        if enabled { locked = true }
    }

    /// Demande la biométrie (ou le code appareil) pour déverrouiller.
    func unlock() async {
        guard locked else { return }
        let context = LAContext()
        context.localizedCancelTitle = "Annuler"
        do {
            let ok = try await context.evaluatePolicy(
                .deviceOwnerAuthentication,
                localizedReason: "Déverrouiller Opale"
            )
            if ok { locked = false }
        } catch {
            // Refus/échec : l'app reste verrouillée, l'utilisateur peut réessayer.
        }
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
    }
}
