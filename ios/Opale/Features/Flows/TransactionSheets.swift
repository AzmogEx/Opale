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
    @State private var applyToSimilar = true
    @State private var errorMessage: String?
    @State private var isSaving = false

    init(transaction: Transaction, categories: [Category], onSaved: @escaping () -> Void) {
        self.transaction = transaction
        self.categories = categories
        self.onSaved = onSaved
        _label = State(initialValue: transaction.label)
        _note = State(initialValue: transaction.note)
        _categoryID = State(initialValue: transaction.categoryID ?? "")
    }

    private var categoryChanged: Bool {
        categoryID != (transaction.categoryID ?? "")
    }

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    HStack {
                        Text(transaction.occurredOn.formatted(.dateTime.day().month(.wide).year()))
                            .foregroundStyle(.secondary)
                        Spacer()
                        AmountText(cents: transaction.amount, style: .full)
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

                Section("Détails") {
                    TextField("Libellé", text: $label)
                    TextField("Note", text: $note, axis: .vertical)
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
                        .disabled(isSaving)
                }
                ToolbarItem(placement: .cancellationAction) {
                    Button("Annuler") { dismiss() }
                }
            }
        }
    }

    private func save() async {
        isSaving = true
        defer { isSaving = false }
        do {
            var patch = APIClient.PatchTransactionRequest()
            if label != transaction.label { patch.label = label }
            if note != transaction.note { patch.note = note }
            if categoryChanged {
                patch.categoryID = categoryID
                patch.applyToSimilar = applyToSimilar && !categoryID.isEmpty
            }
            _ = try await session.api.updateTransaction(id: transaction.id, patch)
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
    @State private var assetID = ""
    @State private var label = ""
    @State private var amountText = ""
    @State private var isExpense = true
    @State private var date = Date.now
    @State private var categoryID = ""
    @State private var errorMessage: String?
    @State private var isSaving = false

    private var parsedAmount: Cents? { Cents.parse(amountText) }
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
                    TextField("Montant (ex. 12,50)", text: $amountText)
                        .keyboardType(.decimalPad)
                    DatePicker("Date", selection: $date, in: ...Date.now, displayedComponents: .date)
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
                assets = (try? await session.api.listAssets()) ?? []
                if assetID.isEmpty, let first = assets.first { assetID = first.id }
            }
        }
    }

    private func save() async {
        guard let amount = parsedAmount else { return }
        isSaving = true
        defer { isSaving = false }
        do {
            _ = try await session.api.createTransaction(.init(
                assetID: assetID,
                amountCents: isExpense ? -amount.raw : amount.raw,
                occurredOn: date.opaleDayString,
                label: label.trimmingCharacters(in: .whitespaces),
                categoryID: categoryID,
                note: ""
            ))
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
                            isImporting ? "Import en cours…" : "Choisir le fichier CSV",
                            systemImage: "square.and.arrow.down"
                        )
                    }
                    .disabled(assetID.isEmpty || isImporting)
                } footer: {
                    Text("Export CSV de ta banque : les doublons sont ignorés, les libellés nettoyés et les catégories proposées automatiquement.")
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
                allowedContentTypes: [.commaSeparatedText, .plainText, .text],
                allowsMultipleSelection: false
            ) { pick in
                Task { await handlePick(pick) }
            }
            .task {
                assets = (try? await session.api.listAssets()) ?? []
                if assetID.isEmpty, let first = assets.first { assetID = first.id }
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
            let data = try Data(contentsOf: url)
            // Le backend gère lui-même l'UTF-8 et le Windows-1252.
            let csv = String(data: data, encoding: .utf8)
                ?? String(data: data, encoding: .isoLatin1)
                ?? ""
            result = try await session.api.importCSV(assetID: assetID, csv: csv)
            errorMessage = nil
            onImported()
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
