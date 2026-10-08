import SwiftUI

struct Beneficiary: Codable, Identifiable {
    var id: String
    var contact_id: String
    var document_id: String
    var share_bps: Int
    var note: String
}
struct EmergencyGrant: Codable, Identifiable {
    var id: String
    var owner_profile_id: String
    var recipient_profile_id: String
    var asset_ids: [String]
    var document_ids: [String]
    var active: Bool
    var expires_at: Date
}
struct TransmissionAccessView: View {
    @Environment(SessionStore.self) private var session
    @State private var beneficiaries: [Beneficiary] = []
    @State private var grants: [EmergencyGrant] = []
    @State private var contacts: [Contact] = []
    @State private var documents: [VaultDocument] = []
    @State private var profiles: [Profile] = []
    @State private var contactID = ""
    @State private var documentID = ""
    @State private var share = "100"
    @State private var note = ""
    @State private var error: String?
    @State private var busy = false
    @State private var showGrant = false
    var body: some View {
        List {
            Section("Bénéficiaire d'un contrat") {
                Picker("Contact", selection: $contactID) { Text("Choisir").tag(""); ForEach(contacts) { Text($0.name).tag($0.id) } }
                Picker("Contrat au coffre", selection: $documentID) { Text("Choisir").tag(""); ForEach(documents) { Text($0.name).tag($0.id) } }
                TextField("Part en % (ex. 33,33)", text: $share).keyboardType(.decimalPad)
                TextField("Note", text: $note)
                Button("Enregistrer la désignation") { Task { await saveBeneficiary() } }.disabled(busy || contactID.isEmpty || documentID.isEmpty)
                Text("Cet inventaire ne modifie pas les clauses contractuelles et ne garantit pas leur validité juridique. Un bénéficiaire ne reçoit aucun accès à l'app.").font(.caption)
            }
            Section("Désignations enregistrées") {
                ForEach(beneficiaries) { item in
                    Button { contactID = item.contact_id; documentID = item.document_id; share = MoneyFormat.input(Cents(Int64(item.share_bps)), currency: "EUR"); note = item.note } label: {
                        VStack(alignment: .leading) {
                            Text(contacts.first { $0.id == item.contact_id }?.name ?? "Contact")
                            Text("\(documents.first { $0.id == item.document_id }?.name ?? "Contrat") · \((Decimal(item.share_bps) / 100).formatted(.number.precision(.fractionLength(0...2)).locale(Locale(identifier: "fr_FR")))) %").font(.caption)
                        }
                    }
                }.onDelete { indices in Task { do { for i in indices { let _: APIClient.EmptyResponse = try await session.api.request("DELETE", "/v1/beneficiaries/\(beneficiaries[i].id)") }; await load() } catch { self.error = error.localizedDescription } } }
            }
            Section("Accès d'urgence accordés") {
                Button("Désigner un proche et son périmètre") { showGrant = true }
                ForEach(grants.filter { $0.owner_profile_id == session.profileID }) { grant in
                    VStack(alignment: .leading) {
                        Text(profiles.first { $0.id == grant.recipient_profile_id }?.name ?? "Proche désigné").font(.headline)
                        Text("\(grant.asset_ids.count) actif(s), \(grant.document_ids.count) document(s) · jusqu'au \(grant.expires_at.formatted(date: .abbreviated, time: .omitted))").font(.caption)
                        Text(grant.active ? "Activé" : "Inactif").foregroundStyle(grant.active ? .orange : .secondary)
                        Button(grant.active ? "Révoquer" : "Activer l'accès") { Task { await activate(grant, active: !grant.active) } }
                    }
                }
            }
            Section("Accès reçus") {
                ForEach(grants.filter { $0.recipient_profile_id == session.profileID }) { grant in
                    NavigationLink { EmergencyReadView(grant: grant) } label: { Label("Dossier d'urgence · \(grant.active ? "actif" : "inactif")", systemImage: "lock.doc") }
                }
            }
            ToolError(message: error)
        }.navigationTitle("Transmission & accès").task { await load() }
        .sheet(isPresented: $showGrant) { EmergencyGrantSheet { Task { await load() } } }
    }
    private func load() async {
        struct Beneficiaries: Decodable { let beneficiaries: [Beneficiary] }
        struct Grants: Decodable { let grants: [EmergencyGrant] }
        do {
            contacts = try await session.api.contacts(); documents = try await session.api.documents().items; profiles = try await session.api.listProfiles()
            let b: Beneficiaries = try await session.api.request("GET", "/v1/beneficiaries"); beneficiaries = b.beneficiaries
            let g: Grants = try await session.api.request("GET", "/v1/emergency-grants"); grants = g.grants; error = nil
        } catch { self.error = error.localizedDescription }
    }
    private func saveBeneficiary() async { guard let value = Cents.parse(share), value.raw > 0, value.raw <= 10000 else { error = "Part valide entre 0,01 et 100 % requise"; return }; busy = true; defer { busy = false }; do { let _: APIClient.EmptyResponse = try await session.api.request("PUT", "/v1/beneficiaries", body: Beneficiary(id: "", contact_id: contactID, document_id: documentID, share_bps: Int(value.raw), note: note)); await load() } catch { self.error = error.localizedDescription } }
    private func activate(_ grant: EmergencyGrant, active: Bool) async { struct Body: Encodable { let active: Bool }; do { let _: APIClient.EmptyResponse = try await session.api.request("PATCH", "/v1/emergency-grants/\(grant.id)", body: Body(active: active)); await load() } catch { self.error = error.localizedDescription } }
}
private struct EmergencyGrantSheet: View {
    var saved: () -> Void
    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss
    @State private var recipient = ""
    @State private var profiles: [Profile] = []
    @State private var assets: [Asset] = []
    @State private var documents: [VaultDocument] = []
    @State private var assetIDs: Set<String> = []
    @State private var documentIDs: Set<String> = []
    @State private var expiration = Calendar.opale.date(byAdding: .month, value: 3, to: .now)!
    @State private var error: String?
    @State private var busy = false
    var body: some View {
        NavigationStack { Form {
            Section("Destinataire") { Picker("Profil du proche", selection: $recipient) { Text("Choisir").tag(""); ForEach(profiles.filter { $0.id != session.profileID }) { Text($0.name).tag($0.id) } }; DatePicker("Expiration", selection: $expiration, in: Date.now...Calendar.opale.date(byAdding: .year, value: 1, to: .now)!, displayedComponents: .date) }
            Section("Actifs consultables") { ForEach(assets) { item in Toggle(item.name, isOn: selected(item.id, in: $assetIDs)) } }
            Section("Documents téléchargeables") { ForEach(documents) { item in Toggle(item.name, isOn: selected(item.id, in: $documentIDs)) } }
            Text("Le droit est créé inactif. Tu choisis ensuite de l'activer. Le proche ne peut ni modifier les données ni consulter les autres ressources.").font(.caption)
            ToolError(message: error)
        }.navigationTitle("Accès d'urgence").toolbar {
            ToolbarItem(placement: .cancellationAction) { Button("Annuler") { dismiss() } }
            ToolbarItem(placement: .confirmationAction) { Button("Créer") { Task { await save() } }.disabled(busy || recipient.isEmpty || (assetIDs.isEmpty && documentIDs.isEmpty)) }
        }.task { do { profiles = try await session.api.listProfiles(); assets = try await session.api.listAssets(); documents = try await session.api.documents().items } catch { self.error = error.localizedDescription } } }
    }
    private func selected(_ id: String, in values: Binding<Set<String>>) -> Binding<Bool> { Binding(get: { values.wrappedValue.contains(id) }, set: { if $0 { values.wrappedValue.insert(id) } else { values.wrappedValue.remove(id) } }) }
    private func save() async {
        struct Body: Encodable { let recipient_profile_id: String; let asset_ids: [String]; let document_ids: [String]; let expires_at: String }
        busy = true; defer { busy = false }
        do { let _: APIClient.EmptyResponse = try await session.api.request("POST", "/v1/emergency-grants", body: Body(recipient_profile_id: recipient, asset_ids: Array(assetIDs), document_ids: Array(documentIDs), expires_at: expiration.ISO8601Format())); saved(); dismiss() } catch { self.error = error.localizedDescription }
    }
}
private struct EmergencyReadView: View {
    let grant: EmergencyGrant
    @Environment(SessionStore.self) private var session
    @State private var assets: [Asset] = []
    @State private var documents: [VaultDocument] = []
    @State private var error: String?
    @State private var file: URL?
    var body: some View {
        List {
            Section("Actifs autorisés") { ForEach(assets) { item in LabeledContent(item.name) { Group { if let value = item.currentValue ?? item.latestValue { AmountText(cents: value, currency: item.currency) } else { Text("Non renseignée") } } } } }
            Section("Documents autorisés") { ForEach(documents) { item in Button(item.name) { Task { await download(item) } } } }
            ToolError(message: error)
        }.navigationTitle("Dossier d'urgence").task { await load() }.refreshable { await load() }
        .sheet(item: $file) { url in NavigationStack { VStack(spacing: 20) { Text("Document déchiffré").font(.headline); Text("La copie partagée sort du coffre. Conserve-la dans un emplacement approprié."); ShareLink("Enregistrer / partager", item: url) }.padding() } }
        .onDisappear { if let file { try? FileManager.default.removeItem(at: file) }; assets = []; documents = [] }
    }
    private func load() async { struct Result: Decodable { let assets: [Asset]; let documents: [VaultDocument] }; do { let r: Result = try await session.api.request("GET", "/v1/emergency/\(grant.id)"); assets = r.assets; documents = r.documents; error = nil } catch { assets = []; documents = []; self.error = error.localizedDescription } }
    private func download(_ document: VaultDocument) async { do { let data = try await session.api.documentContent(id: document.id, grantID: grant.id); let url = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString + "-" + URL(fileURLWithPath: document.name).lastPathComponent); try data.write(to: url, options: [.atomic, .completeFileProtection]); file = url; error = nil } catch { self.error = error.localizedDescription; await load() } }
}
