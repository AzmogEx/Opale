import SwiftUI
import PhotosUI

/// Valeur des objets (EF-035) : montres, véhicules, or, œuvres…
/// avec l'écart entre prix d'achat et valeur estimée.
struct ObjectsView: View {
    @Environment(SessionStore.self) private var session

    @State private var objects: [ObjectStatus] = []
    @State private var editing: ObjectStatus?
    @State private var loaded = false
    // Photos (EF-035) : documents « photo » du coffre, par actif.
    @State private var photoDocs: [String: VaultDocument] = [:]
    @State private var thumbnails: [String: UIImage] = [:]
    @State private var pickingFor: ObjectStatus?
    @State private var pickedItem: PhotosPickerItem?

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
                .contextMenu {
                    Button {
                        pickingFor = object
                    } label: {
                        Label(photoDocs[object.asset.id] == nil ? "Ajouter une photo" : "Changer la photo",
                              systemImage: "camera")
                    }
                }
            }
        }
        .opaleList()
        .navigationTitle("Objets")
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
        .refreshable { await load() }
        .photosPicker(isPresented: Binding(
            get: { pickingFor != nil },
            set: { if !$0 { pickingFor = nil } }
        ), selection: $pickedItem, matching: .images)
        .onChange(of: pickedItem) { _, item in
            guard let item, let target = pickingFor else { return }
            pickedItem = nil
            pickingFor = nil
            Task { await uploadPicked(item, for: target) }
        }
        .sheet(item: $editing) { object in
            ObjectFormSheet(object: object) {
                Task { await load() }
            }
            .presentationDetents([.medium, .large])
        }
    }

    @ViewBuilder
    private func row(_ o: ObjectStatus) -> some View {
        HStack(spacing: 12) {
            if let thumb = thumbnails[o.asset.id] {
                Image(uiImage: thumb)
                    .resizable()
                    .scaledToFill()
                    .frame(width: 44, height: 44)
                    .clipShape(.rect(cornerRadius: 10))
            } else {
                ZStack {
                    Circle().fill(OpaleTheme.iridescent).opacity(0.18)
                    Image(systemName: o.asset.kind.systemImage)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(OpaleTheme.iridescent)
                }
                .frame(width: 44, height: 44)
            }
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
        await loadPhotos()
    }

    /// Repère la photo de chaque objet (document « photo » lié à l'actif)
    /// et charge les vignettes manquantes — déchiffrées à la demande.
    private func loadPhotos() async {
        guard let docs = try? await session.api.documents().items else { return }
        for doc in docs where doc.kind == "photo" {
            guard let assetID = doc.assetID else { continue }
            photoDocs[assetID] = doc
            if thumbnails[assetID] == nil,
               let data = try? await session.api.documentContent(id: doc.id),
               let image = UIImage(data: data) {
                thumbnails[assetID] = image
            }
        }
    }

    /// Dépose la photo choisie au coffre (chiffrée, N3), liée à l'objet.
    private func uploadPicked(_ item: PhotosPickerItem, for object: ObjectStatus) async {
        guard let data = try? await item.loadTransferable(type: Data.self),
              let image = UIImage(data: data) else { return }
        // Recadrée/compressée : une vignette n'a pas besoin de 12 Mpx.
        let resized = image.preparingThumbnail(of: CGSize(width: 800, height: 800)) ?? image
        guard let jpeg = resized.jpegData(compressionQuality: 0.8) else { return }

        // Une seule photo par objet : l'ancienne est remplacée.
        if let old = photoDocs[object.asset.id] {
            try? await session.api.deleteDocument(id: old.id)
        }
        _ = try? await session.api.createDocument(.init(
            name: "photo-\(object.asset.name).jpg",
            kind: "photo",
            mime: "image/jpeg",
            assetID: object.asset.id,
            contentBase64: jpeg.base64EncodedString()
        ))
        thumbnails[object.asset.id] = resized
        await loadPhotos()
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
