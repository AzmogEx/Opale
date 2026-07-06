import SwiftUI

/// Timeline patrimoniale (EF-045) : la chronologie de la vie financière —
/// acquisitions et paliers franchis, puis objectifs et liberté à venir.
struct TimelineView: View {
    @Environment(SessionStore.self) private var session

    @State private var events: [TimelineEvent] = []
    @State private var loaded = false

    private var past: [TimelineEvent] { events.filter { !$0.future } }
    private var future: [TimelineEvent] { events.filter(\.future) }

    var body: some View {
        List {
            if loaded && events.isEmpty {
                ContentUnavailableView(
                    "Timeline vide",
                    systemImage: "calendar.day.timeline.left",
                    description: Text("Les acquisitions, paliers et objectifs apparaîtront ici au fil du temps.")
                )
            }
            if !past.isEmpty {
                Section("Parcours") {
                    ForEach(past) { event in
                        row(event)
                    }
                }
            }
            if !future.isEmpty {
                Section("À venir") {
                    ForEach(future) { event in
                        row(event)
                    }
                }
            }
        }
        .opaleList()
        .navigationTitle("Timeline")
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
        .refreshable { await load() }
    }

    @ViewBuilder
    private func row(_ e: TimelineEvent) -> some View {
        HStack(alignment: .top, spacing: 12) {
            VStack(spacing: 0) {
                Image(systemName: icon(e.kind))
                    .font(.body)
                    .foregroundStyle(color(e.kind))
                    .frame(width: 28, height: 28)
                    .background(color(e.kind).opacity(0.12), in: .circle)
            }
            VStack(alignment: .leading, spacing: 2) {
                Text(e.title)
                    .font(.subheadline.weight(.semibold))
                HStack(spacing: 6) {
                    Text(formatted(e.date))
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    if let amount = e.amount, amount.raw != 0 {
                        AmountText(cents: amount, style: .whole)
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(color(e.kind))
                    }
                }
                if let detail = e.detail, !detail.isEmpty {
                    Text(detail)
                        .font(.caption)
                        .foregroundStyle(.tertiary)
                }
            }
        }
        .padding(.vertical, 2)
    }

    private func icon(_ kind: String) -> String {
        switch kind {
        case "acquisition": "plus.circle.fill"
        case "milestone": "flag.checkered"
        case "goal": "target"
        case "independence": "bird.fill"
        default: "circle.fill"
        }
    }

    private func color(_ kind: String) -> Color {
        switch kind {
        case "milestone": OpaleTheme.gain
        case "goal": .orange
        case "independence": OpaleTheme.accent
        default: .secondary
        }
    }

    private func formatted(_ day: String) -> String {
        guard let date = Date.fromOpaleDay(day) else { return day }
        return date.formatted(.dateTime.day().month(.wide).year())
    }

    private func load() async {
        events = (try? await session.api.timeline()) ?? []
        loaded = true
    }
}
