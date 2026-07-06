import SwiftUI

/// Valeur des objets (EF-035) : montres, véhicules, or, œuvres…
/// avec l'écart entre prix d'achat et valeur estimée.
struct ObjectsView: View {
    @Environment(SessionStore.self) private var session

    @State private var objects: [ObjectStatus] = []
    @State private var editing: ObjectStatus?
    @State private var loaded = false

    var body: some View {
        List {
            if loaded && objects.isEmpty {
                ContentUnavailableView(
                    "Aucun objet de valeur",
                    systemImage: "sparkle.magnifyingglass",
                    description: Text("Ajoute un actif de type « Objet de valeur », « Véhicule » ou « Or / métaux » dans Patrimoine.")
                )
            }
            ForEach(objects) { object in
                Button {
                    editing = object
                } label: {
                    row(object)
                }
                .buttonStyle(.plain)
            }
        }
        .opaleList()
        .navigationTitle("Objets")
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
        .refreshable { await load() }
        .sheet(item: $editing) { object in
            ObjectFormSheet(object: object) {
                Task { await load() }
            }
            .presentationDetents([.medium, .large])
        }
    }

    @ViewBuilder
    private func row(_ o: ObjectStatus) -> some View {
        HStack {
            Image(systemName: o.asset.kind.systemImage)
                .font(.title3)
                .foregroundStyle(OpaleTheme.accent)
                .frame(width: 32)
            VStack(alignment: .leading, spacing: 2) {
                Text(o.asset.name).font(.body.weight(.medium))
                HStack(spacing: 4) {
                    if !o.details.category.isEmpty { Text(o.details.category.capitalized) }
                    if !o.details.brand.isEmpty { Text("· \(o.details.brand)") }
                    if o.details.insured {
                        Image(systemName: "checkmark.shield.fill")
                            .foregroundStyle(OpaleTheme.gain)
                    }
                }
                .font(.caption)
                .foregroundStyle(.secondary)
            }
            Spacer()
            VStack(alignment: .trailing, spacing: 2) {
                AmountText(cents: o.asset.latestValue ?? .zero, style: .whole)
                    .font(.callout.weight(.semibold))
                if o.details.purchasePrice.raw > 0, o.change.raw != 0 {
                    AmountText(cents: o.change, style: .signedDelta)
                        .font(.caption)
                        .foregroundStyle(o.change.raw < 0 ? OpaleTheme.loss : OpaleTheme.gain)
                }
            }
        }
    }

    private func load() async {
        objects = (try? await session.api.objects()) ?? []
        loaded = true
    }
}

/// Saisie des détails d'un objet : catégorie, marque, achat, assurance.
private struct ObjectFormSheet: View {
    let object: ObjectStatus
    var onSaved: () -> Void

    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss

    @State private var category = ""
    @State private var brand = ""
    @State private var purchaseText = ""
    @State private var hasDate = false
    @State private var purchaseDate = Date.now
    @State private var insured = false
    @State private var errorMessage: String?

    var body: some View {
        NavigationStack {
            Form {
                Section("L'objet") {
                    TextField("Catégorie (montre, voiture, or…)", text: $category)
                    TextField("Marque / auteur / référence", text: $brand)
                }
                Section("Achat") {
                    TextField("Prix d'achat (€)", text: $purchaseText)
                        .keyboardType(.decimalPad)
                    Toggle("Date d'achat", isOn: $hasDate)
                    if hasDate {
                        DatePicker("Le", selection: $purchaseDate, displayedComponents: .date)
                    }
                }
                Section {
                    Toggle("Assuré", isOn: $insured)
                } footer: {
                    Text("Pense à rattacher la facture et l'attestation dans le coffre-fort.")
                }
                if let errorMessage {
                    Text(errorMessage).foregroundStyle(OpaleTheme.loss)
                }
            }
            .navigationTitle(object.asset.name)
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("Enregistrer") { Task { await save() } }
                }
                ToolbarItem(placement: .cancellationAction) {
                    Button("Annuler") { dismiss() }
                }
            }
            .onAppear(perform: prefill)
        }
    }

    private func prefill() {
        category = object.details.category
        brand = object.details.brand
        if object.details.purchasePrice.raw > 0 {
            purchaseText = String(object.details.purchasePrice.raw / 100)
        }
        if let date = object.details.purchaseDate {
            hasDate = true
            purchaseDate = date
        }
        insured = object.details.insured
    }

    private func save() async {
        do {
            try await session.api.upsertObject(assetID: object.asset.id, .init(
                category: category.trimmingCharacters(in: .whitespaces),
                brand: brand.trimmingCharacters(in: .whitespaces),
                purchasePriceCents: purchaseText.isEmpty ? 0 : (Cents.parse(purchaseText)?.raw ?? 0),
                purchaseDate: hasDate ? purchaseDate.opaleDayString : "",
                insured: insured
            ))
            onSaved()
            dismiss()
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
