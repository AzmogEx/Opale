import SwiftUI

/// Split multi-catégories (EF-024) : scinder un mouvement en plusieurs
/// parts catégorisées. La somme doit égaler le montant d'origine au centime
/// — la dernière part se calcule toute seule.
struct SplitSheet: View {
    let transaction: Transaction
    let categories: [Category]
    var onSplit: () -> Void

    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss

    /// Une part en cours de saisie (montants en euros positifs, signe repris
    /// du mouvement d'origine à l'envoi).
    struct PartDraft: Identifiable {
        let id = UUID()
        var amountText = ""
        var categoryID = ""
    }

    @State private var parts: [PartDraft] = [PartDraft(), PartDraft()]
    @State private var errorMessage: String?
    @State private var isSaving = false

    /// Montant total en valeur absolue (centimes).
    private var totalAbs: Int64 { abs(transaction.amount.raw) }

    /// Centimes saisis pour les parts 0..n-2 (la dernière = le reste).
    private var enteredCents: [Int64?] {
        parts.dropLast().map { draft in
            draft.amountText.isEmpty ? nil : Cents.parse(draft.amountText)?.raw
        }
    }

    private var remainder: Int64? {
        var sum: Int64 = 0
        for c in enteredCents {
            guard let c, c > 0 else { return nil }
            sum += c
        }
        let rest = totalAbs - sum
        return rest > 0 ? rest : nil
    }

    private var isValid: Bool { remainder != nil }

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    HStack {
                        Text(transaction.label)
                            .font(.subheadline.weight(.medium))
                        Spacer()
                        AmountText(cents: transaction.amount, style: .full)
                            .font(.headline)
                    }
                }

                ForEach($parts) { $part in
                    Section("Part \(index(of: part.id) + 1)") {
                        if index(of: part.id) == parts.count - 1 {
                            // La dernière part absorbe le reste — toujours juste.
                            LabeledContent("Montant (le reste)") {
                                if let remainder {
                                    Text(MoneyFormat.euros(Cents(remainder)))
                                        .fontWeight(.semibold)
                                } else {
                                    Text("—").foregroundStyle(.secondary)
                                }
                            }
                        } else {
                            TextField("Montant (€)", text: $part.amountText)
                                .keyboardType(.decimalPad)
                        }
                        Picker("Catégorie", selection: $part.categoryID) {
                            Text("À catégoriser").tag("")
                            ForEach(categories) { c in
                                Label(c.name, systemImage: c.icon).tag(c.id)
                            }
                        }
                        .pickerStyle(.navigationLink)
                    }
                }

                Section {
                    Button {
                        parts.append(PartDraft())
                    } label: {
                        Label("Ajouter une part", systemImage: "plus.circle")
                    }
                    if parts.count > 2 {
                        Button(role: .destructive) {
                            parts.removeLast()
                        } label: {
                            Label("Retirer la dernière part", systemImage: "minus.circle")
                        }
                    }
                }

                if let errorMessage {
                    Text(errorMessage).foregroundStyle(OpaleTheme.loss)
                }
            }
            .navigationTitle("Scinder le mouvement")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("Scinder") { Task { await split() } }
                        .disabled(!isValid || isSaving)
                }
                ToolbarItem(placement: .cancellationAction) {
                    Button("Annuler") { dismiss() }
                }
            }
        }
    }

    private func index(of id: UUID) -> Int {
        parts.firstIndex { $0.id == id } ?? 0
    }

    private func split() async {
        guard let remainder else { return }
        isSaving = true
        defer { isSaving = false }

        // Reconstitue les centimes SIGNÉS (le signe du mouvement d'origine).
        let sign: Int64 = transaction.amount.raw < 0 ? -1 : 1
        var apiParts: [APIClient.SplitPart] = []
        for (i, entered) in enteredCents.enumerated() {
            guard let entered else { return }
            apiParts.append(.init(amountCents: sign * entered,
                                  categoryID: parts[i].categoryID, label: ""))
        }
        apiParts.append(.init(amountCents: sign * remainder,
                              categoryID: parts[parts.count - 1].categoryID, label: ""))

        do {
            _ = try await session.api.splitTransaction(id: transaction.id, parts: apiParts)
            onSplit()
            dismiss()
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
