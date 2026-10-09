import SwiftUI

/// A fixed income uses the existing revisioned income record; it never creates a received transaction.
struct FixedIncomeSheet: View {
    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss
    @State private var value: VariableIncome
    @State private var amount: String
    @State private var accounts: [Asset] = []
    @State private var showImport = false
    @State private var busy = false
    @State private var error: String?

    init(initial: VariableIncome? = nil) {
        var record = initial ?? VariableIncome()
        if initial == nil { record.name = "Salaire net"; record.kind = "other"; record.forecast = "usual" }
        _value = State(initialValue: record)
        _amount = State(initialValue: record.usual == .zero ? "" : MoneyFormat.input(record.usual, currency: record.currency))
    }
    private var valid: Bool {
        !value.name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty && value.name.count <= 200 &&
        FinancialTools.amount(amount, currency: value.currency) != nil && !value.assetID.isEmpty &&
        accounts.contains { $0.id == value.assetID && $0.currency == value.currency }
    }
    var body: some View {
        NavigationStack {
            Form {
                Section {
                    Text("Recopie le net payé après impôt : c’est la somme qui arrive sur ton compte.")
                    Button("Lire une fiche de paie", systemImage: "doc.viewfinder") { showImport = true }
                    Text("Lecture sur l’iPhone, puis vérification. La fiche n’est pas envoyée à l’IA distante.").font(.caption).foregroundStyle(.secondary)
                }
                Section("Ton revenu") {
                    TextField("Nom du revenu", text: $value.name).accessibilityIdentifier("fixedIncome.name")
                    TextField("Net versé par échéance (\(value.currency))", text: $amount).keyboardType(.decimalPad).accessibilityIdentifier("fixedIncome.amount")
                    Picker("Compte qui reçoit l’argent", selection: $value.assetID) {
                        Text("Choisir un compte").tag("")
                        ForEach(accounts) { Text("\($0.name) · \($0.currency)").tag($0.id) }
                    }
                    FinancialFrequencyPicker(value: $value.frequency)
                    DatePicker("Prochain versement", selection: FinancialDate.binding($value.nextDate), in: FinancialDate.futureRange, displayedComponents: .date)
                    if accounts.isEmpty { Text("Ajoute d’abord un compte à la première étape du parcours.").foregroundStyle(.orange) }
                }
                Section {
                    Text("Ce montant entre dans ton mois type et les prévisions du calendrier. Il ne change pas le solde comme si le salaire avait déjà été reçu.").font(.caption).foregroundStyle(.secondary)
                }
                if let error { Section { Text(error).foregroundStyle(OpaleTheme.loss) } }
            }.opaleList().scrollDismissesKeyboard(.interactively).disabled(busy)
            .navigationTitle(value.revision == 0 ? "Salaire ou revenu fixe" : "Modifier le revenu fixe").navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Annuler") { dismiss() }.disabled(busy) }
                ToolbarItem(placement: .confirmationAction) { Button("Enregistrer") { Task { await save() } }.disabled(busy || !valid).accessibilityIdentifier("fixedIncome.save") }
            }
            .task { await load() }
            .onChange(of: value.assetID) { _, id in
                if let account = accounts.first(where: { $0.id == id }), account.currency != value.currency {
                    value.currency = account.currency
                    amount = ""; error = "Devise changée : recopie le montant dans la devise du compte."
                }
            }
            .sheet(isPresented: $showImport) {
                DocumentPrefillSheet(kind: .payslip, expectedCurrency: nil) { result in
                    if result.currency != value.currency {
                        value.assetID = ""; value.currency = result.currency
                        if let account = accounts.first(where: { $0.currency == result.currency }) { value.assetID = account.id }
                    }
                    amount = MoneyFormat.input(result.amount, currency: result.currency)
                    error = nil
                }
            }
            .interactiveDismissDisabled(busy)
        }
    }
    private func load() async {
        let key = session.profileKey
        do {
            let result = try await session.api.listAssets()
            guard key == session.profileKey, !Task.isCancelled else { return }
            accounts = result.filter { !$0.archived && [.checking, .savings].contains($0.kind) }
            if value.assetID.isEmpty, let account = accounts.first(where: { $0.currency == value.currency }) { value.assetID = account.id }
        } catch { guard key == session.profileKey else { return }; self.error = error.localizedDescription }
    }
    private func save() async {
        guard valid, let parsed = FinancialTools.amount(amount, currency: value.currency) else { return }
        busy = true; defer { busy = false }
        let key = session.profileKey
        value.low = parsed; value.usual = parsed; value.high = parsed; value.forecast = "usual"
        value.name = value.name.trimmingCharacters(in: .whitespacesAndNewlines)
        do {
            _ = try await session.api.saveVariableIncome(value)
            guard key == session.profileKey else { return }
            session.changed(); dismiss()
        } catch { guard key == session.profileKey else { return }; self.error = error.localizedDescription }
    }
}
