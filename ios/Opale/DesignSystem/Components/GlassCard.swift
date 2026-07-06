import SwiftUI

/// Carte en verre — la surface de base d'Opale (Liquid Glass, iOS 26+).
///
/// À utiliser dans un `GlassEffectContainer` quand plusieurs cartes
/// coexistent à l'écran (fusion/morphing des verres).
struct GlassCard<Content: View>: View {
    var tint: Color?
    var interactive = false
    @ViewBuilder var content: Content

    var body: some View {
        content
            .padding(16)
            .frame(maxWidth: .infinity, alignment: .leading)
            .glassEffect(glass, in: .rect(cornerRadius: 24))
    }

    private var glass: Glass {
        var g = Glass.regular
        if let tint { g = g.tint(tint.opacity(0.35)) }
        if interactive { g = g.interactive() }
        return g
    }
}

/// Pastille d'info compacte (icône + libellé + valeur) posée sur du verre.
struct GlassStat: View {
    var label: String
    var systemImage: String
    var cents: Cents
    var tint: Color = OpaleTheme.accent

    var body: some View {
        GlassCard {
            VStack(alignment: .leading, spacing: 6) {
                Label(label, systemImage: systemImage)
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(.secondary)
                AmountText(cents: cents, style: .whole)
                    .font(.title3.weight(.bold))
                    .lineLimit(1)
                    .minimumScaleFactor(0.55)
            }
        }
    }
}

#Preview("Cartes verre") {
    ZStack {
        OpaleTheme.iridescent.ignoresSafeArea()
        GlassEffectContainer(spacing: 16) {
            VStack(spacing: 16) {
                GlassCard {
                    VStack(alignment: .leading, spacing: 8) {
                        Text("Patrimoine net")
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(.secondary)
                        AmountText(cents: Cents(4_830_000), style: .whole)
                            .font(.system(size: 40, weight: .bold, design: .rounded))
                    }
                }
                HStack(spacing: 16) {
                    GlassStat(label: "Actifs", systemImage: "arrow.up.right", cents: Cents(6_030_000))
                    GlassStat(label: "Dettes", systemImage: "arrow.down.right", cents: Cents(1_200_000), tint: OpaleTheme.loss)
                }
            }
            .padding()
        }
    }
}
