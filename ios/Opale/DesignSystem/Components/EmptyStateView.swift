import SwiftUI

/// État vide signature — plus d'écran « système » : un symbole irisé qui
/// respire sur du verre, un titre, un message qui dit quoi faire.
struct EmptyStateView: View {
	let icon: String
	let title: String
	let message: String

	@Environment(\.accessibilityReduceMotion) private var reduceMotion

	var body: some View {
		GlassCard {
			VStack(spacing: 12) {
				ZStack {
					Circle()
						.fill(OpaleTheme.iridescent)
						.opacity(0.14)
						.frame(width: 84, height: 84)
					Image(systemName: icon)
						.font(.system(size: 34, weight: .medium))
						.foregroundStyle(OpaleTheme.iridescent)
						.symbolEffect(.breathe, options: .repeating,
						              isActive: !reduceMotion)
				}
				Text(title)
					.font(.headline)
				Text(message)
					.font(.subheadline)
					.foregroundStyle(.secondary)
					.multilineTextAlignment(.center)
			}
			.frame(maxWidth: .infinity)
			.padding(.vertical, 12)
		}
	}
}
