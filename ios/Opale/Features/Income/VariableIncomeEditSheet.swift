import SwiftUI

struct VariableIncomeEditSheet: View {
    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss
    @State private var value: VariableIncome
    @State private var low: String
    @State private var usual: String
    @State private var high: String
    @State private var accounts: [Asset] = []
    @State private var error: String?
    @State private var busy = false
    @State private var showImport = false
    @State private var importingCurrency = false

    init(initial: VariableIncome) {
        _value = State(initialValue: initial)
        _low = State(initialValue: MoneyFormat.input(initial.low, currency: initial.currency))
        _usual = State(initialValue: initial.usual.raw == 0 ? "" : MoneyFormat.input(initial.usual, currency: initial.currency))
        _high = State(initialValue: initial.high.raw == 0 ? "" : MoneyFormat.input(initial.high, currency: initial.currency))
    }
    private var valid: Bool {
        guard !value.name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty, value.name.count <= 200, value.note.count <= 2000,
              let min = FinancialTools.amount(low, currency: value.currency, zeroAllowed: true),
              let usual = FinancialTools.amount(usual, currency: value.currency), let max = FinancialTools.amount(high, currency: value.currency),
              min <= usual, usual <= max else { return false }
        return value.forecast == "off" || !value.assetID.isEmpty
    }
    var body: some View {
        NavigationStack {
            Form {
                Section {
                    Button("Préremplir depuis une fiche de paie", systemImage: "doc.viewfinder") { showImport = true }.accessibilityIdentifier("income.import")
                    Text("Lecture locale du net payé. Tu confirmeras le montant et la fourchette avant l’enregistrement.").font(.caption).foregroundStyle(.secondary)
                }
                Section("Source") {
                    TextField("Nom du revenu", text: $value.name).accessibilityIdentifier("income.name")
                    Picker("Type", selection: $value.kind) {
                        Text("Activité indépendante").tag("freelance"); Text("Prime / commission").tag("bonus"); Text("Loyer").tag("rental")
                        Text("Dividende").tag("dividend"); Text("Autre").tag("other")
                    }
                    Picker("Devise", selection: $value.currency) { ForEach(FinancialTools.currencies, id: \.self) { Text($0).tag($0) } }.disabled(!value.assetID.isEmpty)
                    FinancialFrequencyPicker(value: $value.frequency, allowOnce: true)
                    DatePicker("Prochain versement attendu", selection: FinancialDate.binding($value.nextDate), in: FinancialDate.futureRange, displayedComponents: .date)
                }
                Section("Fourchette nette par versement · \(value.currency)") {
                    TextField("Minimum prudent, peut être 0", text: $low).keyboardType(.decimalPad).accessibilityIdentifier("income.low")
                    TextField("Montant habituel", text: $usual).keyboardType(.decimalPad).accessibilityIdentifier("income.usual")
                    TextField("Maximum estimé", text: $high).keyboardType(.decimalPad).accessibilityIdentifier("income.high")
                    Text("Après charges et impôts à prévoir. Le minimum doit être inférieur ou égal à l’habituel, puis au maximum. Une prime incertaine peut avoir un minimum de 0.").font(.caption).foregroundStyle(.secondary)
                }
                Section("Prévisions") {
                    Picker("Compte crédité", selection: $value.assetID) {
                        Text("Aucun").tag("")
                        if !value.assetID.isEmpty, !accounts.contains(where: { $0.id == value.assetID }) { Text("Compte indisponible — à changer").tag(value.assetID) }
                        ForEach(accounts) { Text("\($0.name) · \($0.currency)").tag($0.id) }
                    }
                    Picker("Montant prévu au calendrier", selection: $value.forecast) {
                        Text("Ne rien prévoir").tag("off"); Text("Minimum prudent").tag("prudent"); Text("Montant habituel").tag("usual")
                    }.accessibilityIdentifier("income.forecast")
                    Text("Avec un minimum de 0, le mode prudent n’ajoute aucun encaissement. Le maximum n’est jamais utilisé comme prévision. Quand l’argent arrive, relie sa transaction dans le calendrier.")
                        .font(.caption).foregroundStyle(.secondary)
                    TextField("Libellé payeur sur le relevé, facultatif", text: $value.merchantKey).autocorrectionDisabled()
                    Text("Ce libellé permet de rapprocher les récurrences détectées sur ce compte. Sans libellé, vérifie les doublons dans le calendrier.").font(.caption).foregroundStyle(.secondary)
                    Toggle("Source active", isOn: $value.active)
                }
                Section("Notes") { TextField("Hypothèses, saisonnalité…", text: $value.note, axis: .vertical).lineLimit(3...6) }
                if let error { Section { Text(error).foregroundStyle(OpaleTheme.loss) } }
            }.opaleList().scrollDismissesKeyboard(.interactively).disabled(busy)
            .navigationTitle(value.revision == 0 ? "Nouveau revenu" : "Modifier le revenu").navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Annuler") { dismiss() }.disabled(busy) }
                ToolbarItem(placement: .confirmationAction) { Button("Enregistrer") { Task { await save() } }.disabled(!valid || busy).accessibilityIdentifier("income.save") }
            }
            .task { await loadAccounts() }
            .onChange(of: value.assetID) { _, id in
                if let a = accounts.first(where: { $0.id == id }), a.currency != value.currency { value.currency = a.currency }
                if id.isEmpty { value.forecast = "off"; value.merchantKey = "" }
            }
            .onChange(of: value.currency) { _, _ in if importingCurrency { importingCurrency = false; return }; low = "0"; usual = ""; high = ""; error = "Devise changée : ressaisis la fourchette dans cette devise." }
            .sheet(isPresented: $showImport) { DocumentPrefillSheet(kind: .payslip, expectedCurrency: nil) { result in
                if value.currency != result.currency { value.assetID = ""; value.forecast = "off"; low = "0" }
                if value.currency != result.currency { importingCurrency = true }
                value.currency = result.currency
                if !result.name.isEmpty { value.name = result.name }
                usual = MoneyFormat.input(result.amount, currency: result.currency)
                high = usual
                if let minimum = FinancialTools.amount(low, currency: result.currency, zeroAllowed: true), minimum > result.amount { low = "0" }
                error = nil
            } }
            .interactiveDismissDisabled(busy)
        }
    }
    private func loadAccounts() async {
        let key = session.profileKey, api = session.api
        do { let assets = try await api.listAssets(); guard key == session.profileKey, !Task.isCancelled else { return }; accounts = assets.filter { !$0.archived && ($0.kind == .checking || $0.kind == .savings) } }
        catch { guard key == session.profileKey, !Task.isCancelled else { return }; self.error = error.localizedDescription }
    }
    private func save() async {
        guard valid, let min = FinancialTools.amount(low, currency: value.currency, zeroAllowed: true), let mid = FinancialTools.amount(usual, currency: value.currency), let max = FinancialTools.amount(high, currency: value.currency) else { return }
        value.low = min; value.usual = mid; value.high = max; busy = true; defer { busy = false }
        let key = session.profileKey, api = session.api
        do { let _ = try await api.saveVariableIncome(value); guard key == session.profileKey else { return }; session.changed(); dismiss() }
        catch { guard key == session.profileKey else { return }; self.error = error.localizedDescription }
    }
}
