import SwiftUI

struct TransferSheet: View {
    var saved: () -> Void
    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss
    @State private var accounts: [Asset] = []
    @State private var source = ""
    @State private var destination = ""
    @State private var debit = ""
    @State private var credit = ""
    @State private var fees = ""
    @State private var date = Date.now
    @State private var requestID = UUID().uuidString
    @State private var error: String?
    @State private var busy = false
    private var sourceCurrency: String { accounts.first { $0.id == source }?.currency ?? "EUR" }
    private var destinationCurrency: String { accounts.first { $0.id == destination }?.currency ?? "EUR" }
    var body: some View {
        NavigationStack { Form {
            Section("Entre mes comptes") {
                Picker("Depuis", selection: $source) { Text("Choisir").tag(""); ForEach(accounts) { Text($0.name + " · " + $0.currency).tag($0.id) } }
                Picker("Vers", selection: $destination) { Text("Choisir").tag(""); ForEach(accounts.filter { $0.id != source }) { Text($0.name + " · " + $0.currency).tag($0.id) } }
                TextField("Montant débité hors frais (\(sourceCurrency))", text: $debit).keyboardType(.decimalPad)
                TextField("Montant reçu (\(destinationCurrency))", text: $credit).keyboardType(.decimalPad)
                TextField("Frais débités (\(sourceCurrency))", text: $fees).keyboardType(.decimalPad)
                DatePicker("Date", selection: $date, in: ...Date.now, displayedComponents: .date)
            }
            Section { Text("Les deux mouvements sont enregistrés ensemble et exclus des revenus et dépenses. Seuls les frais constituent une dépense. En devises différentes, saisis les montants réellement débités et reçus.").font(.caption) }
            ToolError(message: error)
        }.navigationTitle("Virement interne").toolbar {
            ToolbarItem(placement: .cancellationAction) { Button("Annuler") { dismiss() } }
            ToolbarItem(placement: .confirmationAction) { Button("Enregistrer") { Task { await save() } }.disabled(busy || source.isEmpty || destination.isEmpty || source == destination) }
        }.task { do { accounts = try await session.api.listAssets().filter { !$0.archived && ($0.kind == .checking || $0.kind == .savings) } } catch { self.error = error.localizedDescription } } }
    }
    private func save() async {
        guard let from = Cents.parse(debit, currency: sourceCurrency), let to = Cents.parse(credit, currency: destinationCurrency), let fee = Cents.parse(fees.isEmpty ? "0" : fees, currency: sourceCurrency), from.raw > 0, to.raw > 0, fee.raw >= 0 else { error = "Montants positifs valides requis."; return }
        struct Body: Encodable { let request_id, from_asset_id, to_asset_id: String; let from_amount_cents, to_amount_cents, fee_cents: Int64; let occurred_on: String }
        busy = true; defer { busy = false }
        do { let _: APIClient.EmptyResponse = try await session.api.request("POST", "/v1/transfers", body: Body(request_id: requestID, from_asset_id: source, to_asset_id: destination, from_amount_cents: from.raw, to_amount_cents: to.raw, fee_cents: fee.raw, occurred_on: date.opaleDayString)); session.changed(); saved(); dismiss() } catch { self.error = error.localizedDescription }
    }
}
