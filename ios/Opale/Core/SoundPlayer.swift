import Foundation
import AudioToolbox

/// Sons discrets (cahier des charges §7 — option) : de petits sons système
/// satisfaisants sur les moments clés. DÉSACTIVÉ par défaut, opt-in dans
/// les Réglages — le silence est la valeur par défaut d'une app de patrimoine.
nonisolated enum SoundPlayer {
	private static let enabledKey = "sound.enabled"

	static var enabled: Bool {
		get { UserDefaults.standard.bool(forKey: enabledKey) }
		set { UserDefaults.standard.set(newValue, forKey: enabledKey) }
	}

	/// Les moments sonores d'Opale (identifiants de sons système).
	enum Event: SystemSoundID {
		case success = 1407   // palier franchi, célébration
		case send = 1004      // question envoyée à l'assistant
		case unlock = 1101    // déverrouillage
	}

	/// Joue le son si l'option est active.
	static func play(_ event: Event) {
		guard enabled else { return }
		AudioServicesPlaySystemSound(event.rawValue)
	}
}
