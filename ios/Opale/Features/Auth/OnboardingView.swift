import SwiftUI

/// Onboarding — trois écrans au tout premier lancement (aucun profil) :
/// la promesse, la confidentialité, le moteur. Court, beau, désactivable.
struct OnboardingView: View {
	var onDone: () -> Void

	@State private var page = 0
	@Environment(\.accessibilityReduceMotion) private var reduceMotion

	var body: some View {
		ZStack {
			OpaleBackdrop()

			VStack(spacing: 24) {
				TabView(selection: $page) {
					pageView(
						icon: "circle.hexagongrid.fill",
						title: "Combien tu vaux,\net où tu vas",
						message: "Opale n'est pas une app de budget : c'est ton cockpit patrimonial. Patrimoine net, trajectoire, date d'indépendance — l'essentiel, en un coup d'œil."
					)
					.tag(0)

					pageView(
						icon: "lock.shield.fill",
						title: "Privé,\npar construction",
						message: "Tes données vivent chez toi (ton serveur). L'IA tourne d'abord sur ton iPhone, puis sur ton homelab — le cloud ne voit jamais que des montants arrondis et anonymes, avec ton accord."
					)
					.tag(1)

					pageView(
						icon: "function",
						title: "Le moteur calcule,\nl'IA explique",
						message: "Chaque chiffre vient d'un moteur de calcul exact (jamais d'arrondi flottant, jamais d'invention). L'intelligence artificielle ne fait que t'expliquer ce qu'il a calculé."
					)
					.tag(2)
				}
				.tabViewStyle(.page(indexDisplayMode: .always))
				.indexViewStyle(.page(backgroundDisplayMode: .always))
				.animation(reduceMotion ? nil : .snappy, value: page)

				Button {
					if page < 2 {
						withAnimation(.snappy) { page += 1 }
					} else {
						onDone()
					}
				} label: {
					Text(page < 2 ? "Continuer" : "Créer mon profil")
						.font(.headline)
						.frame(maxWidth: .infinity)
						.padding(.vertical, 6)
				}
				.buttonStyle(.glassProminent)
				.padding(.horizontal, 32)
				.sensoryFeedback(.impact(weight: .light), trigger: page)

				Button("Passer") { onDone() }
					.font(.footnote)
					.foregroundStyle(.secondary)
					.padding(.bottom, 12)
			}
		}
	}

	@ViewBuilder
	private func pageView(icon: String, title: String, message: String) -> some View {
		VStack(spacing: 20) {
			Spacer()
			ZStack {
				Circle()
					.fill(OpaleTheme.iridescent)
					.opacity(0.15)
					.frame(width: 130, height: 130)
				Image(systemName: icon)
					.font(.system(size: 54, weight: .medium))
					.foregroundStyle(OpaleTheme.iridescent)
					.symbolEffect(.breathe, options: .repeating, isActive: !reduceMotion)
			}
			Text(title)
				.font(.system(size: 32, weight: .bold, design: .rounded))
				.multilineTextAlignment(.center)
				.foregroundStyle(OpaleTheme.iridescent)
			Text(message)
				.font(.body)
				.foregroundStyle(.secondary)
				.multilineTextAlignment(.center)
				.padding(.horizontal, 36)
			Spacer()
		}
	}
}
