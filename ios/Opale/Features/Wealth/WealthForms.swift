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
    @State private var currency = "EUR"
    @State private var initialValue = ""
    @State private var initialDate = Date.now
    @State private var requestID = UUID().uuidString
    @State private var errorMessage: String?
    @State private var isSaving = false

    /// Devises proposées (EF-008) — extensibles via l'éditeur de taux.
    static let currencies = ["EUR", "USD", "GBP", "CHF", "JPY", "KWD", "CAD", "AUD"]

    private var parsedValue: Cents? { Cents.parse(initialValue, currency: currency) }
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
                    Picker("Devise", selection: $currency) {
                        ForEach(Self.currencies, id: \.self) { c in
                            Text(c).tag(c)
                        }
                    }
                }
                Section("Valeur de référence en \(currency) (optionnel)") {
                    DatePicker("Date de référence", selection: $initialDate, in: ...Date.now, displayedComponents: .date)
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
            _ = try await session.api.createAsset(
                name: name.trimmingCharacters(in: .whitespaces),
                kind: kind,
                currency: currency, initialValue: parsedValue?.raw, initialAsOf: initialDate.opaleDayString, requestID: requestID
            )
            session.changed()
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
    @State private var currency = "EUR"
    @State private var initialValue = ""
    @State private var initialDate = Date.now
    @State private var requestID = UUID().uuidString
    @State private var errorMessage: String?
    @State private var isSaving = false

    private var parsedValue: Cents? { Cents.parse(initialValue, currency: currency) }
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
                Section { Picker("Devise", selection: $currency) { ForEach(AssetFormSheet.currencies, id: \.self) { Text($0).tag($0) } } }
                Section("Capital restant dû en \(currency) (optionnel)") {
                    DatePicker("Date de référence", selection: $initialDate, in: ...Date.now, displayedComponents: .date)
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
            _ = try await session.api.createLiability(
                name: name.trimmingCharacters(in: .whitespaces),
                kind: kind, currency: currency, initialValue: parsedValue?.raw, initialAsOf: initialDate.opaleDayString, requestID: requestID
            )
            session.changed()
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
    /// Stable French civil-date presentation, independent of the device timezone.
    func opaleFormatted(_ style: Date.FormatStyle) -> String {
        var presentation = style
        presentation.locale = Locale(identifier: "fr_FR")
        presentation.calendar = .opale
        presentation.timeZone = TimeZone(identifier: "Europe/Paris")!
        return formatted(presentation)
    }

    /// Format `yyyy-MM-dd` attendu par le backend (colonne DATE).
    var opaleDayString: String {
        let formatter = DateFormatter()
        formatter.calendar = Calendar(identifier: .gregorian)
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.timeZone = TimeZone(identifier: "Europe/Paris")
        formatter.dateFormat = "yyyy-MM-dd"
        return formatter.string(from: self)
    }

    /// Inverse : lit une date `yyyy-MM-dd` renvoyée par le backend.
    nonisolated static func fromOpaleDay(_ day: String) -> Date? {
        guard day.range(of: #"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"#, options: .regularExpression) != nil else { return nil }
        let formatter = DateFormatter()
        formatter.calendar = Calendar(identifier: .gregorian)
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.timeZone = TimeZone(identifier: "Europe/Paris")
        formatter.dateFormat = "yyyy-MM-dd"
        formatter.isLenient = false
        guard let date = formatter.date(from: day), formatter.string(from: date) == day else { return nil }
        return date
    }
}


extension Calendar {
    nonisolated static var opale: Calendar {
        var calendar = Calendar(identifier: .gregorian)
        calendar.locale = Locale(identifier: "fr_FR")
        calendar.timeZone = TimeZone(identifier: "Europe/Paris")!
        return calendar
    }
}
