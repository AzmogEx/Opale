import SwiftUI

/// Plan de transmission (EF-063) : le dossier « si un jour » — qui contacter,
/// ce qui existe, où sont les documents.
struct TransmissionView: View {
    @Environment(SessionStore.self) private var session

    @State private var summary: TransmissionSummary?
    @State private var showAddContact = false

    var body: some View {
        List {
            if let summary {
                Section {
                    VStack(alignment: .leading, spacing: 6) {
                        Text("Si un jour tes proches doivent reprendre la main, tout est ici : le patrimoine, les contacts clés et les documents du coffre.")
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                        HStack(spacing: 24) {
                            VStack(alignment: .leading, spacing: 2) {
                                Text("Patrimoine net")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                                AmountText(cents: summary.netWorth.net, style: .whole)
                                    .font(.headline)
                            }
                            VStack(alignment: .leading, spacing: 2) {
                                Text("Documents au coffre")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                                Text("\(summary.documentCount)")
                                    .font(.headline)
                            }
                        }
                    }
                    .padding(.vertical, 4)
                }

                Section {
                    if summary.contacts.isEmpty {
                        Text("Ajoute les personnes à contacter : notaire, banquier, proche de confiance…")
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                    }
                    ForEach(summary.contacts) { contact in
                        contactRow(contact)
                    }
                    .onDelete { indexSet in
                        Task {
                            for i in indexSet {
                                try? await session.api.deleteContact(id: summary.contacts[i].id)
                            }
                            await load()
                        }
                    }
                } header: {
                    HStack {
                        Text("Contacts clés")
                        Spacer()
                        Button {
                            showAddContact = true
                        } label: {
                            Image(systemName: "plus.circle.fill")
                        }
                    }
                }

                Section("Le patrimoine en un coup d'œil") {
                    ForEach(summary.assets) { asset in
                        HStack {
                            Image(systemName: asset.kind.systemImage)
                                .foregroundStyle(OpaleTheme.accent)
                                .frame(width: 28)
                            Text(asset.name).font(.subheadline)
                            if asset.documentCount > 0 {
                                Label("\(asset.documentCount)", systemImage: "lock.doc")
                                    .font(.caption2)
                                    .foregroundStyle(.secondary)
                                    .labelStyle(.titleAndIcon)
                            }
                            Spacer()
                            if let value = asset.latestValue {
                                AmountText(cents: value, style: .whole)
                                    .font(.subheadline.weight(.medium))
                            }
                        }
                    }
                    ForEach(summary.liabilities) { liability in
                        HStack {
                            Image(systemName: liability.kind.systemImage)
                                .foregroundStyle(OpaleTheme.loss)
                                .frame(width: 28)
                            Text(liability.name).font(.subheadline)
                            Spacer()
                            if let value = liability.latestValue {
                                AmountText(cents: Cents(-value.raw), style: .whole)
                                    .font(.subheadline.weight(.medium))
                                    .foregroundStyle(OpaleTheme.loss)
                            }
                        }
                    }
                }
            } else {
                ProgressView()
            }
        }
        .navigationTitle("Transmission")
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
        .refreshable { await load() }
        .sheet(isPresented: $showAddContact) {
            ContactFormSheet { Task { await load() } }
                .presentationDetents([.medium])
        }
    }

    @ViewBuilder
    private func contactRow(_ c: Contact) -> some View {
        let role = ContactRole(rawValue: c.role) ?? .other
        HStack(alignment: .top) {
            Image(systemName: role.systemImage)
                .font(.title3)
                .foregroundStyle(OpaleTheme.accent)
                .frame(width: 32)
            VStack(alignment: .leading, spacing: 2) {
                Text(c.name).font(.body.weight(.medium))
                Text(role.label).font(.caption).foregroundStyle(.secondary)
                if !c.note.isEmpty {
                    Text(c.note).font(.caption).foregroundStyle(.tertiary)
                }
            }
            Spacer()
            VStack(alignment: .trailing, spacing: 4) {
                if !c.phone.isEmpty, let url = URL(string: "tel:\(c.phone.filter { !$0.isWhitespace })") {
                    Link(destination: url) {
                        Image(systemName: "phone.circle.fill").font(.title3)
                    }
                }
                if !c.email.isEmpty, let url = URL(string: "mailto:\(c.email)") {
                    Link(destination: url) {
                        Image(systemName: "envelope.circle.fill").font(.title3)
                    }
                }
            }
        }
    }

    private func load() async {
        summary = try? await session.api.transmission()
    }
}

/// Ajout d'un contact clé.
private struct ContactFormSheet: View {
    var onSaved: () -> Void

    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss

    @State private var name = ""
    @State private var role: ContactRole = .notary
    @State private var phone = ""
    @State private var email = ""
    @State private var note = ""
    @State private var errorMessage: String?

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    TextField("Nom", text: $name)
                    Picker("Rôle", selection: $role) {
                        ForEach(ContactRole.allCases) { r in
                            Label(r.label, systemImage: r.systemImage).tag(r)
                        }
                    }
                }
                Section("Coordonnées") {
                    TextField("Téléphone", text: $phone)
                        .keyboardType(.phonePad)
                    TextField("E-mail", text: $email)
                        .keyboardType(.emailAddress)
                        .textInputAutocapitalization(.never)
                }
                Section("Note") {
                    TextField("Ex. détient le testament, n° de contrat…", text: $note, axis: .vertical)
                        .lineLimit(2...4)
                }
                if let errorMessage {
                    Text(errorMessage).foregroundStyle(OpaleTheme.loss)
                }
            }
            .navigationTitle("Nouveau contact")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("Ajouter") { Task { await save() } }
                        .disabled(name.trimmingCharacters(in: .whitespaces).isEmpty)
                }
                ToolbarItem(placement: .cancellationAction) {
                    Button("Annuler") { dismiss() }
                }
            }
        }
    }

    private func save() async {
        do {
            _ = try await session.api.createContact(.init(
                name: name.trimmingCharacters(in: .whitespaces),
                role: role.rawValue,
                phone: phone,
                email: email,
                note: note
            ))
            onSaved()
            dismiss()
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
