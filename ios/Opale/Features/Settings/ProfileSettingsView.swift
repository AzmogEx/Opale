import SwiftUI

struct ProfileSettingsView: View {
    @Environment(SessionStore.self) private var session
    @State private var name = ""
    @State private var pin = ""
    @State private var confirmation = ""
    @State private var privacy = "N1"
    @State private var error: String?
    @State private var feedback = ""
    @State private var busy = false
    var body: some View {
        Form {
            Section("Profil") { TextField("Nom", text: $name) }
            Section("Nouveau code PIN") {
                SecureField("Nouveau code (laisser vide pour conserver)", text: $pin).keyboardType(.numberPad)
                SecureField("Confirmer le code", text: $confirmation).keyboardType(.numberPad)
            }
            Section("Confidentialité de l'IA") {
                Picker("Traitement autorisé", selection: $privacy) { Text("Appareil et homelab uniquement").tag("N1"); Text("Cloud possible, avec consentement").tag("N2"); Text("Cloud interdit").tag("N3") }
                Text("L'autorisation du profil ne transmet rien à elle seule. Chaque demande cloud nécessite aussi ton accord. Seuls l'intention structurée et les agrégats autorisés peuvent sortir.").font(.caption)
            }
            Section { Button("Enregistrer") { Task { await save() } }.disabled(busy || name.isEmpty || (!pin.isEmpty && (pin.count < 4 || pin != confirmation || !pin.allSatisfy(\.isNumber)))) }
            if !feedback.isEmpty { Text(feedback) }
            ToolError(message: error)
        }.navigationTitle("Mon profil").task {
            if case .loggedIn(let profile) = session.state { name = profile.name; privacy = profile.privacyDefault }
        }
    }
    private func save() async {
        struct Body: Encodable { let name: String; let pin: String?; let privacy_default: String }
        busy = true; defer { busy = false }
        do { let _: APIClient.EmptyResponse = try await session.api.request("PATCH", "/v1/me", body: Body(name: name, pin: pin.isEmpty ? nil : pin, privacy_default: privacy)); pin = ""; confirmation = ""; await session.revalidate(); feedback = "Profil enregistré"; error = nil }
        catch { self.error = error.localizedDescription }
    }
}
