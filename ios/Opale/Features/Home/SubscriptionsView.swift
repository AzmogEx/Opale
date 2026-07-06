import SwiftUI

/// Abonnements — le gestionnaire d'abonnements automatique.
///
/// Le moteur détecte les prélèvements récurrents (EF-026) ; cette page les
/// transforme en décisions : combien ça coûte VRAIMENT (par an), quand
/// tombe le prochain, et surtout — « résilie ça et ta liberté avance de
/// N mois » (recalcul d'indépendance par le moteur, EIA-040).
struct SubscriptionsView: View {
	@Environment(SessionStore.self) private var session

	@State private var subscriptions: [SubscriptionStatus] = []
	@State private var totalMonthly: Cents = .zero
	@State private var totalYearly: Cents = .zero
	@State private var loaded = false
	@State private var errorMessage: String?

	var body: some View {
		ZStack {
			OpaleBackdrop()

			ScrollView {
				GlassEffectContainer(spacing: 16) {
					VStack(spacing: 16) {
						if loaded && subscriptions.isEmpty {
							EmptyStateView(
								icon: "repeat.circle",
								title: "Aucun abonnement détecté",
								message: "Le moteur repère les prélèvements réguliers dès qu'il voit 3 occurrences (importe quelques mois de relevés)."
							)
						} else if !subscriptions.isEmpty {
							heroCard
								.cascadeIn(0)
							ForEach(Array(subscriptions.enumerated()), id: \.element.id) { index, sub in
								subscriptionCard(sub)
									.cascadeIn(index + 1)
							}
							insightFooter
						} else if let errorMessage {
							EmptyStateView(icon: "bolt.horizontal.circle",
							               title: "Impossible de charger", message: errorMessage)
						} else {
							ProgressView().frame(minHeight: 200)
						}
					}
					.padding(.horizontal)
					.padding(.bottom, 24)
				}
			}
			.scrollEdgeEffectStyle(.soft, for: .top)
		}
		.navigationTitle("Abonnements")
		.navigationBarTitleDisplayMode(.inline)
		.task { await load() }
		.refreshable { await load() }
	}

	// MARK: - Héro : le vrai coût

	private var heroCard: some View {
		GlassCard {
			VStack(alignment: .leading, spacing: 8) {
				Text("Tes abonnements te coûtent")
					.font(.footnote.weight(.semibold))
					.foregroundStyle(.secondary)
					.textCase(.uppercase)

				HStack(alignment: .firstTextBaseline, spacing: 8) {
					AmountText(cents: totalMonthly, style: .whole)
						.font(.system(size: 40, weight: .bold, design: .rounded))
						.foregroundStyle(OpaleTheme.iridescent)
						.iridescentShimmer()
					Text("/ mois")
						.font(.headline)
						.foregroundStyle(.secondary)
				}

				HStack(spacing: 6) {
					Image(systemName: "calendar")
						.font(.caption)
						.foregroundStyle(OpaleTheme.accent)
					Text("soit")
						.font(.subheadline)
						.foregroundStyle(.secondary)
					AmountText(cents: totalYearly, style: .whole)
						.font(.subheadline.weight(.bold))
					Text("par an")
						.font(.subheadline)
						.foregroundStyle(.secondary)
				}
			}
		}
	}

	// MARK: - Une carte par abonnement

	@ViewBuilder
	private func subscriptionCard(_ sub: SubscriptionStatus) -> some View {
		GlassCard {
			VStack(alignment: .leading, spacing: 10) {
				HStack(spacing: 12) {
					ZStack {
						Circle()
							.fill(OpaleTheme.iridescent)
							.opacity(0.18)
						Image(systemName: "repeat")
							.font(.subheadline.weight(.semibold))
							.foregroundStyle(OpaleTheme.iridescent)
					}
					.frame(width: 38, height: 38)

					VStack(alignment: .leading, spacing: 2) {
						Text(sub.label)
							.font(.body.weight(.semibold))
							.lineLimit(1)
						Text("\(periodicityLabel(sub.periodicity)) · prochain le \(sub.nextDate.formatted(.dateTime.day().month(.abbreviated)))")
							.font(.caption)
							.foregroundStyle(.secondary)
					}
					Spacer()
					VStack(alignment: .trailing, spacing: 2) {
						AmountText(cents: sub.monthlyCost, style: .whole)
							.font(.callout.weight(.bold))
						Text("/ mois")
							.font(.caption2)
							.foregroundStyle(.tertiary)
					}
				}

				// LE chiffre qui fait réfléchir.
				HStack(spacing: 6) {
					Image(systemName: "bird")
						.font(.caption)
						.foregroundStyle(OpaleTheme.accent)
					if sub.freedomGainMonths > 0 {
						Text("Résilié, ta liberté avance de **\(sub.freedomGainMonths) mois**")
							.font(.caption)
							.foregroundStyle(.secondary)
					} else {
						Text("Soit \(MoneyFormat.eurosWhole(sub.yearlyCost)) par an qui ne s'investissent pas")
							.font(.caption)
							.foregroundStyle(.secondary)
					}
				}
				.padding(.top, 2)
			}
		}
	}

	private var insightFooter: some View {
		Label("Détection automatique par le moteur — un abonnement disparaît de lui-même quand les prélèvements s'arrêtent.",
		      systemImage: "sparkle")
			.font(.caption2)
			.foregroundStyle(.tertiary)
			.padding(.horizontal, 6)
	}

	private func periodicityLabel(_ p: String) -> String {
		switch p {
		case "weekly": "Hebdomadaire"
		case "monthly": "Mensuel"
		case "quarterly": "Trimestriel"
		case "yearly": "Annuel"
		default: p.capitalized
		}
	}

	private func load() async {
		do {
			let result = try await session.api.subscriptions()
			subscriptions = result.items
			totalMonthly = result.monthly
			totalYearly = result.yearly
			errorMessage = nil
		} catch {
			errorMessage = error.localizedDescription
		}
		loaded = true
	}
}
