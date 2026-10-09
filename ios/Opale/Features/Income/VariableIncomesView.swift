import SwiftUI

struct VariableIncomesView: View {
    @Environment(SessionStore.self) private var session
    @State private var incomes: [VariableIncome] = []
    @State private var loaded = false
    @State private var error: String?
    @State private var editing: VariableIncome?
    @State private var deleting: VariableIncome?
    @State private var busy = false

    var body: some View {
        List {
            Section {
                Text("Primes, missions, loyers et dividendes : sépare ce qui est espéré de ce qui est déjà reçu.").font(.subheadline).foregroundStyle(.secondary)
                NavigationLink("Voir les encaissements prévus") { CalendarView() }
            }
            let currencies = Set(incomes.filter { $0.active && $0.frequency != "once" }.map(\.currency)).sorted()
            ForEach(currencies, id: \.self) { currency in
                Section("Équivalent mensuel · \(currency)") {
                    incomeTotal("Minimum déclaré", currency: currency, keyPath: \.low)
                    incomeTotal("Habituel", currency: currency, keyPath: \.usual)
                    incomeTotal("Maximum déclaré", currency: currency, keyPath: \.high)
                    Text("Montants nets déclarés, sans garantie de versement. Les revenus ponctuels sont exclus de ces moyennes.").font(.caption).foregroundStyle(.secondary)
                }
            }
            Section("Sources de revenus") {
                if loaded && incomes.isEmpty { ContentUnavailableView("Aucun revenu variable", systemImage: "waveform.path", description: Text("Ajoute une prime ou une activité avec une fourchette prudente.")) }
                ForEach(incomes) { income in
                    Button { editing = income } label: { incomeRow(income) }.buttonStyle(.plain).accessibilityIdentifier("income.row.\(income.name)")
                        .swipeActions { Button("Supprimer", role: .destructive) { deleting = income } }
                }
            }
            if !loaded && error == nil { ProgressView() }
            if let error { Section { Text(error).foregroundStyle(OpaleTheme.loss); Button("Recharger") { Task { await load() } } } }
        }.opaleList().navigationTitle("Revenus variables").navigationBarTitleDisplayMode(.inline)
        .toolbar { Button("Ajouter", systemImage: "plus") { editing = VariableIncome() }.disabled(busy).accessibilityIdentifier("incomes.add") }
        .sheet(item: $editing) { value in VariableIncomeEditSheet(initial: value) }
        .task(id: session.refreshID) { await load() }.refreshable { await load() }
        .confirmationDialog("Supprimer ce revenu et ses prochaines prévisions ?", isPresented: Binding(get: { deleting != nil }, set: { if !$0 { deleting = nil } }), titleVisibility: .visible) {
            if let deleting { Button("Supprimer \(deleting.name)", role: .destructive) { Task { await remove(deleting) } } }
        }
    }
    private func incomeTotal(_ label: String, currency: String, keyPath: KeyPath<VariableIncome, Cents>) -> some View {
        LabeledContent(label) { if let value = FinancialTools.monthlyIncome(incomes, currency: currency, keyPath: keyPath) { AmountText(cents: value, currency: currency) } else { Text("Montant hors limite") } }
    }
    private func incomeRow(_ income: VariableIncome) -> some View {
        VStack(alignment: .leading, spacing: 5) {
            HStack { Text(income.name).font(.body.weight(.medium)); Spacer(); AmountText(cents: income.usual, currency: income.currency) }
            Text("\(FinancialTools.frequencyLabel(income.frequency)) · \(FinancialTools.dayLabel(income.nextDate))" + (income.active ? "" : " · arrêté")).font(.caption).foregroundStyle(.secondary)
            HStack(spacing: 4) {
                Text("De"); AmountText(cents: income.low, currency: income.currency); Text("à"); AmountText(cents: income.high, currency: income.currency)
            }.font(.caption)
            Text(income.calendarRuleID.isEmpty ? "Hors prévision" : income.forecast == "prudent" ? "Prévision au minimum déclaré" : "Prévision au montant habituel")
                .font(.caption).foregroundStyle(OpaleTheme.accent)
        }
    }
    private func load() async {
        let key = session.profileKey, api = session.api
        do { let values = try await api.variableIncomes(); guard key == session.profileKey, !Task.isCancelled else { return }; incomes = values; loaded = true; error = nil }
        catch { guard key == session.profileKey, !Task.isCancelled else { return }; self.error = error.localizedDescription }
    }
    private func remove(_ income: VariableIncome) async {
        let key = session.profileKey, api = session.api; busy = true; defer { busy = false; deleting = nil }
        do { try await api.deleteVariableIncome(income); guard key == session.profileKey else { return }; session.changed() }
        catch { guard key == session.profileKey else { return }; self.error = error.localizedDescription }
    }
}
