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
	@State private var declaredSubscriptions: [CalendarRule] = []
	@State private var declaredCurrencies: [String: String] = [:]
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
						if loaded && subscriptions.isEmpty && declaredSubscriptions.isEmpty && errorMessage == nil {
							EmptyStateView(
								icon: "repeat.circle",
								title: "Aucun abonnement détecté",
								message: "Le moteur repère les prélèvements réguliers dès qu'il voit 3 occurrences (importe quelques mois de relevés)."
							)
						}
						if !declaredSubscriptions.isEmpty {
                            declaredSection
                        }
						if !subscriptions.isEmpty {
                            Text("Détectés dans les opérations")
                                .font(.headline).frame(maxWidth: .infinity, alignment: .leading)
							heroCard
								.cascadeIn(0)
							ForEach(Array(subscriptions.enumerated()), id: \.element.id) { index, sub in
								subscriptionCard(sub)
									.cascadeIn(index + 1)
							}
							insightFooter
						}
                        if let errorMessage {
							EmptyStateView(icon: "bolt.horizontal.circle",
							               title: "Impossible de charger", message: errorMessage)
						} else if !loaded {
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
		.task(id: session.refreshID) { await load() }
		.refreshable { await load() }
	}

    private var declaredSection: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("Abonnements déclarés").font(.headline)
            Text("Saisis pendant la configuration. Les montants et dates ci-dessous suivent les séries actuelles du calendrier.")
                .font(.caption).foregroundStyle(.secondary)
            ForEach(declaredSubscriptions) { rule in
                GlassCard {
                    VStack(alignment: .leading, spacing: 6) {
                        HStack(alignment: .firstTextBaseline) {
                            Text(rule.label).font(.body.weight(.medium))
                            Spacer()
                            if let currency = declaredCurrencies[rule.asset_id] {
                                AmountText(cents: rule.amount_cents, style: .full, currency: currency)
                                    .font(.callout.weight(.semibold))
                            } else {
                                Text("Devise indisponible").font(.caption).foregroundStyle(.secondary)
                            }
                        }
                        Text("\(periodicityLabel(rule.frequency)) · depuis le \(Date.fromOpaleDay(rule.date)?.opaleFormatted(.dateTime.day().month(.wide).year()) ?? rule.date)")
                            .font(.caption).foregroundStyle(.secondary)
                        if !rule.active {
                            Label("Série arrêtée", systemImage: "pause.circle")
                                .font(.caption).foregroundStyle(.secondary)
                        } else if let end = rule.end_date {
                            Text("Fin le \(Date.fromOpaleDay(end)?.opaleFormatted(.dateTime.day().month(.wide).year()) ?? end)")
                                .font(.caption).foregroundStyle(.secondary)
                        }
                    }
                }
            }
            NavigationLink { CalendarView() } label: {
                Label("Modifier dans le calendrier", systemImage: "calendar")
            }.font(.subheadline)
        }
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
						Text("\(periodicityLabel(sub.periodicity)) · prochain le \(sub.nextDate.opaleFormatted(.dateTime.day().month(.abbreviated)))")
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
          .sensitive()
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
		case "once": "Une fois"
		case "weekly": "Hebdomadaire"
		case "monthly": "Mensuel"
		case "quarterly": "Trimestriel"
		case "yearly": "Annuel"
		default: p.capitalized
		}
	}

	private func load() async {
		let api = session.api
		let profileKey = session.profileKey
		var failures: [String] = []
		do {
			let result = try await api.subscriptions()
			guard !Task.isCancelled, session.profileKey == profileKey else { return }
			subscriptions = result.items
			totalMonthly = result.monthly
			totalYearly = result.yearly
		} catch {
			guard !Task.isCancelled, session.profileKey == profileKey else { return }
			failures.append(error.localizedDescription)
		}
        do {
            let setup = try await api.fetchFinancialSetup()
            guard !Task.isCancelled, session.profileKey == profileKey else { return }
            let ids = Set(setup.result?.subscriptionRuleIDs ?? [])
            if ids.isEmpty {
                declaredSubscriptions = []
                declaredCurrencies = [:]
            } else {
                async let assetsRequest = api.listAssets()
                let calendar: CalendarData = try await api.request("GET", "/v1/calendar")
                let assets = try await assetsRequest
                guard !Task.isCancelled, session.profileKey == profileKey else { return }
                declaredSubscriptions = calendar.rules.filter { ids.contains($0.id) }
                declaredCurrencies = Dictionary(uniqueKeysWithValues: assets.map { ($0.id, $0.currency) })
            }
        } catch APIError.badStatus(404, _) {
            // Les serveurs antérieurs au parcours initial restent utilisables.
            declaredSubscriptions = []
            declaredCurrencies = [:]
        } catch {
            guard !Task.isCancelled, session.profileKey == profileKey else { return }
            failures.append("Abonnements déclarés : \(error.localizedDescription)")
        }
		errorMessage = failures.isEmpty ? nil : failures.joined(separator: "\n")
		loaded = true
	}
}
