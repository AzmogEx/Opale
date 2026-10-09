import SwiftUI

struct ContractsView: View {
    @Environment(SessionStore.self) private var session
    @State private var data: ContractsResponse?
    @State private var error: String?
    @State private var adding: FinancialContract?
    @State private var showArchived = false

    var body: some View {
        List {
            Section {
                Text("Abonnements, assurances et autres engagements : garde leurs échéances et leur prix au même endroit.")
                    .font(.subheadline).foregroundStyle(.secondary)
            }
            if let data {
                if !data.alerts.isEmpty {
                    Section("À surveiller") {
                        ForEach(data.alerts) { alert in
                            NavigationLink { ContractDetailView(contractID: alert.contractID) } label: {
                                Label { VStack(alignment: .leading, spacing: 3) {
                                    Text(alert.title).font(.subheadline.weight(.semibold))
                                    Text(alert.detail).font(.caption).foregroundStyle(.secondary)
                                } } icon: { Image(systemName: "bell.badge").foregroundStyle(.orange) }
                            }
                        }
                    }
                }
                let currencies = Set(data.contracts.filter(\.active).map(\.currency)).sorted()
                if !currencies.isEmpty {
                    Section("Coût mensuel déclaré") {
                        ForEach(currencies, id: \.self) { currency in
                            if let total = FinancialTools.monthlyContracts(data.contracts, currency: currency) {
                                LabeledContent(currency) { AmountText(cents: total, currency: currency) }
                            }
                        }
                        Text("Équivalent mensuel des contrats actifs, sans conversion de devise. Les dépenses détectées automatiquement restent dans Abonnements.")
                            .font(.caption).foregroundStyle(.secondary)
                    }
                }
                Section("Mes contrats") {
                    if data.contracts.isEmpty { ContentUnavailableView("Aucun contrat", systemImage: "doc.text", description: Text("Ajoute ton premier abonnement ou une assurance.")) }
                    ForEach(data.contracts.filter { $0.active || showArchived }) { contract in
                        NavigationLink { ContractDetailView(contractID: contract.id) } label: { ContractRow(contract: contract) }
                            .accessibilityIdentifier("contract.row.\(contract.name)")
                    }
                    if data.contracts.contains(where: { !$0.active }) { Toggle("Afficher les contrats arrêtés", isOn: $showArchived) }
                }
            } else if error == nil { ProgressView() }
            if let error { Section { Text(error).foregroundStyle(OpaleTheme.loss); Button("Réessayer") { Task { await load() } } } }
        }
        .opaleList().navigationTitle("Contrats").navigationBarTitleDisplayMode(.inline)
        .toolbar { Button("Ajouter", systemImage: "plus") { adding = FinancialContract() }.accessibilityIdentifier("contracts.add") }
        .sheet(item: $adding) { value in ContractEditSheet(initial: value) }
        .task(id: session.refreshID) { await load() }
        .refreshable { await load() }
    }
    private func load() async {
        let key = session.profileKey, api = session.api
        do {
            let response = try await api.contracts()
            guard key == session.profileKey, !Task.isCancelled else { return }
            data = response; error = nil
        } catch { guard key == session.profileKey, !Task.isCancelled else { return }; self.error = error.localizedDescription }
    }
}

private struct ContractRow: View {
    let contract: FinancialContract
    var body: some View {
        HStack(spacing: 12) {
            Image(systemName: contract.pendingPrice == nil ? "doc.text" : "arrow.up.right.circle")
                .foregroundStyle(contract.pendingPrice == nil ? OpaleTheme.accent : .orange)
            VStack(alignment: .leading, spacing: 4) {
                Text(contract.name).font(.body.weight(.medium))
                Text(contract.active ? "\(FinancialTools.frequencyLabel(contract.frequency)) · \(FinancialTools.dayLabel(contract.nextDueDate))" : "Arrêté")
                    .font(.caption).foregroundStyle(.secondary)
            }
            Spacer(minLength: 6)
            AmountText(cents: contract.amount, currency: contract.currency).font(.callout.weight(.semibold))
        }
    }
}
