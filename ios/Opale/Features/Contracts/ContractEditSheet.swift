import SwiftUI

struct ContractEditSheet: View {
    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss
    private let guided: Bool
    @State private var value: FinancialContract
    @State private var amount: String
    @State private var accounts: [Asset] = []
    @State private var error: String?
    @State private var busy = false
    @State private var showImport = false
    @State private var importingCurrency = false

    init(initial: FinancialContract, guided: Bool = false) {
        self.guided = guided
        _value = State(initialValue: initial)
        _amount = State(initialValue: initial.amount.raw == 0 ? "" : MoneyFormat.input(initial.amount, currency: initial.currency))
    }
    private var valid: Bool {
        !value.name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty && value.name.count <= 200 && value.note.count <= 2000 &&
        FinancialTools.amount(amount, currency: value.currency) != nil && (value.merchantKey.isEmpty || !value.assetID.isEmpty)
    }
    var body: some View {
        NavigationStack {
            Form {
                Section {
                    Button("Préremplir depuis une facture", systemImage: "doc.viewfinder") { showImport = true }
                        .accessibilityIdentifier("contract.import")
                    Text("Lecture sur cet iPhone, puis vérification des champs. Le document n’est pas envoyé au serveur.")
                        .font(.caption).foregroundStyle(.secondary)
                }
                Section("Tarif et prélèvement") {
                    TextField("Nom du contrat", text: $value.name).accessibilityIdentifier("contract.name")
                    Picker("Catégorie", selection: $value.category) {
                        Text("Abonnement").tag("subscription"); Text("Assurance").tag("insurance")
                        Text("Logement").tag("housing"); Text("Énergie et services").tag("utilities"); Text("Autre").tag("other")
                    }
                    TextField("Montant par échéance (\(value.currency))", text: $amount).keyboardType(.decimalPad).accessibilityIdentifier("contract.amount")
                    Picker("Devise", selection: $value.currency) { ForEach(FinancialTools.currencies, id: \.self) { Text($0).tag($0) } }.disabled(!value.assetID.isEmpty)
                    FinancialFrequencyPicker(value: $value.frequency)
                    DatePicker("Prochain prélèvement", selection: FinancialDate.binding($value.nextDueDate), in: FinancialDate.futureRange, displayedComponents: .date)
                    Picker("Compte débité", selection: $value.assetID) {
                        Text("Suivi sans prévision").tag("")
                        if !value.assetID.isEmpty, !accounts.contains(where: { $0.id == value.assetID }) { Text("Compte indisponible — à changer").tag(value.assetID) }
                        ForEach(accounts) { Text("\($0.name) · \($0.currency)").tag($0.id) }
                    }.accessibilityIdentifier("contract.account")
                    Text("Avec un compte, ce tarif sera prévu au calendrier. Il ne modifie pas ton solde comme si le prélèvement était déjà passé.")
                        .font(.caption).foregroundStyle(.secondary)
                }
                if guided {
                    Section { DisclosureGroup("Options : hausses, engagement et rappels") { advancedFields } }
                } else { advancedFields }
                if let error { Section { Text(error).foregroundStyle(OpaleTheme.loss) } }
            }.opaleList().scrollDismissesKeyboard(.interactively).disabled(busy)
            .navigationTitle(value.revision == 0 ? "Nouveau contrat" : "Modifier le contrat").navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Annuler") { dismiss() }.disabled(busy) }
                ToolbarItem(placement: .confirmationAction) { Button("Enregistrer") { Task { await save() } }.disabled(busy || !valid).accessibilityIdentifier("contract.save") }
            }
            .task { await loadAccounts() }
            .onChange(of: value.assetID) { _, id in
                if let account = accounts.first(where: { $0.id == id }), account.currency != value.currency { value.currency = account.currency }
                if id.isEmpty { value.merchantKey = "" }
            }
            .onChange(of: value.currency) { _, _ in if importingCurrency { importingCurrency = false; return }; amount = ""; error = "Devise changée : ressaisis le montant dans la nouvelle devise." }
            .sheet(isPresented: $showImport) { DocumentPrefillSheet(kind: .invoice, expectedCurrency: nil) { result in
                if value.currency != result.currency { value.assetID = ""; value.merchantKey = "" }
                if value.currency != result.currency { importingCurrency = true }
                value.currency = result.currency
                if !result.name.isEmpty { value.name = result.name }
                amount = MoneyFormat.input(result.amount, currency: result.currency)
                error = nil
                if let day = result.dueDate {
                    if day >= Date.now.opaleDayString { value.nextDueDate = day }
                    else { error = "La date de la facture est passée. Choisis la prochaine date de prélèvement avant d’enregistrer." }
                }
            } }
            .interactiveDismissDisabled(busy)
        }
    }
    @ViewBuilder private var advancedFields: some View {
                Section("Détection des hausses") {
                    TextField("Libellé marchand sur le relevé, facultatif", text: $value.merchantKey)
                        .textInputAutocapitalization(.characters).autocorrectionDisabled().accessibilityIdentifier("contract.merchant")
                    Text("Copie le nom du marchand tel qu’il apparaît sur tes opérations. Le dernier débit comptabilisé de ce marchand, sur ce compte et dans cette devise, sera comparé au tarif déclaré. Chaque hausse devra être confirmée. Sans libellé marchand, vérifie les doublons avec les récurrences détectées.")
                        .font(.caption).foregroundStyle(.secondary)
                }
                Section("Engagement et rappels") {
                    FinancialOptionalDate(title: "Fin d’essai gratuit", value: $value.trialEnd)
                    FinancialOptionalDate(title: "Fin d’engagement", value: $value.commitmentEnd)
                    FinancialOptionalDate(title: "Date de renouvellement", value: $value.renewalDate)
                    Toggle("Reconduction automatique", isOn: $value.autoRenew)
                    Stepper("Préavis : \(value.noticeDays) jours", value: $value.noticeDays, in: 0...365)
                    Stepper("Rappel : \(value.reminderDays) jours avant", value: $value.reminderDays, in: 0...90)
                    Text("Renseigne les dates et le préavis indiqués dans ton contrat. Les rappels s’affichent dans Contrats et Alertes ; les notifications nécessitent leur activation.")
                        .font(.caption).foregroundStyle(.secondary)
                }
                Section("Notes") { TextField("Référence, conditions de résiliation…", text: $value.note, axis: .vertical).lineLimit(3...6) }
    }
    private func loadAccounts() async {
        let key = session.profileKey, api = session.api
        do {
            let assets = try await api.listAssets()
            guard key == session.profileKey, !Task.isCancelled else { return }
            accounts = assets.filter { !$0.archived && ($0.kind == .checking || $0.kind == .savings) }
            if guided, value.revision == 0, value.assetID.isEmpty,
               let account = accounts.first(where: { $0.currency == value.currency }) { value.assetID = account.id }
        } catch { guard key == session.profileKey, !Task.isCancelled else { return }; self.error = error.localizedDescription }
    }
    private func save() async {
        guard valid, let parsed = FinancialTools.amount(amount, currency: value.currency) else { return }
        value.amount = parsed; busy = true; defer { busy = false }
        let key = session.profileKey, api = session.api
        do { let _ = try await api.saveContract(value); guard key == session.profileKey else { return }; session.changed(); dismiss() }
        catch { guard key == session.profileKey else { return }; self.error = error.localizedDescription }
    }
}
