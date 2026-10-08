import SwiftUI
import Charts

/// Centre investissement (EF-034) : répartition du portefeuille et
/// performance de chaque placement depuis sa première valorisation.
struct InvestmentsView: View {
    @Environment(SessionStore.self) private var session

    @State private var investments: [InvestmentStatus] = []
    @State private var total: Cents = .zero
    @State private var loaded = false
    @State private var errorMessage: String?

    var body: some View {
        List {
            if let errorMessage { Text(errorMessage).foregroundStyle(.red) }
            if loaded && investments.isEmpty {
                ContentUnavailableView(
                    "Aucun placement",
                    systemImage: "chart.pie",
                    description: Text("Ajoute un PEA, un compte-titres, une assurance-vie ou de la crypto dans Patrimoine.")
                )
            }

            if !investments.isEmpty {
                Section {
                    allocationChart.sensitive()
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
                        NavigationLink { InvestmentDetailView(assetID: inv.asset.id) } label: { row(inv) }
                    }
                }
            }
        }
        .opaleList()
        .navigationTitle("Placements")
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
        .refreshable { await load() }
    }

    // Donut de répartition (SectorMark, valeurs strictement positives).
    private var allocationChart: some View {
        Chart(investments.filter { ($0.valueEUR?.raw ?? 0) > 0 }) { inv in
            SectorMark(
                angle: .value("Valeur", inv.valueEUR?.chartValue ?? 0),
                innerRadius: .ratio(0.62),
                angularInset: 1.5
            )
            .cornerRadius(4)
            .foregroundStyle(by: .value("Placement", inv.asset.name))
            .accessibilityLabel(inv.asset.name)
            .accessibilityValue(MoneyFormat.eurosWhole(inv.valueEUR ?? .zero))
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
                Group { if let value = inv.asset.currentValue ?? inv.asset.latestValue { AmountText(cents: value, style: .whole, currency: inv.asset.currency) } else { Text("Valeur non renseignée") } }
                    .font(.callout.weight(.semibold))
                if let performance = inv.performance, performance.known {
                    AmountText(cents: performance.gain_cents, style: .signedDelta, currency: inv.asset.currency).font(.caption)
                } else { Text("Performance indéterminée").font(.caption2).foregroundStyle(.secondary) }

            }
        }
    }

    private func percentLabel(_ bps: Int) -> String {
        let sign = bps > 0 ? "+" : ""
        return "\(sign)\(bps / 100),\(abs(bps % 100) / 10) %"
    }

    private func load() async {
        do { let result = try await session.api.investments()
            investments = result.items
            total = result.total
            errorMessage = nil
        } catch { errorMessage = error.localizedDescription }
        loaded = true
    }
}
