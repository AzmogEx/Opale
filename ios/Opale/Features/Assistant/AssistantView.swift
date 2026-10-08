import SwiftUI

/// Onglet Assistant — le cerveau d'Opale (P5).
///
/// Chat patrimonial (EF-050/051), radar de risques (EF-061), bilan mensuel
/// (EF-062) et Mode Décision (EF-052). Tous les chiffres viennent du moteur
/// déterministe ; l'IA (cascade N2 homelab → N3 cloud anonymisé) ne fait
/// qu'expliquer — et l'écran reste utile sans elle (EIA-020/021).
struct AssistantView: View {
    @Environment(SessionStore.self) private var session

    @State private var messages: [ChatMessage] = []
    @State private var draft = ""
    @State private var isThinking = false
    @State private var requestTask: Task<Void, Never>?
    @State private var facts: [AssistantFact] = []
    @State private var confirmCloud = false
    @State private var risks: [Risk] = []
    @State private var status: AssistantStatus?
    @State private var showDecision = false
    @State private var showReview = false
    // EIA-022 : proposition d'escalade cloud en attente de consentement.
    @State private var pendingCloudQuestion: String?

    private let suggestions = [
        "Comment va mon épargne ?",
        "Quels sont mes risques ?",
        "Puis-je dépenser 500 € ?",
    ]

    var body: some View {
        NavigationStack {
            ZStack {
                OpaleBackdrop()

                VStack(spacing: 0) {
                    ScrollViewReader { proxy in
                        ScrollView {
                            GlassEffectContainer(spacing: 14) {
                                VStack(spacing: 14) {
                                    toolsRow
                                    if !risks.isEmpty {
                                        riskRadarCard
                                    }
                                    conversation
                                }
                                .padding(.horizontal)
                                .padding(.bottom, 12)
                            }
                        }
                        .scrollEdgeEffectStyle(.soft, for: .top)
                        .scrollDismissesKeyboard(.interactively)
                        .animation(.spring(duration: 0.45, bounce: 0.22), value: messages.count)
                        .onChange(of: messages.count) {
                            if let last = messages.last {
                                withAnimation(.spring(duration: 0.4)) {
                                    proxy.scrollTo(last.id, anchor: .bottom)
                                }
                            }
                        }
                    }
                    inputBar
                }
            }
            .navigationTitle("Assistant")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    // État de la cascade (EIA-021) — le reste vit dans Réglages.
                    Menu {
                        if let status {
                            Label(status.homelabAvailable ? "Homelab en ligne" : "Homelab hors ligne",
                                  systemImage: status.homelabAvailable ? "server.rack" : "wifi.slash")
                            Label(status.cloudConfigured ? "Cloud configuré · opt-in" : "Cloud non configuré",
                                  systemImage: "cloud")
                        }
                        Button("Effacer la conversation", role: .destructive) { requestTask?.cancel(); messages = []; facts = []; persistMessages() }
                        if LocalAI.isAvailable {
                            Label("IA locale iPhone active", systemImage: "iphone")
                        }
                    } label: {
                        Image(systemName: "point.3.connected.trianglepath.dotted")
                    }
                }
            }
            .task {
                messages = DiskCache.load([ChatMessage].self, key: "chat-" + session.profileKey)?.value ?? []
                await load()
            }
            .onDisappear { requestTask?.cancel() }
            .alert("Utiliser le cloud pour cette demande ?", isPresented: $confirmCloud) {
                Button("Envoyer les agrégats") { if let question = pendingCloudQuestion { ask(question, allowCloud: true) } }
                Button("Annuler", role: .cancel) {}
            } message: { Text("Seuls une intention structurée et des agrégats minimisés sont transmis. Ces montants restent sensibles. Les textes libres et documents ne sont pas transmis. Ton profil doit autoriser le cloud.") }
            .sheet(isPresented: $showDecision) {
                DecisionSheet()
            }
            .sheet(isPresented: $showReview) {
                ReviewSheet()
            }
        }
    }

    // MARK: - Outils (Mode Décision, Bilan)

    private var toolsRow: some View {
        HStack(spacing: 12) {
            toolButton("Mode Décision", icon: "scalemass.fill") { showDecision = true }
            toolButton("Bilan mensuel", icon: "doc.text.magnifyingglass") { showReview = true }
        }
    }

    private func toolButton(_ title: String, icon: String, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            VStack(spacing: 6) {
                Image(systemName: icon)
                    .font(.title3)
                    .foregroundStyle(OpaleTheme.accent)
                Text(title)
                    .font(.footnote.weight(.semibold))
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 14)
        }
        .buttonStyle(.pressable)
        .glassEffect(.regular.interactive(), in: .rect(cornerRadius: 20))
    }

    // MARK: - Radar de risques (EF-061)

    private var riskRadarCard: some View {
        GlassCard {
            VStack(alignment: .leading, spacing: 10) {
                Text("Radar de risques")
                    .font(.footnote.weight(.semibold))
                    .foregroundStyle(.secondary)
                    .textCase(.uppercase)

                ForEach(risks) { risk in
                    HStack(alignment: .top, spacing: 10) {
                        Image(systemName: severityIcon(risk.severity))
                            .foregroundStyle(severityColor(risk.severity))
                            .font(.body)
                            .padding(.top, 1)
                        VStack(alignment: .leading, spacing: 2) {
                            Text(risk.title)
                                .font(.subheadline.weight(.semibold))
                            Text(risk.detail)
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                    }
                }
            }
        }
    }

    private func severityIcon(_ s: String) -> String {
        switch s {
        case "critical": "exclamationmark.octagon.fill"
        case "warning": "exclamationmark.triangle.fill"
        default: "info.circle.fill"
        }
    }

    private func severityColor(_ s: String) -> Color {
        switch s {
        case "critical": OpaleTheme.loss
        case "warning": .orange
        default: OpaleTheme.accent
        }
    }

    // MARK: - Conversation (EF-050/051)

    @ViewBuilder
    private var conversation: some View {
        if messages.isEmpty {
            VStack(spacing: 10) {
                Image(systemName: "sparkles")
                    .font(.largeTitle)
                    .foregroundStyle(OpaleTheme.iridescent)
                Text("Pose une question sur ton patrimoine")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
                ForEach(suggestions, id: \.self) { s in
                    Button {
                        draft = s
                        send()
                    } label: {
                        Text(s)
                            .font(.subheadline)
                            .padding(.horizontal, 14)
                            .padding(.vertical, 9)
                    }
                    .buttonStyle(.pressable)
                    .glassEffect(.regular.interactive(), in: .capsule)
                }
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 24)
        } else {
            ForEach(messages) { message in
                ChatBubble(message: message)
                    .id(message.id)
                    // Chaque bulle surgit du bas avec un ressort.
                    .transition(.asymmetric(
                        insertion: .move(edge: .bottom)
                            .combined(with: .opacity)
                            .combined(with: .scale(scale: 0.96, anchor: .bottom)),
                        removal: .opacity
                    ))
            }
            // EIA-021/022 : proposer l'escalade cloud, avec consentement.
            if pendingCloudQuestion != nil {
                Button {
                    confirmCloud = true
                } label: {
                    Label("Analyse cloud · agrégats sensibles",
                          systemImage: "cloud.fill")
                        .font(.footnote.weight(.semibold))
                        .padding(.horizontal, 14)
                        .padding(.vertical, 10)
                }
                .buttonStyle(.plain)
                .glassEffect(.regular.tint(OpaleTheme.accent.opacity(0.2)).interactive(), in: .capsule)
            }
            if !facts.isEmpty {
                VStack(alignment: .leading) {
                    ForEach(facts) { fact in
                        VStack(alignment: .leading) {
                            Text(fact.text).sensitive()
                            if let value = fact.value_cents { AmountText(cents: value) }
                            Text("\(fact.period) · \(fact.source)").font(.caption2).foregroundStyle(.secondary)
                        }
                    }
                }.padding().glassEffect(.regular, in: .rect(cornerRadius: 16))
            }
            if isThinking {
                HStack {
                    ProgressView()
                    Text("Analyse en cours…")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                .frame(maxWidth: .infinity, alignment: .leading)
            }
        }
    }

    // MARK: - Saisie

    private var inputBar: some View {
        HStack(spacing: 10) {
            TextField("Ta question…", text: $draft, axis: .vertical)
                .lineLimit(1...4)
                .textFieldStyle(.plain)
                .padding(.horizontal, 14)
                .padding(.vertical, 10)
                .glassEffect(.regular, in: .rect(cornerRadius: 22))
                .onSubmit(send)

            Button(action: send) {
                Image(systemName: "arrow.up.circle.fill")
                    .font(.system(size: 30))
                    .foregroundStyle(draft.isEmpty ? Color.secondary : OpaleTheme.accent)
            }
            .disabled(draft.isEmpty || isThinking)
            .sensoryFeedback(.impact(weight: .medium), trigger: messages.count)
        }
        .padding(.horizontal)
        .padding(.vertical, 8)
    }

    // MARK: - Actions

    private func send() {
        let question = draft.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !question.isEmpty, !isThinking else { return }
        draft = ""
        SoundPlayer.play(.send)
        ask(question, allowCloud: false)
    }

    private func ask(_ question: String, allowCloud: Bool) {
        if !allowCloud {
            messages.append(ChatMessage(role: .user, text: question, tier: ""))
        }
        isThinking = true
        pendingCloudQuestion = nil
        let profileKey = session.profileKey
        let api = session.api
        let history = messages.dropLast().suffix(12).map { APIClient.HistoryMessage(role: $0.role == .user ? "user" : "assistant", text: $0.text) }
        requestTask = Task {
            defer { if session.profileKey == profileKey { isThinking = false; persistMessages() } }

            // N1 (EIA-001) : le modèle local de l'iPhone d'abord — rien ne
            // quitte l'appareil. Indisponible/insuffisant → cascade backend.
            if !allowCloud, let local = await LocalAI.conceptualAnswer(question) {
                guard !Task.isCancelled, session.profileKey == profileKey else { return }
                messages.append(ChatMessage(role: .assistant, text: local, tier: "n1"))
                return
            }

            do {
                let resp = try await api.ask(question: question, allowCloud: allowCloud, history: history)
                guard !Task.isCancelled, session.profileKey == profileKey else { return }
                facts = resp.facts ?? []
                messages.append(ChatMessage(role: .assistant, text: resp.answer, tier: resp.tier, state: resp.state, providerState: resp.providerState))
                // Repli moteur + cloud configuré → proposer l'escalade (EIA-021).
                if resp.cloudEligible == true, status?.cloudConfigured == true, !allowCloud {
                    pendingCloudQuestion = question
                }
            } catch {
                guard !Task.isCancelled, session.profileKey == profileKey else { return }
                messages.append(ChatMessage(role: .assistant,
                    text: "Impossible de répondre : \(error.localizedDescription)", tier: ""))
            }
        }
    }

    private func persistMessages() {
        messages = Array(messages.suffix(20))
        DiskCache.save(messages, key: "chat-" + session.profileKey)
    }

    private func load() async {
        async let r = session.api.risks()
        async let s = session.api.assistantStatus()
        risks = (try? await r) ?? []
        status = try? await s
    }

}

// MARK: - Messages

struct ChatMessage: Identifiable, Hashable, Codable {
    enum Role: String, Codable { case user, assistant }
    var id = UUID()
    let role: Role
    let text: String
    let tier: String // "n2" | "n3" | ""
    var state: String? = nil
    var providerState: String? = nil
}

/// Une bulle de conversation, style verre.
private struct ChatBubble: View {
    let message: ChatMessage

    var body: some View {
        HStack {
            if message.role == .user { Spacer(minLength: 40) }
            VStack(alignment: .leading, spacing: 4) {
                Text(message.text)
                    .sensitive()
                    .font(.subheadline)
                    .frame(maxWidth: .infinity, alignment: .leading)
                if message.role == .assistant {
                    if let state = message.state, state != "grounded" { Text(state == "clarification_needed" ? "Précision nécessaire" : state == "unsupported" ? "Demande non prise en charge" : "Réponse du moteur").font(.caption2) }
                    if let provider = message.providerState, ["unavailable", "invalid", "timeout"].contains(provider) { Text("IA indisponible ou réponse non vérifiée · repli sur le moteur").font(.caption2) }
                    Label(tierLabel, systemImage: tierIcon)
                        .font(.caption2)
                        .foregroundStyle(.tertiary)
                }
            }
            .padding(.horizontal, 14)
            .padding(.vertical, 10)
            .glassEffect(
                message.role == .user
                    ? .regular.tint(OpaleTheme.accent.opacity(0.35))
                    : .regular,
                in: .rect(cornerRadius: 18)
            )
            if message.role == .assistant { Spacer(minLength: 40) }
        }
    }

    private var tierLabel: String {
        switch message.tier {
        case "data": "Moteur — réponse exacte"
        case "n1": "iPhone — 100 % local"
        case "n2": "Homelab — privé"
        case "n3": "Cloud — agrégats minimisés"
        default: "Moteur déterministe"
        }
    }

    private var tierIcon: String {
        switch message.tier {
        case "data": "magnifyingglass"
        case "n1": "iphone"
        case "n2": "server.rack"
        case "n3": "cloud"
        default: "function"
        }
    }
}
