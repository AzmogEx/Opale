import SwiftUI

struct InvestmentPerformance: Codable, Hashable, Sendable {
    let initial_capital_cents: Cents
    let contributions_cents: Cents
    let withdrawals_cents: Cents
    let distributions_cents: Cents
    let fees_cents: Cents
    let gain_cents: Cents
    let return_bps: Int?
    let known: Bool
    let reason: String
}
struct InvestmentFlow: Codable, Identifiable {
    var id: String
    var asset_id: String
    var kind: String
    var amount_cents: Cents
    var occurred_on: String
    var note: String
}
struct InvestmentDetail: Decodable {
    let asset: Asset
    let flows: [InvestmentFlow]
    let performance: InvestmentPerformance
    let coverage_complete: Bool
}
struct InvestmentDetailView: View {
    let assetID: String
    @Environment(SessionStore.self) private var session
    @State private var data: InvestmentDetail?
    @State private var error: String?
    @State private var adding = false
    @State private var editing: InvestmentFlow?
    @State private var busy = false
    var body: some View {
        List {
            if let data {
                Section("Performance du moteur") {
                    LabeledContent("Capital initial") { AmountText(cents: data.performance.initial_capital_cents, currency: data.asset.currency) }
                    LabeledContent("Apports") { AmountText(cents: data.performance.contributions_cents, currency: data.asset.currency) }
                    LabeledContent("Retraits") { AmountText(cents: data.performance.withdrawals_cents, currency: data.asset.currency) }
                    LabeledContent("Distributions") { AmountText(cents: data.performance.distributions_cents, currency: data.asset.currency) }
                    LabeledContent("Frais externes") { AmountText(cents: data.performance.fees_cents, currency: data.asset.currency) }
                    if data.performance.known {
                        LabeledContent("Gain / perte") { AmountText(cents: data.performance.gain_cents, currency: data.asset.currency) }
                        if let rate = data.performance.return_bps { LabeledContent("Rendement cumulé", value: "\(Decimal(rate) / 100) %") }
                    }
                    Text(data.performance.reason).font(.caption)
                }
                Section("Qualité des données") {
                    Toggle("Historique des flux complet", isOn: Binding(get: { data.coverage_complete }, set: { wanted in Task { await setCoverage(wanted) } })).disabled(busy)
                    Text("Ne confirme qu'après avoir saisi tous les flux postérieurs à la première valorisation. Les frais déjà déduits de la valeur du placement ne doivent pas être saisis une seconde fois.").font(.caption)
                }
                Section("Flux du placement") {
                    Button("Ajouter un flux") { adding = true }
                    ForEach(data.flows) { flow in
                        Button { editing = flow } label: {
                            VStack(alignment: .leading) {
                                LabeledContent(Self.label(flow.kind)) { AmountText(cents: flow.amount_cents, currency: data.asset.currency) }
                                Text(flow.occurred_on + (flow.note.isEmpty ? "" : " · " + flow.note)).font(.caption)
                            }
                        }.buttonStyle(.plain)
                    }.onDelete { indices in Task { do { for i in indices { let _: APIClient.EmptyResponse = try await session.api.request("DELETE", "/v1/assets/\(assetID)/investment/flows/\(data.flows[i].id)") }; await load() } catch { self.error = error.localizedDescription } } }
                }
            } else if error == nil { ProgressView() }
            ToolError(message: error)
        }.navigationTitle(data?.asset.name ?? "Placement").task { await load() }
        .sheet(isPresented: $adding) { InvestmentFlowSheet(assetID: assetID, currency: data?.asset.currency ?? "EUR") { Task { await load() } } }
        .sheet(item: $editing) { flow in InvestmentFlowSheet(assetID: assetID, currency: data?.asset.currency ?? "EUR", existing: flow) { Task { await load() } } }
    }
    static func label(_ kind: String) -> String { switch kind { case "contribution": "Apport"; case "withdrawal": "Retrait"; case "distribution": "Distribution"; default: "Frais externe" } }
    private func load() async { do { data = try await session.api.request("GET", "/v1/assets/\(assetID)/investment"); error = nil } catch { self.error = error.localizedDescription } }
    private func setCoverage(_ complete: Bool) async { struct Body: Encodable { let complete: Bool }; busy = true; defer { busy = false }; do { let _: APIClient.EmptyResponse = try await session.api.request("PUT", "/v1/assets/\(assetID)/investment/coverage", body: Body(complete: complete)); await load() } catch { self.error = error.localizedDescription } }
}
private struct InvestmentFlowSheet: View {
    let assetID: String
    let currency: String
    var existing: InvestmentFlow?
    var saved: () -> Void
    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss
    @State private var kind = "contribution"
    @State private var amount = ""
    @State private var date = Date.now
    @State private var note = ""
    @State private var error: String?
    @State private var busy = false
    var body: some View {
        NavigationStack { Form {
            Section {
                Picker("Nature", selection: $kind) { ForEach(["contribution", "withdrawal", "distribution", "fee"], id: \.self) { Text(InvestmentDetailView.label($0)).tag($0) } }
                TextField("Montant en \(currency)", text: $amount).keyboardType(.decimalPad)
                DatePicker("Date", selection: $date, in: ...Date.now, displayedComponents: .date)
                TextField("Note", text: $note)
            }
            ToolError(message: error)
        }.navigationTitle("Flux du placement").toolbar {
            ToolbarItem(placement: .cancellationAction) { Button("Annuler") { dismiss() } }
            ToolbarItem(placement: .confirmationAction) { Button("Enregistrer") { Task { await save() } }.disabled(busy || (Cents.parse(amount, currency: currency)?.raw ?? 0) <= 0) }
        }.onAppear { if let existing { kind = existing.kind; amount = MoneyFormat.input(existing.amount_cents, currency: currency); date = Date.fromOpaleDay(existing.occurred_on) ?? .now; note = existing.note } } }
    }
    private func save() async {
        guard let value = Cents.parse(amount, currency: currency), value.raw > 0 else { return }
        busy = true; defer { busy = false }
        let body = InvestmentFlow(id: existing?.id ?? "", asset_id: assetID, kind: kind, amount_cents: value, occurred_on: date.opaleDayString, note: note)
        do { let _: APIClient.EmptyResponse = try await session.api.request(existing == nil ? "POST" : "PATCH", "/v1/assets/\(assetID)/investment/flows" + (existing.map { "/\($0.id)" } ?? ""), body: body); session.changed(); saved(); dismiss() } catch { self.error = error.localizedDescription }
    }
}
