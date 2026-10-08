import SwiftUI
import UniformTypeIdentifiers

// MARK: - Édition (correction de catégorie apprenante, EF-022/024)

struct TransactionEditSheet: View {
    let transaction: Transaction
    let categories: [Category]
    var onSaved: () -> Void

    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss

    @State private var label: String
    @State private var note: String
    @State private var categoryID: String
    @State private var amountText: String
    @State private var occurredOn: Date
    @State private var accountID: String
    @State private var accounts: [Asset] = []
    @State private var selectedSpaceID = ""
    @State private var flowKind = "expense_income"
    @State private var applyToSimilar = true
    @State private var errorMessage: String?
    @State private var isSaving = false
    // Espace partagé (EF-007) : dépense commune du foyer.
    @State private var spaces: [Space] = []
    // Split multi-catégories (EF-024).
    @State private var showSplit = false
    @State private var proposal: LocalAI.ValidatedProposal?
    @State private var proposing = false

    init(transaction: Transaction, categories: [Category], onSaved: @escaping () -> Void) {
        self.transaction = transaction
        self.categories = categories
        self.onSaved = onSaved
        _amountText = State(initialValue: MoneyFormat.input(transaction.amount, currency: transaction.currency ?? "EUR"))
        _occurredOn = State(initialValue: transaction.occurredOn)
        _accountID = State(initialValue: transaction.assetID)
        _selectedSpaceID = State(initialValue: transaction.spaceID ?? "")
        _flowKind = State(initialValue: transaction.flowKind ?? "expense_income")
        _label = State(initialValue: transaction.label)
        _note = State(initialValue: transaction.note)
        _categoryID = State(initialValue: transaction.categoryID ?? "")
    }

    private var currency: String { accounts.first { $0.id == accountID }?.currency ?? transaction.currency ?? "EUR" }

    private var categoryChanged: Bool {
        categoryID != (transaction.categoryID ?? "")
    }

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    HStack {
                        Text(transaction.occurredOn.opaleFormatted(.dateTime.day().month(.wide).year()))
                            .foregroundStyle(.secondary)
                        Spacer()
                        AmountText(cents: transaction.amount, style: .full, currency: transaction.currency ?? "EUR")
                            .font(.headline)
                    }
                    if transaction.rawLabel != transaction.label {
                        LabeledContent("Libellé bancaire") {
                            Text(transaction.rawLabel)
                                .font(.caption)
                                .lineLimit(2)
                        }
                    }
                }

                if transaction.transferID != nil { Section { Text("Ce mouvement appartient à un virement lié. Pour le corriger, supprime le virement entier depuis Flux puis recrée-le.").font(.caption) } }
                Section("Catégorie") {
                    Picker("Catégorie", selection: $categoryID) {
                        Text("À catégoriser").tag("")
                        ForEach(categories) { c in
                            Label(c.name, systemImage: c.icon).tag(c.id)
                        }
                    }
                    .pickerStyle(.navigationLink)

                    if categoryChanged, !categoryID.isEmpty {
                        Toggle(isOn: $applyToSimilar) {
                            VStack(alignment: .leading, spacing: 2) {
                                Text("Appliquer au marchand")
                                Text("Tous les mouvements de ce marchand, passés et futurs")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                        }
                    }
                }

                if transaction.linkedLiabilityID != nil { Section { Text("Capital lié à une dette. Pour changer le montant, la date, la nature ou le compte, supprime puis recrée ce remboursement.").font(.caption) } }
                Section("Mouvement") {
                    TextField("Montant signé (\(currency))", text: $amountText).keyboardType(.numbersAndPunctuation)
                    DatePicker("Date", selection: $occurredOn, in: ...Date.now, displayedComponents: .date)
                    Picker("Compte", selection: $accountID) { ForEach(accounts) { Text($0.name).tag($0.id) } }
                    Picker("Nature", selection: $flowKind) {
                        Text("Revenu / dépense").tag("expense_income")
                        Text("Virement interne").tag("internal_transfer")
                        Text("Apport investissement").tag("investment_contribution")
                        Text("Retrait investissement").tag("investment_withdrawal")
                        Text("Capital remboursé").tag("loan_principal")
                        Text("Intérêts").tag("interest")
                        Text("Frais").tag("fee")
                    }
                }
                .disabled(transaction.linkedLiabilityID != nil || transaction.transferID != nil)
                Section("Détails") {
                    TextField("Libellé", text: $label)
                    TextField("Note", text: $note, axis: .vertical)
                }

                Section("Suggestion locale sur l’iPhone") {
                    Button(proposing ? "Analyse locale…" : "Proposer catégorie et libellé") { Task {
                        proposing = true; defer { proposing = false }
                        do { proposal = try await LocalAI.proposeTransaction(rawLabel: transaction.rawLabel, categories: categories); errorMessage = nil }
                        catch { errorMessage = error.localizedDescription }
                    } }.disabled(proposing || !LocalAI.isAvailable || transaction.transferID != nil)
                    if !LocalAI.isAvailable { Text("Apple Intelligence indisponible sur cet appareil. La correction manuelle et les règles du serveur restent disponibles.").font(.caption) }
                    if let proposal {
                        Text(proposal.label)
                        Text(proposal.categoryID.flatMap { id in categories.first { $0.id == id }?.name } ?? "Catégorie incertaine : choisis-la manuellement").font(.caption)
                        Button("Utiliser cette proposition") { label = proposal.label; if let id = proposal.categoryID { categoryID = id }; self.proposal = nil }
                    }
                    Text("Le libellé reste sur l’appareil. Vérifie la proposition puis enregistre : une correction de catégorie alimente la règle du marchand.").font(.caption)
                }
                // Split multi-catégories (EF-024).
                Section {
                    Button {
                        showSplit = true
                    } label: {
                        Label("Scinder en plusieurs catégories", systemImage: "square.split.2x1")
                    }.disabled(transaction.transferID != nil || transaction.linkedLiabilityID != nil)
                }

                // Dépense commune (EF-007) — seulement si un espace existe
                // et que le mouvement est une dépense.
                if !spaces.isEmpty, transaction.amount.raw < 0 {
                    Picker("Espace partagé", selection: $selectedSpaceID) { Text("Aucun").tag(""); ForEach(spaces) { Text($0.name).tag($0.id) } }
                }
                if let errorMessage {
                    Text(errorMessage).foregroundStyle(OpaleTheme.loss)
                }
            }
            .navigationTitle("Mouvement")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("Enregistrer") { Task { await save() } }
                        .disabled(transaction.transferID != nil || isSaving || Cents.parse(amountText, currency: currency) == nil)
                }
                ToolbarItem(placement: .cancellationAction) {
                    Button("Annuler") { dismiss() }
                }
            }
            .task {
                do {
                    spaces = try await session.api.spaces()
                    accounts = try await session.api.listAssets()
                } catch { errorMessage = error.localizedDescription }
            }
            .sheet(isPresented: $showSplit) {
                SplitSheet(transaction: transaction, categories: categories) {
                    onSaved()
                    dismiss()
                }
            }
        }
    }

    private func save() async {
        isSaving = true
        defer { isSaving = false }
        do {
            var patch = APIClient.PatchTransactionRequest()
            if transaction.linkedLiabilityID == nil {
                patch.amountCents = Cents.parse(amountText, currency: currency)?.raw
                patch.occurredOn = occurredOn.opaleDayString
                patch.assetID = accountID
                patch.flowKind = flowKind
            }
            if label != transaction.label { patch.label = label }
            if note != transaction.note { patch.note = note }
            if categoryChanged {
                patch.categoryID = categoryID
                patch.applyToSimilar = applyToSimilar && !categoryID.isEmpty
            }
            _ = try await session.api.updateTransaction(id: transaction.id, patch)
            // Marquage commun : appel dédié, seulement si l'état a changé.
            if selectedSpaceID != (transaction.spaceID ?? "") {
                try await session.api.setTransactionSpace(
                    transactionID: transaction.id,
                    spaceID: selectedSpaceID.isEmpty ? nil : selectedSpaceID
                )
            }
            session.changed()
            onSaved()
            dismiss()
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}

// MARK: - Saisie manuelle (EF-020)

struct ManualTransactionSheet: View {
    let categories: [Category]
    var onSaved: () -> Void

    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss

    @State private var assets: [Asset] = []
    @State private var liabilities: [Liability] = []
    @State private var flowKind = "expense_income"
    @State private var linkedLiabilityID = ""
    @State private var assetID = ""
    @State private var label = ""
    @State private var amountText = ""
    @State private var isExpense = true
    @State private var date = Date.now
    @State private var categoryID = ""
    @State private var errorMessage: String?
    @State private var isSaving = false

    private var currency: String { assets.first { $0.id == assetID }?.currency ?? "EUR" }
    private var parsedAmount: Cents? { Cents.parse(amountText, currency: currency) }
    private var isValid: Bool {
        !assetID.isEmpty && !label.trimmingCharacters(in: .whitespaces).isEmpty
            && (parsedAmount?.raw ?? 0) > 0
    }

    var body: some View {
        NavigationStack {
            Form {
                Section("Mouvement") {
                    Picker("Sens", selection: $isExpense) {
                        Text("Dépense").tag(true)
                        Text("Revenu").tag(false)
                    }
                    .pickerStyle(.segmented)
                    TextField("Libellé (ex. Boulangerie)", text: $label)
                    TextField("Montant (\(currency))", text: $amountText)
                        .keyboardType(.decimalPad)
                    DatePicker("Date", selection: $date, in: ...Date.now, displayedComponents: .date)
                }
                Section("Nature du mouvement") {
                    Picker("Nature", selection: $flowKind) {
                        Text("Revenu / dépense").tag("expense_income")
                        Text("Apport investissement").tag("investment_contribution")
                        Text("Retrait investissement").tag("investment_withdrawal")
                        Text("Capital remboursé").tag("loan_principal")
                        Text("Intérêts").tag("interest")
                        Text("Frais").tag("fee")
                    }
                    if flowKind == "loan_principal" {
                        Picker("Dette remboursée", selection: $linkedLiabilityID) {
                            Text("Aucune dette liée").tag("")
                            ForEach(liabilities.filter { !$0.archived && $0.currency == currency }) { Text($0.name).tag($0.id) }
                        }
                        Text(linkedLiabilityID.isEmpty ? "Sans lien, ce mouvement ne diminuera pas le capital restant dû d’une dette." : "Le capital restant dû diminuera du montant saisi ; les intérêts et frais doivent être saisis séparément.").font(.caption)
                    }
                }
                Section("Compte") {
                    Picker("Compte", selection: $assetID) {
                        ForEach(assets) { a in
                            Text(a.name).tag(a.id)
                        }
                    }
                }
                Section("Catégorie (optionnel)") {
                    Picker("Catégorie", selection: $categoryID) {
                        Text("Automatique").tag("")
                        ForEach(categories) { c in
                            Label(c.name, systemImage: c.icon).tag(c.id)
                        }
                    }
                    .pickerStyle(.navigationLink)
                }
                if let errorMessage {
                    Text(errorMessage).foregroundStyle(OpaleTheme.loss)
                }
            }
            .navigationTitle("Nouvelle transaction")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("Ajouter") { Task { await save() } }
                        .disabled(!isValid || isSaving)
                }
                ToolbarItem(placement: .cancellationAction) {
                    Button("Annuler") { dismiss() }
                }
            }
            .task {
                do { assets = try await session.api.listAssets(); liabilities = try await session.api.listLiabilities(); if assetID.isEmpty, let first = assets.first { assetID = first.id } } catch { errorMessage = error.localizedDescription }
            }
        }
    }

    private func save() async {
        guard let amount = parsedAmount else { return }
        guard flowKind != "loan_principal" || isExpense else { errorMessage = "Un remboursement de capital doit être une dépense."; return }
        isSaving = true
        defer { isSaving = false }
        do {
            _ = try await session.api.createTransaction(.init(
                flowKind: flowKind,
                linkedLiabilityID: flowKind == "loan_principal" && !linkedLiabilityID.isEmpty ? linkedLiabilityID : nil,
                assetID: assetID,
                amountCents: isExpense ? -amount.raw : amount.raw,
                occurredOn: date.opaleDayString,
                label: label.trimmingCharacters(in: .whitespaces),
                categoryID: categoryID,
                note: ""
            ))
            session.changed()
            onSaved()
            dismiss()
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}

// MARK: - Import CSV (EF-021)

struct ImportCSVSheet: View {
    var onImported: () -> Void

    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss

    @State private var assets: [Asset] = []
    @State private var assetID = ""
    @State private var showPicker = false
    @State private var result: ImportResult?
    @State private var errorMessage: String?
    @State private var isImporting = false

    var body: some View {
        NavigationStack {
            Form {
                Section("Compte de destination") {
                    Picker("Compte", selection: $assetID) {
                        ForEach(assets) { a in
                            Text(a.name).tag(a.id)
                        }
                    }
                }

                Section {
                    Button {
                        showPicker = true
                    } label: {
                        Label(
                            isImporting ? "Import en cours…" : "Choisir un fichier CSV ou OFX",
                            systemImage: "square.and.arrow.down"
                        )
                    }
                    .disabled(assetID.isEmpty || isImporting)
                } footer: {
                    Text("CSV ou OFX : 5 Mio et 10 000 opérations maximum. Les doublons sont ignorés ; des règles proposent les catégories et nettoient les libellés. Une suggestion IA sur l’appareil est disponible depuis chaque transaction compatible.")
                }

                if let result {
                    Section("Résultat") {
                        LabeledContent("Importées", value: "\(result.imported)")
                        LabeledContent("Doublons ignorés", value: "\(result.duplicates)")
                        LabeledContent("Catégorisées auto", value: "\(result.categorized)")
                    }
                }
                if let errorMessage {
                    Text(errorMessage).foregroundStyle(OpaleTheme.loss)
                }
            }
            .navigationTitle("Importer un relevé")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button(result == nil ? "Fermer" : "Terminé") { dismiss() }
                }
            }
            .fileImporter(
                isPresented: $showPicker,
                allowedContentTypes: [.commaSeparatedText, .plainText, .text, .data],
                allowsMultipleSelection: false
            ) { pick in
                Task { await handlePick(pick) }
            }
            .task {
                do { assets = try await session.api.listAssets(); if assetID.isEmpty, let first = assets.first { assetID = first.id } } catch { errorMessage = error.localizedDescription }
            }
        }
    }

    private func handlePick(_ pick: Result<[URL], Error>) async {
        guard case .success(let urls) = pick, let url = urls.first else { return }
        isImporting = true
        defer { isImporting = false }
        do {
            let secured = url.startAccessingSecurityScopedResource()
            defer { if secured { url.stopAccessingSecurityScopedResource() } }
            let size = try url.resourceValues(forKeys: [.fileSizeKey]).fileSize ?? 0
            guard size <= 5 * 1024 * 1024 else { throw APIError.badStatus(413, message: "Fichier trop volumineux (5 Mio maximum).") }
            let data = try Data(contentsOf: url)
            // Le transport JSON est UTF-8 ; décodage des exports Windows avant envoi.
            let csv = String(data: data, encoding: .utf8)
                ?? String(data: data, encoding: .windowsCP1252)
                ?? String(data: data, encoding: .isoLatin1)
                ?? ""
            result = try await session.api.importCSV(assetID: assetID, csv: csv)
            errorMessage = nil
            session.changed()
            onImported()
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
