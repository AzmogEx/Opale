import Foundation
import WidgetKit

/// Pont vers le widget (P7) : l'app écrit un instantané du patrimoine dans
/// l'App Group ; le widget le lit sans réseau ni jeton. Aucune donnée ne
/// quitte l'appareil — et le mode discret du widget suit celui de l'app.
nonisolated enum WidgetBridge {
    static let suiteName = "group.app.opale.shared"

    /// Publie le patrimoine net + une mini-courbe (12 derniers points).
    static func publish(netWorthCents: Int64, historyCents: [Int64]) {
        guard let defaults = UserDefaults(suiteName: suiteName) else { return }
        defaults.set(netWorthCents, forKey: "netWorthCents")
        defaults.set(historyCents.suffix(12).map { NSNumber(value: $0) }, forKey: "historyCents")
        defaults.set(Date.now.timeIntervalSince1970, forKey: "updatedAt")
        WidgetCenter.shared.reloadAllTimelines()
    }

    /// Efface l'instantané (déconnexion).
    static func clear() {
        guard let defaults = UserDefaults(suiteName: suiteName) else { return }
        defaults.removeObject(forKey: "netWorthCents")
        defaults.removeObject(forKey: "historyCents")
        defaults.removeObject(forKey: "updatedAt")
        WidgetCenter.shared.reloadAllTimelines()
    }
}
