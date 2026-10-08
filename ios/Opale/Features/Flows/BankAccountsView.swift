import SwiftUI

struct BankAccount: Decodable, Identifiable {
    let id: String
    let link_id: String
    let provider_account_id: String
    let asset_id: String?
    let currency: String
    let name: String
    let status: String
    let last_synced_at: Date?
    let balance_cents: Cents?
    let balance_date: String?
    let balance_type: String
    let balance_reconciled: Bool
    let last_error: String
}

struct BankAccountsView: View {
    @Environment(SessionStore.self) private var session
    struct Pending: Decodable, Identifiable { let account_id: String; let amount_cents: Cents; let currency: String; let occurred_on: String?; let label: String; var id: String { account_id + label + (occurred_on ?? "") + String(amount_cents.raw) } }
    @State private var pending: [Pending] = []
    @State private var accounts: [BankAccount] = []
    @State private var assets: [Asset] = []
    @State private var error: String?
    @State private var busy = false
    var body: some View {
        List {
            Section { Text("Après le consentement, synchronise pour découvrir les comptes. Associe chaque compte bancaire à un compte Opale distinct de même devise, puis synchronise de nouveau.").font(.caption) }
            if accounts.isEmpty { ContentUnavailableView("Aucun compte découvert", systemImage: "building.columns", description: Text("Lance une synchronisation après le consentement bancaire.")) }
            ForEach(accounts) { account in
                Section(account.name) {
                    Text("Identifiant fournisseur : \(account.provider_account_id)").font(.caption).textSelection(.enabled)
                    LabeledContent("Devise", value: account.currency.isEmpty ? "En attente" : account.currency)
                    Picker("Compte Opale", selection: Binding(get: { account.asset_id ?? "" }, set: { id in Task { await map(account.id, to: id) } })) {
                        Text("Non associé").tag("")
                        ForEach(assets.filter { !$0.archived && ($0.kind == .checking || $0.kind == .savings) && (account.currency.isEmpty || $0.currency == account.currency) }) { Text($0.name).tag($0.id) }
                    }.disabled(busy)
                    if let balance = account.balance_cents {
                        LabeledContent("Solde fournisseur") { AmountText(cents: balance, currency: account.currency.isEmpty ? "EUR" : account.currency) }
                        Text("Au \(account.balance_date ?? "—") · \(account.balance_type)").font(.caption)
                        Label(account.balance_reconciled ? "Solde rapproché" : "Solde à rapprocher", systemImage: account.balance_reconciled ? "checkmark.circle" : "exclamationmark.circle")
                    }
                    if let date = account.last_synced_at { Text("Dernière synchronisation : \(date.formatted(date: .abbreviated, time: .shortened))").font(.caption) }
                    if !account.last_error.isEmpty { Text(account.last_error).foregroundStyle(.red) }
                }
            }
            if !pending.isEmpty { Section("Mouvements provisoires") {
                Text("En attente de comptabilisation par la banque. Exclus des dépenses et du solde calculé.").font(.caption)
                ForEach(pending) { item in LabeledContent(item.label) { AmountText(cents: item.amount_cents, currency: item.currency) } }
            } }
            ToolError(message: error)
        }.navigationTitle("Comptes bancaires").task { await load() }.refreshable { await load() }
    }
    private func load() async {
        do {
            struct Envelope: Decodable { let accounts: [BankAccount]; let pending: [Pending]? }
            let response: Envelope = try await session.api.request("GET", "/v1/bank/accounts")
            accounts = response.accounts
            pending = response.pending ?? []
            assets = try await session.api.listAssets()
            error = nil
        } catch { self.error = error.localizedDescription }
    }
    private func map(_ id: String, to asset: String) async {
        busy = true; defer { busy = false }
        do {
            struct Body: Encodable { let asset_id: String }
            let _: APIClient.EmptyResponse = try await session.api.request("PUT", "/v1/bank/accounts/\(id)", body: Body(asset_id: asset))
            await load()
        } catch { self.error = error.localizedDescription }
    }
}
