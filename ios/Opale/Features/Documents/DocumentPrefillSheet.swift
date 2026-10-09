import SwiftUI
import UniformTypeIdentifiers

struct DocumentPrefillSheet: View {
    let kind: FinancialDocumentKind
    let expectedCurrency: String?
    let apply: (DocumentPrefillResult) -> Void
    @Environment(\.dismiss) private var dismiss
    @State private var importing = false
    @State private var reading = false
    @State private var applyingDocument = false
    @State private var parsed: ParsedFinancialDocument?
    @State private var name = ""
    @State private var amount = ""
    @State private var currency = ""
    @State private var dueDate = ""
    @State private var reviewed = false
    @State private var error: String?
    @State private var pageNote: String?
    @State private var task: Task<Void, Never>?
    private var valid: Bool {
        reviewed && !currency.isEmpty && (expectedCurrency == nil || currency == expectedCurrency) &&
        FinancialTools.amount(amount, currency: currency) != nil && name.count <= 200
    }
    var body: some View {
        NavigationStack {
            Form {
                Section {
                    Label("Lecture sur cet iPhone", systemImage: "lock.shield").font(.headline)
                    Text("Choisis un PDF ou une photo dans Fichiers. L’analyse utilise la reconnaissance de texte Apple sur l’appareil. Le document et son texte ne sont ni envoyés au serveur, ni enregistrés dans Opale.")
                        .font(.caption).foregroundStyle(.secondary)
                    Button(parsed == nil ? "Choisir un document" : "Choisir un autre document", systemImage: "doc") { importing = true }.disabled(reading)
                        .accessibilityIdentifier("prefill.choose")
                    if reading { ProgressView("Lecture du document…") }
                    if let pageNote { Text(pageNote).font(.caption).foregroundStyle(.secondary) }
                }
                if let parsed {
                    if let warning = parsed.warning { Section { Label(warning, systemImage: "exclamationmark.triangle").font(.caption).foregroundStyle(.orange) } }
                    if !parsed.candidates.isEmpty {
                        Section("Montants trouvés") {
                            ForEach(parsed.candidates) { candidate in
                                Button {
                                    amount = candidate.amountText; reviewed = false
                                } label: {
                                    VStack(alignment: .leading, spacing: 5) {
                                        Text(candidate.evidence).font(.caption).foregroundStyle(.secondary).sensitive()
                                        if let value = FinancialTools.amount(candidate.amountText, currency: currency), !currency.isEmpty { AmountText(cents: value, currency: currency) }
                                        else { Text("Choisis d’abord la devise").font(.caption) }
                                    }
                                }.disabled(currency.isEmpty || FinancialTools.amount(candidate.amountText, currency: currency) == nil)
                            }
                        }
                    }
                    Section("Champs à vérifier") {
                        TextField(kind == .payslip ? "Nom du revenu, facultatif" : "Nom du contrat, facultatif", text: $name)
                        Picker("Devise", selection: $currency) { Text("Choisir").tag(""); ForEach(FinancialTools.currencies, id: \.self) { Text($0).tag($0) } }
                        TextField(kind == .payslip ? "Net réellement payé" : "Montant à payer", text: $amount).keyboardType(.decimalPad).accessibilityIdentifier("prefill.amount")
                        if let expectedCurrency, !currency.isEmpty, expectedCurrency != currency {
                            Text("Ce formulaire attend des \(expectedCurrency). Aucune conversion automatique n’est effectuée.").font(.caption).foregroundStyle(OpaleTheme.loss)
                        }
                        if kind == .invoice { FinancialOptionalDate(title: "Date d’échéance", value: $dueDate) }
                        if kind == .payslip {
                            Text("Utilise le net reçu après prélèvement à la source, hors net imposable et cumul annuel. La prochaine date de versement reste à choisir dans le formulaire.").font(.caption).foregroundStyle(.secondary)
                        }
                        Toggle("J’ai vérifié ces champs sur le document", isOn: $reviewed).accessibilityIdentifier("prefill.reviewed")
                    }
                }
                if let error { Section { Text(error).foregroundStyle(OpaleTheme.loss) } }
            }.opaleList().navigationTitle(kind == .payslip ? "Lire une fiche de paie" : "Lire une facture").navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Fermer") { task?.cancel(); dismiss() } }
                ToolbarItem(placement: .confirmationAction) { Button("Utiliser") {
                    guard let parsedAmount = FinancialTools.amount(amount, currency: currency), valid else { return }
                    apply(DocumentPrefillResult(name: name, amount: parsedAmount, currency: currency, dueDate: dueDate.isEmpty ? nil : dueDate)); dismiss()
                }.disabled(!valid || reading).accessibilityIdentifier("prefill.apply") }
            }
            .fileImporter(isPresented: $importing, allowedContentTypes: [.pdf, .image]) { result in
                switch result { case .success(let url): read(url); case .failure(let error): self.error = error.localizedDescription }
            }
            .onChange(of: currency) { old, new in if applyingDocument { applyingDocument = false; return }; if !old.isEmpty && old != new { amount = ""; reviewed = false } }
            .onChange(of: amount) { _, _ in reviewed = false }
            .onChange(of: name) { _, _ in reviewed = false }
            .onChange(of: dueDate) { _, _ in reviewed = false }
            .onDisappear { task?.cancel() }
        }
    }
    private func read(_ url: URL) {
        task?.cancel(); reading = true; error = nil; parsed = nil; reviewed = false; pageNote = nil
        task = Task {
            defer { if !Task.isCancelled { reading = false } }
            do {
                let document = try await LocalDocumentReader.read(url)
                guard !Task.isCancelled else { return }
                let result = FinancialDocumentParser.parse(document.text, kind: kind)
                parsed = result; name = result.name; if currency != (result.currency ?? "") { applyingDocument = true }; currency = result.currency ?? ""; dueDate = result.dueDate ?? ""
                amount = result.candidates.count == 1 && result.currency != nil ? MoneyFormat.input(result.candidates[0].amount, currency: result.currency!) : ""
                if document.totalPages > document.pagesRead { pageNote = "\(document.pagesRead) premières pages lues sur \(document.totalPages). Les autres pages ne sont pas analysées." }
            } catch { if !Task.isCancelled { self.error = error.localizedDescription } }
        }
    }
}
