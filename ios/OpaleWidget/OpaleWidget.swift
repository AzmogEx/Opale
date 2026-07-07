import WidgetKit
import SwiftUI
import Charts
import AppIntents

/// Intent INTERACTIF : flouter/défloutrer le montant directement depuis le
/// widget — un œil sur l'écran d'accueil, sans ouvrir l'app.
struct ToggleDiscreetIntent: AppIntent {
	static let title: LocalizedStringResource = "Mode discret"
	static let description = IntentDescription("Floute ou révèle le patrimoine sur le widget.")

	func perform() async throws -> some IntentResult {
		let defaults = UserDefaults(suiteName: "group.app.opale.shared")
		let current = defaults?.bool(forKey: "discreet") ?? false
		defaults?.set(!current, forKey: "discreet")
		return .result()
	}
}

// Widget « Patrimoine net » (P7) : l'essentiel d'Opale sur l'écran d'accueil.
// Lit l'instantané publié par l'app dans l'App Group — aucun réseau, aucun
// jeton : si l'app est déconnectée, le widget est vide.

/// L'instantané affiché par le widget.
struct NetWorthEntry: TimelineEntry {
    let date: Date
    let netWorthCents: Int64?
    let historyCents: [Int64]
    let discreet: Bool
}

struct NetWorthProvider: TimelineProvider {
    private static let suiteName = "group.app.opale.shared"

    func placeholder(in context: Context) -> NetWorthEntry {
        NetWorthEntry(date: .now, netWorthCents: 4_830_000,
                      historyCents: [4_500_000, 4_600_000, 4_700_000, 4_830_000],
                      discreet: false)
    }

    func getSnapshot(in context: Context, completion: @escaping (NetWorthEntry) -> Void) {
        completion(read())
    }

    func getTimeline(in context: Context, completion: @escaping (Timeline<NetWorthEntry>) -> Void) {
        // L'app pousse un reload à chaque ouverture ; on rafraîchit quand même
        // toutes les heures pour garder une date honnête.
        completion(Timeline(entries: [read()], policy: .after(.now.addingTimeInterval(3600))))
    }

    private func read() -> NetWorthEntry {
        guard let defaults = UserDefaults(suiteName: Self.suiteName),
              defaults.object(forKey: "netWorthCents") != nil else {
            return NetWorthEntry(date: .now, netWorthCents: nil, historyCents: [], discreet: false)
        }
        let history = (defaults.array(forKey: "historyCents") as? [NSNumber])?.map(\.int64Value) ?? []
        return NetWorthEntry(
            date: Date(timeIntervalSince1970: defaults.double(forKey: "updatedAt")),
            netWorthCents: defaults.object(forKey: "netWorthCents") as? Int64
                ?? Int64(defaults.integer(forKey: "netWorthCents")),
            historyCents: history,
            discreet: defaults.bool(forKey: "discreet")
        )
    }
}

struct NetWorthWidgetView: View {
    @Environment(\.widgetFamily) private var family
    var entry: NetWorthEntry

    private static let accent = Color(red: 0.35, green: 0.72, blue: 0.71)

    var body: some View {
        if let cents = entry.netWorthCents {
            content(cents)
                .containerBackground(for: .widget) {
                    LinearGradient(
                        colors: [Self.accent.opacity(0.16), .clear],
                        startPoint: .topLeading, endPoint: .bottomTrailing
                    )
                }
        } else {
            VStack(spacing: 4) {
                Image(systemName: "lock")
                    .foregroundStyle(.secondary)
                Text("Ouvre Opale")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            .containerBackground(.background, for: .widget)
        }
    }

    @ViewBuilder
    private func content(_ cents: Int64) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text("PATRIMOINE NET")
                .font(.system(size: 9, weight: .semibold))
                .foregroundStyle(.secondary)

            HStack(alignment: .center) {
                Text(Self.euros(cents))
                    .font(.system(size: family == .systemSmall ? 20 : 26,
                                  weight: .bold, design: .rounded))
                    .minimumScaleFactor(0.6)
                    .lineLimit(1)
                    .blur(radius: entry.discreet ? 9 : 0)
                Spacer(minLength: 4)
                // INTERACTIF : flouter/révéler sans ouvrir l'app (AppIntent).
                Button(intent: ToggleDiscreetIntent()) {
                    Image(systemName: entry.discreet ? "eye.slash.fill" : "eye")
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundStyle(.secondary)
                }
                .buttonStyle(.plain)
            }

            if let delta = monthDelta {
                Text(Self.signedEuros(delta))
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundStyle(delta >= 0
                        ? Color(red: 0.24, green: 0.71, blue: 0.54)
                        : Color(red: 0.88, green: 0.36, blue: 0.38))
            }

            if entry.historyCents.count >= 2 {
                sparkline
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
    }

    private var monthDelta: Int64? {
        guard entry.historyCents.count >= 2 else { return nil }
        return entry.historyCents[entry.historyCents.count - 1]
            - entry.historyCents[entry.historyCents.count - 2]
    }

    private var sparkline: some View {
        Chart(Array(entry.historyCents.enumerated()), id: \.offset) { index, value in
            LineMark(
                x: .value("Mois", index),
                y: .value("Net", Double(value) / 100)
            )
            .interpolationMethod(.monotone)
            .lineStyle(StrokeStyle(lineWidth: 2, lineCap: .round))
            .foregroundStyle(Self.accent)
        }
        .chartXAxis(.hidden)
        .chartYAxis(.hidden)
        .chartYScale(domain: .automatic(includesZero: false))
        .frame(maxHeight: family == .systemSmall ? 28 : 44)
    }

    // Formatage local au widget (pas de dépendance au code de l'app).
    private static func euros(_ cents: Int64) -> String {
        let formatter = NumberFormatter()
        formatter.numberStyle = .currency
        formatter.locale = Locale(identifier: "fr_FR")
        formatter.currencyCode = "EUR"
        formatter.maximumFractionDigits = 0
        return formatter.string(from: NSNumber(value: Double(cents) / 100)) ?? "\(cents / 100) €"
    }

    private static func signedEuros(_ cents: Int64) -> String {
        (cents >= 0 ? "+" : "") + euros(cents) + " ce mois-ci"
    }
}

struct OpaleNetWorthWidget: Widget {
    var body: some WidgetConfiguration {
        StaticConfiguration(kind: "OpaleNetWorth", provider: NetWorthProvider()) { entry in
            NetWorthWidgetView(entry: entry)
        }
        .configurationDisplayName("Patrimoine net")
        .description("Ton patrimoine net et sa tendance, en un coup d'œil.")
        .supportedFamilies([.systemSmall, .systemMedium])
    }
}

@main
struct OpaleWidgetBundle: WidgetBundle {
    var body: some Widget {
        OpaleNetWorthWidget()
    }
}
