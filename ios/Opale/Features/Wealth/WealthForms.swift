import SwiftUI

// Formulaires de saisie du patrimoine (EF-030→032) :
// création d'actif, création de dette, ajout de valorisation.

/// Création d'un actif, avec valorisation initiale optionnelle.
struct AssetFormSheet: View {
    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss
    var onSaved: () -> Void

    @State private var name = ""
    @State private var kind: AssetKind = .checking
    @State private var initialValue = ""
    @State private var errorMessage: String?
    @State private var isSaving = false

    private var parsedValue: Cents? { Cents.parse(initialValue) }
    private var isValid: Bool {
        !name.trimmingCharacters(in: .whitespaces).isEmpty
            && (initialValue.isEmpty || parsedValue != nil)
    }

    var body: some View {
        NavigationStack {
            Form {
                Section("Actif") {
                    TextField("Nom (ex. Compte courant BNP)", text: $name)
                    Picker("Type", selection: $kind) {
                        ForEach(AssetKind.allCases) { kind in
                            Label(kind.label, systemImage: kind.systemImage).tag(kind)
                        }
                    }
                }
                Section("Valeur actuelle (optionnel)") {
                    TextField("Ex. 12 500,00", text: $initialValue)
                        .keyboardType(.decimalPad)
                    if !initialValue.isEmpty, parsedValue == nil {
                        Text("Montant invalide")
                            .font(.footnote)
                            .foregroundStyle(OpaleTheme.loss)
                    }
                }
                if let errorMessage {
                    Text(errorMessage).foregroundStyle(OpaleTheme.loss)
                }
            }
            .navigationTitle("Nouvel actif")
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
        }
    }

    private func save() async {
        isSaving = true
        defer { isSaving = false }
        do {
            let asset = try await session.api.createAsset(
                name: name.trimmingCharacters(in: .whitespaces),
                kind: kind
            )
            if let value = parsedValue {
                _ = try await session.api.addAssetValuation(
                    assetID: asset.id,
                    valueCents: value.raw,
                    asOf: Date.now.opaleDayString
                )
            }
            onSaved()
            dismiss()
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}

/// Création d'une dette, avec capital restant dû optionnel.
struct LiabilityFormSheet: View {
    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss
    var onSaved: () -> Void

    @State private var name = ""
    @State private var kind: LiabilityKind = .mortgage
    @State private var initialValue = ""
    @State private var errorMessage: String?
    @State private var isSaving = false

    private var parsedValue: Cents? { Cents.parse(initialValue) }
    private var isValid: Bool {
        !name.trimmingCharacters(in: .whitespaces).isEmpty
            && (initialValue.isEmpty || parsedValue != nil)
    }

    var body: some View {
        NavigationStack {
            Form {
                Section("Dette") {
                    TextField("Nom (ex. Crédit immobilier)", text: $name)
                    Picker("Type", selection: $kind) {
                        ForEach(LiabilityKind.allCases) { kind in
                            Label(kind.label, systemImage: kind.systemImage).tag(kind)
                        }
                    }
                }
                Section("Capital restant dû (optionnel)") {
                    TextField("Ex. 162 000", text: $initialValue)
                        .keyboardType(.decimalPad)
                    if !initialValue.isEmpty, parsedValue == nil {
                        Text("Montant invalide")
                            .font(.footnote)
                            .foregroundStyle(OpaleTheme.loss)
                    }
                }
                if let errorMessage {
                    Text(errorMessage).foregroundStyle(OpaleTheme.loss)
                }
            }
            .navigationTitle("Nouvelle dette")
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
        }
    }

    private func save() async {
        isSaving = true
        defer { isSaving = false }
        do {
            let liability = try await session.api.createLiability(
                name: name.trimmingCharacters(in: .whitespaces),
                kind: kind
            )
            if let value = parsedValue {
                _ = try await session.api.addLiabilityValuation(
                    liabilityID: liability.id,
                    valueCents: value.raw,
                    asOf: Date.now.opaleDayString
                )
            }
            onSaved()
            dismiss()
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}

/// Ajout d'une valorisation datée (EF-032) — mutualisé actif/dette via closure.
struct ValuationSheet: View {
    var title: String
    /// Persiste la valorisation (centimes, date `yyyy-MM-dd`).
    var save: (Int64, String) async throws -> Void
    var onSaved: () -> Void

    @Environment(\.dismiss) private var dismiss
    @State private var value = ""
    @State private var asOf = Date.now
    @State private var errorMessage: String?
    @State private var isSaving = false

    private var parsedValue: Cents? { Cents.parse(value) }

    var body: some View {
        NavigationStack {
            Form {
                Section("Nouvelle valeur") {
                    TextField("Ex. 43 200,00", text: $value)
                        .keyboardType(.decimalPad)
                    DatePicker("Date", selection: $asOf, in: ...Date.now, displayedComponents: .date)
                }
                if !value.isEmpty, parsedValue == nil {
                    Text("Montant invalide").foregroundStyle(OpaleTheme.loss)
                }
                if let errorMessage {
                    Text(errorMessage).foregroundStyle(OpaleTheme.loss)
                }
            }
            .navigationTitle(title)
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("Enregistrer") { Task { await doSave() } }
                        .disabled(parsedValue == nil || isSaving)
                }
                ToolbarItem(placement: .cancellationAction) {
                    Button("Annuler") { dismiss() }
                }
            }
        }
    }

    private func doSave() async {
        guard let parsedValue else { return }
        isSaving = true
        defer { isSaving = false }
        do {
            try await save(parsedValue.raw, asOf.opaleDayString)
            onSaved()
            dismiss()
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}

extension Date {
    /// Format `yyyy-MM-dd` attendu par le backend (colonne DATE).
    var opaleDayString: String {
        formatted(.iso8601.year().month().day())
    }
}
