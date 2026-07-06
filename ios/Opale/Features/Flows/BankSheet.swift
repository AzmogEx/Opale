import SwiftUI

/// Synchro bancaire GoCardless (EF-071) : connexion DSP2 chez SA banque
/// (Opale ne voit jamais les identifiants), puis synchro des mouvements —
/// dédupliqués et catégorisés comme un import CSV.
struct BankSheet: View {
    var onSynced: () -> Void

    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss
    @Environment(\.openURL) private var openURL

    @State private var status: BankStatus?
    @State private var syncResults: [BankSyncResult] = []
    @State private var isSyncing = false
    @State private var showConnect = false
    @State private var errorMessage: String?

    var body: some View {
        NavigationStack {
            List {
                if let status {
                    if !status.configured {
                        notConfiguredSection
                    } else {
                        linksSection(status.links ?? [])
                        if !syncResults.isEmpty {
                            resultsSection
                        }
                    }
                } else {
                    HStack {
                        ProgressView()
                        Text("État de la synchro…")
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                    }
                }
                if let errorMessage {
                    Text(errorMessage).foregroundStyle(OpaleTheme.loss)
                }
            }
            .navigationTitle("Ma banque")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Fermer") { dismiss() }
                }
            }
            .task { await load() }
            .sheet(isPresented: $showConnect) {
                BankConnectSheet {
                    Task { await load() }
                }
            }
        }
    }

    private var notConfiguredSection: some View {
        Section {
            Label {
                VStack(alignment: .leading, spacing: 6) {
                    Text("Synchro bancaire non configurée")
                        .font(.subheadline.weight(.semibold))
                    Text("Crée un compte GoCardless Bank Account Data (gratuit, DSP2) et définis OPALE_GC_SECRET_ID / OPALE_GC_SECRET_KEY sur le serveur. En attendant, l'import CSV fait très bien le travail.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            } icon: {
                Image(systemName: "building.columns")
                    .foregroundStyle(.secondary)
            }
        }
    }

    @ViewBuilder
    private func linksSection(_ links: [BankLink]) -> some View {
        Section {
            if links.isEmpty {
                Text("Aucune banque connectée. Tes identifiants ne passent jamais par Opale : tu t'authentifies chez ta banque (DSP2).")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
            }
            ForEach(links) { link in
                HStack {
                    Image(systemName: "building.columns.fill")
                        .foregroundStyle(OpaleTheme.accent)
                        .frame(width: 30)
                    VStack(alignment: .leading, spacing: 2) {
                        Text(link.institutionName.isEmpty ? "Banque" : link.institutionName)
                            .font(.body.weight(.medium))
                        HStack(spacing: 4) {
                            Text("→ \(link.assetName ?? "compte")")
                            if let sync = link.lastSyncedAt {
                                Text("· sync \(sync.formatted(.relative(presentation: .named)))")
                            } else {
                                Text("· jamais synchronisée")
                            }
                        }
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    }
                    Spacer()
                    if link.status == "created" {
                        Text("En attente")
                            .font(.caption2.weight(.semibold))
                            .padding(.horizontal, 8)
                            .padding(.vertical, 3)
                            .background(Color.orange.opacity(0.15), in: .capsule)
                            .foregroundStyle(.orange)
                    }
                }
            }
            .onDelete { indexSet in
                Task {
                    for i in indexSet {
                        try? await session.api.bankDisconnect(id: links[i].id)
                    }
                    await load()
                }
            }

            Button {
                showConnect = true
            } label: {
                Label("Connecter une banque", systemImage: "plus.circle.fill")
            }

            if !links.isEmpty {
                Button {
                    Task { await sync() }
                } label: {
                    if isSyncing {
                        HStack {
                            ProgressView()
                            Text("Synchronisation…")
                        }
                    } else {
                        Label("Synchroniser maintenant", systemImage: "arrow.triangle.2.circlepath")
                    }
                }
                .disabled(isSyncing)
            }
        } header: {
            Text("Banques connectées")
        } footer: {
            Label("Connexion DSP2 via GoCardless — lecture seule, révocable à tout moment.",
                  systemImage: "lock.shield")
                .font(.caption2)
        }
    }

    private var resultsSection: some View {
        Section("Dernière synchro") {
            ForEach(syncResults) { r in
                HStack {
                    Image(systemName: resultIcon(r.status))
                        .foregroundStyle(resultColor(r.status))
                    VStack(alignment: .leading, spacing: 2) {
                        Text(r.institution.isEmpty ? "Banque" : r.institution)
                            .font(.subheadline.weight(.medium))
                        Text(resultLabel(r))
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }
            }
        }
    }

    private func resultIcon(_ s: String) -> String {
        switch s {
        case "synced": "checkmark.circle.fill"
        case "pending_consent": "hourglass"
        default: "xmark.circle.fill"
        }
    }

    private func resultColor(_ s: String) -> Color {
        switch s {
        case "synced": OpaleTheme.gain
        case "pending_consent": .orange
        default: OpaleTheme.loss
        }
    }

    private func resultLabel(_ r: BankSyncResult) -> String {
        switch r.status {
        case "synced": "\(r.imported) nouveau(x) mouvement(s), \(r.duplicates) déjà connu(s)"
        case "pending_consent": "Consentement pas encore donné à la banque"
        default: r.error ?? "Erreur"
        }
    }

    private func load() async {
        do {
            status = try await session.api.bankStatus()
            errorMessage = nil
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func sync() async {
        isSyncing = true
        defer { isSyncing = false }
        do {
            syncResults = try await session.api.bankSync()
            errorMessage = nil
            onSynced()
            await load()
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}

/// Choix de la banque + du compte Opale cible, puis ouverture du lien DSP2.
private struct BankConnectSheet: View {
    var onConnected: () -> Void

    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss
    @Environment(\.openURL) private var openURL

    @State private var institutions: [BankInstitution] = []
    @State private var search = ""
    @State private var assetID = ""
    @State private var assets: [Asset] = []
    @State private var isConnecting = false
    @State private var errorMessage: String?

    private var filtered: [BankInstitution] {
        search.isEmpty ? institutions
            : institutions.filter { $0.name.localizedCaseInsensitiveContains(search) }
    }

    var body: some View {
        NavigationStack {
            List {
                Section("Compte Opale qui recevra les mouvements") {
                    Picker("Compte", selection: $assetID) {
                        Text("Choisir…").tag("")
                        ForEach(assets.filter { $0.kind == .checking || $0.kind == .savings }) { a in
                            Text(a.name).tag(a.id)
                        }
                    }
                }
                Section("Ta banque") {
                    ForEach(filtered) { inst in
                        Button {
                            Task { await connect(inst) }
                        } label: {
                            HStack {
                                Text(inst.name)
                                Spacer()
                                if isConnecting { ProgressView() }
                            }
                        }
                        .disabled(assetID.isEmpty || isConnecting)
                    }
                }
                if let errorMessage {
                    Text(errorMessage).foregroundStyle(OpaleTheme.loss)
                }
            }
            .searchable(text: $search, prompt: "Chercher ma banque")
            .navigationTitle("Connecter une banque")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Annuler") { dismiss() }
                }
            }
            .task {
                assets = (try? await session.api.listAssets()) ?? []
                do {
                    institutions = try await session.api.bankInstitutions()
                } catch {
                    errorMessage = error.localizedDescription
                }
            }
        }
    }

    private func connect(_ inst: BankInstitution) async {
        isConnecting = true
        defer { isConnecting = false }
        do {
            let resp = try await session.api.bankConnect(
                institutionID: inst.id,
                institutionName: inst.name,
                assetID: assetID
            )
            // Le consentement se donne chez la banque (DSP2) : on ouvre le lien.
            if let url = URL(string: resp.consentLink) {
                openURL(url)
            }
            onConnected()
            dismiss()
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
