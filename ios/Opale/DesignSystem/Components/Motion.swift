import SwiftUI

// La grammaire de mouvement d'Opale — le « feel » Revolut :
// tout entre en scène avec du ressort, tout répond au doigt, rien ne bouge
// pour ceux qui ont désactivé les animations (reduceMotion).

// MARK: - Entrée en cascade

/// Fait entrer une carte en scène : fondu + translation + net-flou,
/// décalée selon son rang — l'écran « se construit » sous les yeux.
private struct CascadeIn: ViewModifier {
	@Environment(\.accessibilityReduceMotion) private var reduceMotion
	let index: Int
	@State private var shown = false

	func body(content: Content) -> some View {
		content
			.opacity(shown ? 1 : 0)
			.offset(y: shown ? 0 : 26)
			.blur(radius: shown ? 0 : 5)
			.onAppear {
				guard !shown else { return }
				guard !reduceMotion else {
					shown = true
					return
				}
				withAnimation(.spring(duration: 0.55, bounce: 0.18)
					.delay(Double(index) * 0.07)) {
					shown = true
				}
			}
	}
}

extension View {
	/// Entrée en cascade — `index` = rang de la carte (0, 1, 2…).
	func cascadeIn(_ index: Int = 0) -> some View {
		modifier(CascadeIn(index: index))
	}
}

// MARK: - Cartes pressables

/// Style de bouton « carte » : se comprime sous le doigt avec un ressort
/// vif — chaque tap devient tactile.
struct PressableStyle: ButtonStyle {
	func makeBody(configuration: Configuration) -> some View {
		configuration.label
			.scaleEffect(configuration.isPressed ? 0.96 : 1)
			.animation(.snappy(duration: 0.22, extraBounce: 0.06),
			           value: configuration.isPressed)
	}
}

extension ButtonStyle where Self == PressableStyle {
	/// `.buttonStyle(.pressable)` — la carte répond au doigt.
	static var pressable: PressableStyle { PressableStyle() }
}

// MARK: - Dégradé vivant

/// Fait respirer le dégradé irisé des chiffres héros : une oscillation
/// de teinte lente et subtile — vivant sans être agité.
private struct IridescentShimmer: ViewModifier {
	@Environment(\.accessibilityReduceMotion) private var reduceMotion
	@State private var phase = false

	func body(content: Content) -> some View {
		content
			.hueRotation(.degrees(phase ? 9 : -9))
			.onAppear {
				guard !reduceMotion else { return }
				withAnimation(.easeInOut(duration: 5).repeatForever(autoreverses: true)) {
					phase = true
				}
			}
	}
}

extension View {
	/// Dégradé irisé « vivant » pour les montants héros.
	func iridescentShimmer() -> some View {
		modifier(IridescentShimmer())
	}
}
