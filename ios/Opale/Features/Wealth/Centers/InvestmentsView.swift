import SwiftUI
import Charts

/// Centre investissement (EF-034) : répartition du portefeuille et
/// performance de chaque placement depuis sa première valorisation.
struct InvestmentsView: View {
    @Environment(SessionStore.self) private var session

    @State private var investments: [InvestmentStatus] = []
    @State private var total: Cents = .zero
    @State private var loaded = false

    var body: some View {
        List {
            if loaded && investments.isEmpty {
                ContentUnavailableView(
                    "Aucun placement",
                    systemImage: "chart.pie",
                    description: Text("Ajoute un PEA, un compte-titres, une assurance-vie ou de la crypto dans Patrimoine.")
                )
            }

            if !investments.isEmpty {
                Section {
                    allocationChart
                        .listRowBackground(Color.clear)
                } header: {
                    HStack {
                        Text("Répartition")
                        Spacer()
                        AmountText(cents: total, style: .whole)
                            .font(.subheadline.weight(.bold))
                            .textCase(nil)
                    }
                }

                Section("Performance") {
                    ForEach(investments) { inv in
                        row(inv)
                    }
                }
            }
        }
        .navigationTitle("Placements")
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
        .refreshable { await load() }
    }

    // Donut de répartition (SectorMark, valeurs strictement positives).
    private var allocationChart: some View {
        Chart(investments.filter { ($0.asset.latestValue?.raw ?? 0) > 0 }) { inv in
            SectorMark(
                angle: .value("Valeur", inv.asset.latestValue?.chartValue ?? 0),
                innerRadius: .ratio(0.62),
                angularInset: 1.5
            )
            .cornerRadius(4)
            .foregroundStyle(by: .value("Placement", inv.asset.name))
            .accessibilityLabel(inv.asset.name)
            .accessibilityValue(MoneyFormat.eurosWhole(inv.asset.latestValue ?? .zero))
        }
        .chartLegend(position: .bottom, spacing: 8)
        .frame(minHeight: 220)
        .padding(.vertical, 4)
    }

    @ViewBuilder
    private func row(_ inv: InvestmentStatus) -> some View {
        HStack {
            Image(systemName: inv.asset.kind.systemImage)
                .font(.title3)
                .foregroundStyle(OpaleTheme.accent)
                .frame(width: 32)
            VStack(alignment: .leading, spacing: 2) {
                Text(inv.asset.name).font(.body.weight(.medium))
                HStack(spacing: 4) {
                    Text(inv.asset.kind.label)
                    if inv.allocationBps > 0 {
                        Text("· \(inv.allocationBps / 100) %")
                    }
                }
                .font(.caption)
                .foregroundStyle(.secondary)
            }
            Spacer()
            VStack(alignment: .trailing, spacing: 2) {
                AmountText(cents: inv.asset.latestValue ?? .zero, style: .whole)
                    .font(.callout.weight(.semibold))
                if inv.firstValue.raw > 0, inv.change.raw != 0 {
                    Text("\(MoneyFormat.signedEurosWhole(inv.change)) (\(percentLabel(inv.changeBps)))")
                        .font(.caption)
                        .foregroundStyle(inv.change.raw < 0 ? OpaleTheme.loss : OpaleTheme.gain)
                }
            }
        }
    }

    private func percentLabel(_ bps: Int) -> String {
        let sign = bps > 0 ? "+" : ""
        return "\(sign)\(bps / 100),\(abs(bps % 100) / 10) %"
    }

    private func load() async {
        if let result = try? await session.api.investments() {
            investments = result.items
            total = result.total
        }
        loaded = true
    }
}
