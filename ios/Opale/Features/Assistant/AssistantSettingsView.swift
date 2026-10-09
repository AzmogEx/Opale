import SwiftUI

enum AssistantMode: Int, CaseIterable, Identifiable {
    case automatic, homelab, cloud
    var id: Int { rawValue }
    var apiValue: String { switch self { case .automatic: "auto"; case .homelab: "homelab"; case .cloud: "cloud" } }
    var title: String { switch self { case .automatic: "Automatique · privé d’abord"; case .homelab: "Mon PC · Ollama"; case .cloud: "Cloud · Claude" } }
    var detail: String { switch self {
    case .automatic: "Essaie ton PC, puis le moteur Opale. Le cloud reste une proposition qui demande ton accord."
    case .homelab: "Utilise uniquement le modèle configuré sur ton PC. S’il est hors ligne, le moteur Opale prend le relais."
    case .cloud: "Demande ton accord avant chaque envoi. Le cloud reçoit une intention reconnue et des agrégats minimisés, jamais ta question libre ni tes documents."
    } }
    static func current(_ session: SessionStore) -> Self { Self(rawValue: session.preferenceInt("assistant.mode", default: 0)) ?? .automatic }
}

/// Shared by Settings and the assistant. Status describes configuration and reachability separately.
struct AssistantSettingsView: View {
    @Environment(SessionStore.self) private var session
    var onModeChanged: () -> Void = {}
    @State private var mode: AssistantMode = .automatic
    @State private var status: AssistantStatus?
    @State private var loading = false
    @State private var error: String?
    var body: some View {
        Form {
            Section("Quel moteur pour mes questions ?") {
                Picker("Mon choix", selection: $mode) { ForEach(AssistantMode.allCases) { Text($0.title).tag($0) } }.accessibilityIdentifier("assistant.mode")
                Text(mode.detail).font(.subheadline).foregroundStyle(.secondary)
                Text("Ce choix est propre à ce profil. Choisir un fournisseur ne l’installe pas et n’autorise aucun envoi à lui seul.").font(.caption).foregroundStyle(.secondary)
            }
            Section("Mon PC Windows") {
                if let status {
                    Label(status.homelabAvailable ? "Modèle installé et serveur joignable" : status.homelabConfigured == true ? "Configuré, mais modèle ou serveur indisponible" : "À connecter", systemImage: status.homelabAvailable ? "checkmark.circle" : "server.rack")
                    if let model = status.homelabModel, !model.isEmpty { LabeledContent("Modèle demandé", value: model) }
                }
                DisclosureGroup("Connecter mon PC Windows") {
                Text("Installe Ollama sur Windows et télécharge un modèle. Relie ton PC au serveur Coolify par un réseau privé, puis renseigne l’URL et le modèle dans les variables du service API.").font(.subheadline)
                Text("Point de départ à essayer : qwen3.5:9b. Vérifie la mémoire disponible et la vitesse sur ta carte ; le PC doit rester allumé. Le port Ollama ne doit pas être publié sur Internet.").font(.caption).foregroundStyle(.secondary)
                Text("OPALE_OLLAMA_URL\nOPALE_OLLAMA_MODEL\nOPALE_OLLAMA_API_KEY (proxy, facultatif)").font(.caption.monospaced()).textSelection(.enabled)
                Link("Installation Windows officielle", destination: URL(string: "https://docs.ollama.com/windows")!)
                }
            }
            Section("Cloud facultatif") {
                if let status {
                    LabeledContent("Clé sur le serveur", value: status.cloudCredentialsConfigured == true || status.cloudConfigured ? "Renseignée" : "À renseigner")
                    LabeledContent("Activation serveur", value: status.cloudEnabled == true || status.cloudConfigured ? "Activée" : "Désactivée")
                    LabeledContent("Autorisation du profil", value: status.cloudAllowed == true ? "Possible avec accord" : "Non autorisé")
                    if let model = status.cloudModel, !model.isEmpty { LabeledContent("Modèle demandé", value: model) }
                }
                DisclosureGroup("Configurer Claude dans Coolify") {
                Text("Renseigne la clé dans les secrets Coolify du service API, jamais dans l’application. Active le cloud côté serveur puis autorise-le dans ton profil. Une clé renseignée ne confirme ni son crédit ni l’accès au modèle : cela sera vérifié à la première demande consentie.").font(.subheadline)
                Text("OPALE_ANTHROPIC_API_KEY\nOPALE_ANTHROPIC_MODEL\nOPALE_CLOUD_AI=on").font(.caption.monospaced()).textSelection(.enabled)
                }
                NavigationLink("Autorisation et confidentialité de mon profil") { ProfileSettingsView() }
            }
            Section("Ce que fait l’iPhone") {
                Label(LocalAI.isAvailable ? "Modèle Apple disponible pour les fonctions locales" : "Modèle Apple indisponible sur cet appareil", systemImage: "iphone")
                Text("La lecture des factures reste locale. Les questions du chat passent au serveur Opale : le petit modèle de l’iPhone ne prend plus la main automatiquement. Les chiffres viennent du moteur ; les explications et actions proposées sont vérifiées.").font(.subheadline)
            }
            Section {
                if loading { ProgressView("Vérification…") }
                Button("Vérifier la connexion") { Task { await load() } }.disabled(loading)
                if let error { Text(error).foregroundStyle(OpaleTheme.loss) }
            }
        }.opaleList().navigationTitle("Choisir mon IA").navigationBarTitleDisplayMode(.inline)
        .task(id: session.refreshID) { mode = AssistantMode.current(session); await load() }
        .onChange(of: mode) { _, value in session.setPreference(value.rawValue, "assistant.mode"); onModeChanged() }
    }
    private func load() async {
        let key = session.profileKey
        loading = true; defer { loading = false }
        do {
            let result = try await session.api.assistantStatus()
            guard key == session.profileKey, !Task.isCancelled else { return }
            status = result; error = nil
        } catch { guard key == session.profileKey else { return }; self.error = error.localizedDescription }
    }
}
