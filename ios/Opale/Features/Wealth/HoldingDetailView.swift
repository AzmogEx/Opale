import SwiftUI

/// Editing and history share a single path for assets and liabilities.
struct HoldingDetailView: View {
    let id: String, kind: String, currency: String
    let liability: Bool
    var allowsOverdraft = false
    var onChanged: () -> Void
    @Environment(SessionStore.self) private var session
    @State var name: String
    @State var note: String
    @State var archived: Bool
    @State private var valuations: [Valuation] = []
    @State private var error: String?
    @State private var busy = false
    @State private var showAdd = false
    @State private var editing: Valuation?
    @State private var deleting: Valuation?
    var body: some View {
        Form {
            Section("Informations") {
                TextField("Nom", text: $name)
                TextField("Note", text: $note, axis: .vertical)
                LabeledContent("Type", value: kind)
                LabeledContent("Devise", value: currency)
                Toggle("Archivé / clôturé", isOn: $archived)
                Text("L'archivage conserve l'historique. Les agrégats suivent les conventions de clôture du serveur.").font(.caption)
                Button("Enregistrer") { Task { await save() } }.disabled(busy || name.trimmingCharacters(in: .whitespaces).isEmpty)
            }
            Section("Historique des valorisations") {
                if valuations.isEmpty { Text("Aucune valorisation enregistrée").foregroundStyle(.secondary) }
                ForEach(valuations) { valuation in
                    Button { editing = valuation } label: {
                        HStack {
                            Text(valuation.asOf.opaleFormatted(.dateTime.day().month().year()))
                            Spacer()
                            AmountText(cents: valuation.value, currency: currency)
                        }
                    }.buttonStyle(.plain)
                    .swipeActions { Button(role: .destructive) { deleting = valuation } label: { Label("Supprimer", systemImage: "trash") } }
                }
                Button("Ajouter une valorisation") { showAdd = true }
            }
            ToolError(message: error)
        }.navigationTitle(name).navigationBarTitleDisplayMode(.inline)
        .task { await load() }
        .sheet(isPresented: $showAdd) {
            ValuationEditSheet(currency: currency, allowsOverdraft: allowsOverdraft) { value, date in
                if liability { _ = try await session.api.addLiabilityValuation(liabilityID: id, valueCents: value, asOf: date) }
                else { _ = try await session.api.addAssetValuation(assetID: id, valueCents: value, asOf: date) }
                changed()
            }
        }
        .sheet(item: $editing) { valuation in
            ValuationEditSheet(currency: currency, allowsOverdraft: allowsOverdraft, existing: valuation) { value, date in
                try await session.api.updateValuation(id: valuation.id, amount: value, date: date)
                changed()
            }
        }
        .confirmationDialog("Supprimer cette valorisation ? Le patrimoine et les projections seront recalculés.", isPresented: Binding(get: { deleting != nil }, set: { if !$0 { deleting = nil } }), titleVisibility: .visible) {
            Button("Supprimer", role: .destructive) { guard let value = deleting else { return }; Task { do { try await session.api.deleteValuation(id: value.id); changed() } catch { self.error = error.localizedDescription }; deleting = nil } }
        }
    }
    private func changed() { session.changed(); onChanged(); Task { await load() } }
    private func load() async { do { valuations = try await liability ? session.api.liabilityValuations(liabilityID: id) : session.api.assetValuations(assetID: id); error = nil } catch { self.error = error.localizedDescription } }
    private func save() async { busy = true; defer { busy = false }; do { try await session.api.updateHolding(id: id, liability: liability, name: name.trimmingCharacters(in: .whitespaces), note: note, archived: archived); changed(); error = nil } catch { self.error = error.localizedDescription } }
}

private struct ValuationEditSheet: View {
    let currency: String
    var allowsOverdraft = false
    var existing: Valuation?
    var save: (Int64, String) async throws -> Void
    @Environment(\.dismiss) private var dismiss
    @State private var amount = ""
    @State private var date = Date.now
    @State private var busy = false
    @State private var error: String?
    var body: some View {
        NavigationStack { Form {
            Section("Valeur en \(currency)") {
                TextField("Montant", text: $amount).keyboardType(allowsOverdraft ? .numbersAndPunctuation : .decimalPad)
                DatePicker("Date de référence", selection: $date, in: ...Date.now, displayedComponents: .date)
                if allowsOverdraft { Text("En cas de découvert, saisis un solde négatif avec le signe −.").font(.caption).foregroundStyle(.secondary) }
            }
            Text("Le solde constaté inclut les opérations de cette date. Les mouvements ultérieurs sont ajoutés par le serveur.").font(.caption)
            ToolError(message: error)
        }.navigationTitle(existing == nil ? "Nouvelle valorisation" : "Corriger la valorisation").toolbar {
            ToolbarItem(placement: .cancellationAction) { Button("Annuler") { dismiss() } }
            ToolbarItem(placement: .confirmationAction) { Button("Enregistrer") { Task { await submit() } }.disabled(busy || Cents.parse(amount, currency: currency) == nil) }
        }.onAppear { if let existing { amount = MoneyFormat.input(existing.value, currency: currency); date = existing.asOf } } }
    }
    private func submit() async { guard let value = Cents.parse(amount, currency: currency) else { return }; busy = true; defer { busy = false }; do { try await save(value.raw, date.opaleDayString); dismiss() } catch { self.error = error.localizedDescription } }
}
