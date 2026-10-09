import SwiftUI
import UniformTypeIdentifiers

/// Coffre-fort patrimonial (EF-064) : documents chiffrés sur le homelab
/// (AES-256-GCM côté serveur — jamais transmis aux fournisseurs IA).
struct VaultView: View {
    @Environment(SessionStore.self) private var session

    @State private var documents: [VaultDocument] = []
    @State private var vaultConfigured = true
    @State private var loaded = false
    @State private var showImporter = false
    @State private var pendingImport: PendingImport?
    @State private var viewing: VaultDocument?
    @State private var editing: VaultDocument?
    @State private var deleting: VaultDocument?
    @State private var errorMessage: String?

    var body: some View {
        List {
            if !vaultConfigured {
                Section {
                    Label {
                        Text("Le coffre chiffré n’est pas disponible sur ce serveur.")
                            .font(.subheadline)
                    } icon: {
                        Image(systemName: "lock.slash")
                            .foregroundStyle(OpaleTheme.loss)
                    }
                }
            } else if loaded && documents.isEmpty {
                ContentUnavailableView(
                    "Coffre vide",
                    systemImage: "lock.doc",
                    description: Text("Actes, contrats, factures, attestations… stockés chiffrés sur ton homelab.")
                )
            }

            if !documents.isEmpty {
                Section {
                    ForEach(documents) { doc in
                        Button {
                            viewing = doc
                        } label: {
                            row(doc)
                        }
                        .buttonStyle(.plain)
                        .contextMenu { Button("Modifier les informations") { editing = doc } }
                    }
                    .onDelete { indexSet in if let index = indexSet.first { deleting = documents[index] } }
                } footer: {
                    Label("Chiffré AES-256 sur ton homelab — jamais envoyé au cloud (N3).",
                          systemImage: "lock.shield")
                        .font(.caption2)
                }
            }

            if let errorMessage {
                Text(errorMessage).foregroundStyle(OpaleTheme.loss)
            }
        }
        .opaleList()
        .navigationTitle("Coffre-fort")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    showImporter = true
                } label: {
                    Label("Ajouter", systemImage: "plus")
                }
                .disabled(!vaultConfigured)
            }
        }
        .confirmationDialog("Supprimer ce document ?", isPresented: Binding(get: { deleting != nil }, set: { if !$0 { deleting = nil } }), titleVisibility: .visible) {
            Button("Supprimer définitivement", role: .destructive) { if let doc = deleting { Task { do { try await session.api.deleteDocument(id: doc.id); await load() } catch { errorMessage = error.localizedDescription }; deleting = nil } } }
        }
        .task { await load() }
        .refreshable { await load() }
        .fileImporter(isPresented: $showImporter,
                      allowedContentTypes: [.item],
                      allowsMultipleSelection: false) { result in
            handleImport(result)
        }
        .sheet(item: $pendingImport) { pending in
            DocumentMetaSheet(pending: pending) {
                Task { await load() }
            }
            .presentationDetents([.medium])
        }
        .sheet(item: $editing) { doc in DocumentEditSheet(document: doc) { Task { await load() } } }
        .sheet(item: $viewing) { doc in
            DocumentDetailSheet(document: doc)
                .presentationDetents([.medium])
        }
    }

    @ViewBuilder
    private func row(_ d: VaultDocument) -> some View {
        HStack {
            Image(systemName: DocumentKind(rawValue: d.kind)?.systemImage ?? "doc")
                .font(.title3)
                .foregroundStyle(OpaleTheme.accent)
                .frame(width: 32)
            VStack(alignment: .leading, spacing: 2) {
                Text(d.name).font(.body.weight(.medium)).lineLimit(1)
                HStack(spacing: 4) {
                    Text(DocumentKind(rawValue: d.kind)?.label ?? d.kind)
                    if let assetName = d.assetName, !assetName.isEmpty {
                        Text("· \(assetName)")
                    }
                }
                .font(.caption)
                .foregroundStyle(.secondary)
            }
            Spacer()
            Text(ByteCountFormatStyle().format(d.sizeBytes))
                .font(.caption)
                .foregroundStyle(.tertiary)
        }
    }

    private func handleImport(_ result: Result<[URL], Error>) {
        guard case .success(let urls) = result, let url = urls.first else { return }
        guard url.startAccessingSecurityScopedResource() else {
            errorMessage = "Accès au fichier refusé"
            return
        }
        defer { url.stopAccessingSecurityScopedResource() }
        do {
            let data = try Data(contentsOf: url)
            guard data.count <= 10 << 20 else {
                errorMessage = "Fichier trop volumineux (10 Mio max)"
                return
            }
            pendingImport = PendingImport(
                fileName: url.lastPathComponent,
                mime: UTType(filenameExtension: url.pathExtension)?.preferredMIMEType
                    ?? "application/octet-stream",
                data: data
            )
            errorMessage = nil
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func load() async {
        do {
            let result = try await session.api.documents()
            documents = result.items
            vaultConfigured = result.vaultConfigured
            errorMessage = nil
        } catch { errorMessage = error.localizedDescription }
        loaded = true
    }
}

// MARK: - Types de documents

/// Types de documents du coffre, libellés français.
enum DocumentKind: String, CaseIterable, Identifiable {
    case deed, contract, invoice, identity, insurance, tax, other
    var id: String { rawValue }
    var label: String {
        switch self {
        case .deed: "Acte"
        case .contract: "Contrat"
        case .invoice: "Facture"
        case .identity: "Identité"
        case .insurance: "Assurance"
        case .tax: "Impôts"
        case .other: "Autre"
        }
    }
    var systemImage: String {
        switch self {
        case .deed: "text.book.closed"
        case .contract: "signature"
        case .invoice: "receipt"
        case .identity: "person.text.rectangle"
        case .insurance: "shield.lefthalf.filled"
        case .tax: "building.columns"
        case .other: "doc"
        }
    }
}

/// Un fichier prêt à être déposé dans le coffre.
struct PendingImport: Identifiable {
    let id = UUID()
    let fileName: String
    let mime: String
    let data: Data
}

/// Métadonnées avant dépôt : type de document et actif lié.
private struct DocumentMetaSheet: View {
    let pending: PendingImport
    var onSaved: () -> Void

    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss

    @State private var name = ""
    @State private var kind: DocumentKind = .other
    @State private var assetID = ""
    @State private var assets: [Asset] = []
    @State private var isUploading = false
    @State private var errorMessage: String?

    var body: some View {
        NavigationStack {
            Form {
                Section("Document") {
                    TextField("Nom", text: $name)
                    Picker("Type", selection: $kind) {
                        ForEach(DocumentKind.allCases) { k in
                            Label(k.label, systemImage: k.systemImage).tag(k)
                        }
                    }
                }
                Section("Rattacher à un actif (optionnel)") {
                    Picker("Actif", selection: $assetID) {
                        Text("Aucun").tag("")
                        ForEach(assets) { a in
                            Text(a.name).tag(a.id)
                        }
                    }
                }
                if let errorMessage {
                    Text(errorMessage).foregroundStyle(OpaleTheme.loss)
                }
            }
            .navigationTitle("Déposer au coffre")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("Déposer") { Task { await upload() } }
                        .disabled(name.isEmpty || isUploading)
                }
                ToolbarItem(placement: .cancellationAction) {
                    Button("Annuler") { dismiss() }
                }
            }
            .task {
                name = pending.fileName
                assets = (try? await session.api.listAssets()) ?? []
            }
        }
    }

    private func upload() async {
        isUploading = true
        defer { isUploading = false }
        do {
            _ = try await session.api.createDocument(.init(
                name: name,
                kind: kind.rawValue,
                mime: pending.mime,
                assetID: assetID,
                contentBase64: pending.data.base64EncodedString()
            ))
            onSaved()
            dismiss()
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}

/// Détail d'un document : téléchargement déchiffré + partage.
private struct DocumentDetailSheet: View {
    let document: VaultDocument

    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss

    @State private var fileURL: URL?
    @State private var errorMessage: String?

    var body: some View {
        NavigationStack {
            List {
                Section {
                    LabeledContent("Type", value: DocumentKind(rawValue: document.kind)?.label ?? document.kind)
                    LabeledContent("Taille", value: ByteCountFormatStyle().format(document.sizeBytes))
                    LabeledContent("Déposé le", value: document.createdAt.opaleFormatted(.dateTime.day().month().year()))
                    if let assetName = document.assetName, !assetName.isEmpty {
                        LabeledContent("Actif lié", value: assetName)
                    }
                }
                Section {
                    if let fileURL {
                        ShareLink(item: fileURL) {
                            Label("Partager / ouvrir", systemImage: "square.and.arrow.up")
                        }
                    } else if let errorMessage {
                        Text(errorMessage).foregroundStyle(OpaleTheme.loss)
                    } else {
                        HStack {
                            ProgressView()
                            Text("Déchiffrement…")
                                .font(.subheadline)
                                .foregroundStyle(.secondary)
                        }
                    }
                } footer: {
                    Label("Déchiffré à la demande depuis ton homelab.", systemImage: "lock.open")
                        .font(.caption2)
                }
            }
            .navigationTitle(document.name)
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Fermer") { dismiss() }
                }
            }
            .task { await download() }
            .onDisappear { if let fileURL { try? FileManager.default.removeItem(at: fileURL.deletingLastPathComponent()) }; fileURL = nil }
        }
    }

    private func download() async {
        do {
            let data = try await session.api.documentContent(id: document.id)
            guard !Task.isCancelled else { return }
            let directory = FileManager.default.temporaryDirectory.appendingPathComponent("opale-document-" + UUID().uuidString, isDirectory: true)
            try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
            let filename = URL(fileURLWithPath: document.name).lastPathComponent
            let url = directory.appendingPathComponent(filename.isEmpty ? "document" : filename)
            try data.write(to: url, options: [.atomic, .completeFileProtection])
            fileURL = url
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}


private struct DocumentEditSheet: View {
    let document: VaultDocument
    var saved: () -> Void
    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss
    @State private var name = ""
    @State private var kind = "other"
    @State private var assetID = ""
    @State private var assets: [Asset] = []
    @State private var error: String?
    @State private var busy = false
    var body: some View {
        NavigationStack { Form {
            Section("Document") {
                TextField("Nom", text: $name)
                Picker("Type", selection: $kind) { ForEach(DocumentKind.allCases) { Text($0.label).tag($0.rawValue) } }
                Picker("Actif lié", selection: $assetID) { Text("Aucun").tag(""); ForEach(assets) { Text($0.name).tag($0.id) } }
            }
            ToolError(message: error)
        }.navigationTitle("Modifier le document").toolbar {
            ToolbarItem(placement: .cancellationAction) { Button("Annuler") { dismiss() } }
            ToolbarItem(placement: .confirmationAction) { Button("Enregistrer") { Task { await save() } }.disabled(busy || name.isEmpty) }
        }.task { name = document.name; kind = document.kind; assetID = document.assetID ?? ""; do { assets = try await session.api.listAssets() } catch { self.error = error.localizedDescription } } }
    }
    private func save() async { busy = true; defer { busy = false }; do { try await session.api.updateDocument(id: document.id, name: name, kind: kind, assetID: assetID); saved(); dismiss() } catch { self.error = error.localizedDescription } }
}
