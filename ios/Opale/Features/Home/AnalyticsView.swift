import SwiftUI
import Charts

/// Analyses — le cœur addictif : où part l'argent, chez qui, et mieux ou
/// moins bien que le mois dernier. Tous les chiffres viennent du moteur.
struct AnalyticsView: View {
	@Environment(SessionStore.self) private var session

	@State private var offset = 0 // décalage de mois vs aujourd'hui
	@State private var analytics: MonthAnalytics?
	@State private var selectedCategory: String?
	@State private var errorMessage: String?

	/// Couleurs des anneaux : la palette irisée déclinée.
	private static let ringPalette: [Color] = [
		OpaleTheme.accent,
		Color(red: 0.51, green: 0.62, blue: 0.87),
		Color(red: 0.78, green: 0.57, blue: 0.79),
		Color(red: 0.93, green: 0.65, blue: 0.42),
		Color(red: 0.38, green: 0.65, blue: 0.45),
		Color(red: 0.85, green: 0.53, blue: 0.60),
	]

	private var month: Date {
		let now = Date.now
		return Calendar.current.date(byAdding: .month, value: offset,
		                             to: Calendar.current.dateInterval(of: .month, for: now)!.start)!
	}

	var body: some View {
		ZStack {
			OpaleBackdrop()

			ScrollView {
				GlassEffectContainer(spacing: 16) {
					VStack(spacing: 16) {
						monthPicker
						if let analytics {
							comparisonCard(analytics)
								.cascadeIn(0)
							if let categories = analytics.categories, !categories.isEmpty {
								donutCard(categories, total: analytics.summary.expenses)
									.cascadeIn(1)
							}
							if let merchants = analytics.topMerchants, !merchants.isEmpty {
								merchantsCard(merchants, total: analytics.summary.expenses)
									.cascadeIn(2)
							}
							if (analytics.categories ?? []).isEmpty {
								EmptyStateView(
									icon: "chart.pie",
									title: "Rien à analyser",
									message: "Aucune dépense ce mois-ci — importe un relevé ou change de mois."
								)
							}
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
		.navigationTitle("Analyses")
		.navigationBarTitleDisplayMode(.inline)
		.task(id: offset) { await load() }
	}

	// MARK: - Navigation mensuelle

	private var monthPicker: some View {
		HStack {
			Button { offset -= 1 } label: { Image(systemName: "chevron.left") }
			Spacer()
			Text(month.formatted(.dateTime.month(.wide).year()).capitalized)
				.font(.headline)
				.contentTransition(.numericText())
				.animation(.snappy, value: offset)
			Spacer()
			Button { offset += 1 } label: { Image(systemName: "chevron.right") }
				.disabled(offset >= 0)
		}
		.padding(.horizontal, 6)
		.sensoryFeedback(.selection, trigger: offset)
	}

	// MARK: - Mieux / moins bien que le mois dernier

	@ViewBuilder
	private func comparisonCard(_ a: MonthAnalytics) -> some View {
		let delta = a.summary.expenses.raw - a.previous.expenses.raw
		GlassCard {
			HStack(spacing: 16) {
				VStack(alignment: .leading, spacing: 4) {
					Text("Dépensé ce mois")
						.font(.footnote.weight(.semibold))
						.foregroundStyle(.secondary)
						.textCase(.uppercase)
					AmountText(cents: a.summary.expenses, style: .whole)
						.font(.system(size: 34, weight: .bold, design: .rounded))
						.foregroundStyle(OpaleTheme.iridescent)
						.iridescentShimmer()
				}
				Spacer()
				if a.previous.expenses.raw > 0 {
					VStack(alignment: .trailing, spacing: 4) {
						Image(systemName: delta <= 0 ? "arrow.down.right.circle.fill" : "arrow.up.right.circle.fill")
							.font(.title2)
							.foregroundStyle(delta <= 0 ? OpaleTheme.gain : OpaleTheme.loss)
							.symbolEffect(.bounce, value: offset)
						AmountText(cents: Cents(delta), style: .signedDelta)
							.font(.subheadline.weight(.bold))
							.foregroundStyle(delta <= 0 ? OpaleTheme.gain : OpaleTheme.loss)
						Text("vs mois précédent")
							.font(.caption2)
							.foregroundStyle(.secondary)
					}
				}
			}
		}
	}

	// MARK: - L'anneau des catégories

	@ViewBuilder
	private func donutCard(_ categories: [CategorySpend], total: Cents) -> some View {
		GlassCard {
			VStack(alignment: .leading, spacing: 14) {
				Text("Par catégorie")
					.font(.footnote.weight(.semibold))
					.foregroundStyle(.secondary)
					.textCase(.uppercase)

				Chart(categories) { category in
					SectorMark(
						angle: .value("Montant", category.total.chartValue),
						innerRadius: .ratio(0.68),
						angularInset: 2
					)
					.cornerRadius(5)
					.foregroundStyle(color(for: category, in: categories))
					.opacity(selectedCategory == nil || selectedCategory == category.name ? 1 : 0.35)
					.accessibilityLabel(category.name)
					.accessibilityValue(MoneyFormat.eurosWhole(category.total))
				}
				.frame(height: 210)
				.chartBackground { proxy in
					GeometryReader { geo in
						if let frame = proxy.plotFrame.map({ geo[$0] }) {
							VStack(spacing: 2) {
								Text(selectedCategory ?? "Total")
									.font(.caption)
									.foregroundStyle(.secondary)
									.lineLimit(1)
								AmountText(cents: selectedAmount(categories, total: total), style: .whole)
									.font(.title3.weight(.bold))
							}
							.position(x: frame.midX, y: frame.midY)
						}
					}
				}

				// Rangées : pastille couleur, part, montant — tapables.
				ForEach(categories.prefix(6)) { category in
					Button {
						withAnimation(.snappy) {
							selectedCategory = selectedCategory == category.name ? nil : category.name
						}
					} label: {
						HStack(spacing: 10) {
							Circle()
								.fill(color(for: category, in: categories))
								.frame(width: 10, height: 10)
							Text(category.name)
								.font(.subheadline.weight(.medium))
							Spacer()
							if total.raw > 0 {
								Text("\(category.total.raw * 100 / total.raw) %")
									.font(.caption)
									.foregroundStyle(.secondary)
							}
							AmountText(cents: category.total, style: .whole)
								.font(.subheadline.weight(.semibold))
						}
						.contentShape(.rect)
					}
					.buttonStyle(.pressable)
				}
			}
		}
	}

	private func color(for category: CategorySpend, in all: [CategorySpend]) -> Color {
		let index = all.firstIndex(of: category) ?? 0
		return Self.ringPalette[index % Self.ringPalette.count]
	}

	private func selectedAmount(_ categories: [CategorySpend], total: Cents) -> Cents {
		guard let selectedCategory,
		      let category = categories.first(where: { $0.name == selectedCategory })
		else { return total }
		return category.total
	}

	// MARK: - Top marchands

	@ViewBuilder
	private func merchantsCard(_ merchants: [MerchantSpend], total: Cents) -> some View {
		GlassCard {
			VStack(alignment: .leading, spacing: 12) {
				Text("Top marchands")
					.font(.footnote.weight(.semibold))
					.foregroundStyle(.secondary)
					.textCase(.uppercase)

				let maxTotal = merchants.map(\.total.raw).max() ?? 1
				ForEach(merchants) { merchant in
					VStack(alignment: .leading, spacing: 5) {
						HStack {
							Text(merchant.label)
								.font(.subheadline.weight(.medium))
								.lineLimit(1)
							Spacer()
							Text("×\(merchant.count)")
								.font(.caption2)
								.foregroundStyle(.tertiary)
							AmountText(cents: merchant.total, style: .whole)
								.font(.subheadline.weight(.semibold))
						}
						// Barre de proportion, animée à l'apparition.
						GeometryReader { geo in
							Capsule()
								.fill(OpaleTheme.iridescent)
								.frame(width: geo.size.width
									* CGFloat(merchant.total.raw) / CGFloat(maxTotal))
						}
						.frame(height: 5)
						.background(Capsule().fill(.quaternary.opacity(0.5)))
					}
				}
			}
		}
	}

	// MARK: - Chargement

	private func load() async {
		let comps = Calendar.current.dateComponents([.year, .month], from: month)
		do {
			analytics = try await session.api.analytics(year: comps.year, month: comps.month)
			errorMessage = nil
		} catch {
			errorMessage = error.localizedDescription
		}
	}
}
