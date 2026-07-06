import SwiftUI

/// Espace partagé (EF-007) — segment « Commun » de l'onglet Flux.
/// Les dépenses marquées « communes » sont mises au pot : chacun doit une
/// quote-part égale, la balance dit qui doit quoi à qui.
struct SharedSpaceView: View {
    @Environment(SessionStore.self) private var session

    @State private var spaces: [Space] = []
    @State private var detail: SpaceDetail?
    @State private var newSpaceName = ""
    @State private var showAddMember = false
    @State private var loaded = false
    @State private var errorMessage: String?

    private var space: Space? { spaces.first }

    var body: some View {
        List {
            if let space {
                if let detail {
                    balanceSection(detail)
                    membersSection(space, detail)
                    transactionsSection(detail)
                } else {
                    ProgressView()
                }
            } else if loaded {
                createSection
            } else {
                ProgressView()
            }
            if let errorMessage {
                Text(errorMessage).foregroundStyle(OpaleTheme.loss)
            }
        }
        .opaleList()
        .task { await load() }
        .refreshable { await load() }
        .sheet(isPresented: $showAddMember) {
            if let space {
                AddMemberSheet(space: space) { Task { await load() } }
                    .presentationDetents([.medium])
            }
        }
    }

    // MARK: - Création du premier espace

    private var createSection: some View {
        Section {
            VStack(alignment: .leading, spacing: 10) {
                Label("Dépenses communes du foyer", systemImage: "person.2.fill")
                    .font(.headline)
                Text("Crée un espace, ajoute les profils du foyer, puis marque des dépenses « communes » depuis leur fiche : la balance se calcule toute seule.")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
                TextField("Nom de l'espace (ex. Foyer)", text: $newSpaceName)
                    .textFieldStyle(.roundedBorder)
                Button {
                    Task { await createSpace() }
                } label: {
                    Label("Créer l'espace", systemImage: "plus.circle.fill")
                }
                .buttonStyle(.glassProminent)
                .disabled(newSpaceName.trimmingCharacters(in: .whitespaces).isEmpty)
            }
            .padding(.vertical, 6)
        }
    }

    // MARK: - Balance

    @ViewBuilder
    private func balanceSection(_ d: SpaceDetail) -> some View {
        Section {
            HStack {
                Text("Total des dépenses communes")
                    .font(.subheadline)
                Spacer()
                AmountText(cents: d.total, style: .whole)
                    .font(.headline)
            }
            ForEach(d.members) { m in
                HStack {
                    VStack(alignment: .leading, spacing: 2) {
                        Text(m.name).font(.body.weight(.medium))
                        Text("payé \(MoneyFormat.eurosWhole(m.paid)) · quote-part \(MoneyFormat.eurosWhole(m.share))")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                    Spacer()
                    AmountText(cents: m.balance, style: .signedDelta)
                        .font(.callout.weight(.bold))
                        .foregroundStyle(m.balance.raw < 0 ? OpaleTheme.loss : OpaleTheme.gain)
                }
            }
        } header: {
            Text("Balance")
        } footer: {
            Text("Positif : le pot lui doit. Négatif : il doit au pot.")
        }
    }

    // MARK: - Membres

    @ViewBuilder
    private func membersSection(_ space: Space, _ d: SpaceDetail) -> some View {
        Section {
            ForEach(space.members) { m in
                Label(m.name, systemImage: "person.circle")
            }
            Button {
                showAddMember = true
            } label: {
                Label("Ajouter un membre du foyer", systemImage: "person.badge.plus")
            }
        } header: {
            Text("Membres — \(space.name)")
        }
    }

    // MARK: - Dépenses communes

    @ViewBuilder
    private func transactionsSection(_ d: SpaceDetail) -> some View {
        Section {
            if (d.transactions ?? []).isEmpty {
                Text("Aucune dépense commune pour l'instant. Ouvre un mouvement dans « Mouvements » et active « Dépense commune ».")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
            }
            ForEach(d.transactions ?? []) { t in
                HStack {
                    VStack(alignment: .leading, spacing: 2) {
                        Text(t.label).font(.subheadline.weight(.medium))
                        Text("\(t.payerName) · \(t.occurredOn.formatted(.dateTime.day().month(.abbreviated)))")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                    Spacer()
                    AmountText(cents: t.amount, style: .whole)
                        .font(.callout.weight(.semibold))
                }
            }
        } header: {
            Text("Dépenses communes")
        }
    }

    // MARK: - Actions

    private func load() async {
        spaces = (try? await session.api.spaces()) ?? []
        if let space {
            detail = try? await session.api.spaceDetail(id: space.id)
        }
        loaded = true
    }

    private func createSpace() async {
        do {
            _ = try await session.api.createSpace(
                name: newSpaceName.trimmingCharacters(in: .whitespaces))
            newSpaceName = ""
            await load()
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}

/// Ajout d'un profil du foyer à l'espace.
private struct AddMemberSheet: View {
    let space: Space
    var onAdded: () -> Void

    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss

    @State private var profiles: [Profile] = []
    @State private var errorMessage: String?

    private var candidates: [Profile] {
        let memberIDs = Set(space.members.map(\.profileID))
        return profiles.filter { !memberIDs.contains($0.id) }
    }

    var body: some View {
        NavigationStack {
            List {
                if candidates.isEmpty {
                    Text("Tous les profils du foyer sont déjà membres.")
                        .foregroundStyle(.secondary)
                }
                ForEach(candidates) { p in
                    Button {
                        Task { await add(p) }
                    } label: {
                        Label(p.name, systemImage: "person.badge.plus")
                    }
                }
                if let errorMessage {
                    Text(errorMessage).foregroundStyle(OpaleTheme.loss)
                }
            }
            .navigationTitle("Ajouter un membre")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Annuler") { dismiss() }
                }
            }
            .task {
                profiles = (try? await session.api.listProfiles()) ?? []
            }
        }
    }

    private func add(_ profile: Profile) async {
        do {
            try await session.api.addSpaceMember(spaceID: space.id, profileID: profile.id)
            onAdded()
            dismiss()
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
