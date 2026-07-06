import SwiftUI

/// Célébration de palier (EF-016) : une pluie de confettis dessinée en
/// Canvas — légère, sans dépendance, jamais bloquante pour les taps.
struct ConfettiView: View {
    /// Une particule déterministe (graine = index).
    nonisolated private struct Particle {
        let x: Double        // position horizontale relative (0…1)
        let delay: Double    // départ décalé (s)
        let speed: Double    // vitesse de chute (fraction d'écran/s)
        let sway: Double     // amplitude du balancement
        let size: Double
        let color: Color
        let spin: Double

        init(index: Int) {
            // Pseudo-aléa déterministe : pas de vrai hasard nécessaire.
            func rand(_ salt: Int) -> Double {
                let v = sin(Double(index * 37 + salt * 101)) * 43758.5453
                return v - v.rounded(.down)
            }
            x = rand(1)
            delay = rand(2) * 0.8
            speed = 0.35 + rand(3) * 0.45
            sway = 12 + rand(4) * 36
            size = 6 + rand(5) * 7
            spin = (rand(7) - 0.5) * 10
            // Couleurs en dur : Particle est nonisolated (OpaleTheme est MainActor).
            let palette: [Color] = [
                Color(red: 0.35, green: 0.72, blue: 0.71), // accent Opale
                Color(red: 0.24, green: 0.71, blue: 0.54), // gain
                .orange, .pink, .purple, .yellow,
            ]
            color = palette[Int(rand(6) * Double(palette.count)) % palette.count]
        }
    }

    nonisolated private static let particles = (0..<90).map { Particle(index: $0) }
    private let start = Date.now

    var body: some View {
        // Qualifié : notre écran Timeline (Patrimoine) masque SwiftUI.TimelineView.
        SwiftUI.TimelineView(.animation) { timeline in
            Canvas { context, size in
                let t = timeline.date.timeIntervalSince(start)
                for p in Self.particles {
                    let elapsed = t - p.delay
                    guard elapsed > 0 else { continue }
                    let y = elapsed * p.speed * size.height - 20
                    guard y < size.height + 20 else { continue }
                    let x = p.x * size.width + sin(elapsed * 3 + p.x * 10) * p.sway
                    var ctx = context
                    ctx.translateBy(x: x, y: y)
                    ctx.rotate(by: .radians(elapsed * p.spin))
                    ctx.opacity = min(1, max(0, 3.5 - elapsed))
                    ctx.fill(
                        Path(roundedRect: CGRect(x: -p.size / 2, y: -p.size / 3,
                                                 width: p.size, height: p.size * 0.66),
                             cornerRadius: 1.5),
                        with: .color(p.color)
                    )
                }
            }
        }
        .allowsHitTesting(false)
        .ignoresSafeArea()
    }
}
