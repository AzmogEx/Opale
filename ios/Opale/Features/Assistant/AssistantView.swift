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
                            Label(status.cloudConfigured ? "Cloud configuré (anonymisé)" : "Cloud non configuré",
                                  systemImage: "cloud")
                        }
                        if LocalAI.isAvailable {
                            Label("IA locale iPhone active", systemImage: "iphone")
                        }
                    } label: {
                        Image(systemName: "point.3.connected.trianglepath.dotted")
                    }
                }
            }
            .task { await load() }
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
            if let question = pendingCloudQuestion {
                Button {
                    pendingCloudQuestion = nil
                    ask(question, allowCloud: true)
                } label: {
                    Label("Analyser avec le modèle cloud (données anonymisées)",
                          systemImage: "cloud.fill")
                        .font(.footnote.weight(.semibold))
                        .padding(.horizontal, 14)
                        .padding(.vertical, 10)
                }
                .buttonStyle(.plain)
                .glassEffect(.regular.tint(OpaleTheme.accent.opacity(0.2)).interactive(), in: .capsule)
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
        Task {
            defer { isThinking = false }

            // N1 (EIA-001) : le modèle local de l'iPhone d'abord — rien ne
            // quitte l'appareil. Indisponible/insuffisant → cascade backend.
            if !allowCloud, let local = await askLocally(question) {
                messages.append(ChatMessage(role: .assistant, text: local, tier: "n1"))
                return
            }

            do {
                let resp = try await session.api.ask(question: question, allowCloud: allowCloud)
                messages.append(ChatMessage(role: .assistant, text: resp.answer, tier: resp.tier))
                // Repli moteur + cloud configuré → proposer l'escalade (EIA-021).
                if resp.tier.isEmpty, status?.cloudConfigured == true, !allowCloud {
                    pendingCloudQuestion = question
                }
            } catch {
                messages.append(ChatMessage(role: .assistant,
                    text: "Impossible de répondre : \(error.localizedDescription)", tier: ""))
            }
        }
    }

    /// Tente la réponse sur le modèle local (N1) avec un contexte compact
    /// calculé par le moteur — jamais envoyé hors de l'appareil.
    private func askLocally(_ question: String) async -> String? {
        guard LocalAI.isAvailable else { return nil }
        guard let netWorth = try? await session.api.netWorth(),
              let health = try? await session.api.healthScore() else { return nil }

        var context = """
        Situation (chiffres exacts du moteur) :
        - Patrimoine net : \(MoneyFormat.eurosWhole(netWorth.net))
        - Actifs : \(MoneyFormat.eurosWhole(netWorth.assetsTotal)) ; dettes : \(MoneyFormat.eurosWhole(netWorth.liabilitiesTotal))
        - Score de santé financière : \(health.score)/100
        """
        for c in health.components {
            context += "\n  - \(c.name) : \(c.score)/\(c.max) (\(c.comment))"
        }
        for r in risks {
            context += "\n- Risque (\(r.severity)) : \(r.title)"
        }
        return await LocalAI.answer(context: context, question: question)
    }

    private func load() async {
        async let r = session.api.risks()
        async let s = session.api.assistantStatus()
        risks = (try? await r) ?? []
        status = try? await s
    }

}

// MARK: - Messages

struct ChatMessage: Identifiable, Hashable {
    enum Role { case user, assistant }
    let id = UUID()
    let role: Role
    let text: String
    let tier: String // "n2" | "n3" | ""
}

/// Une bulle de conversation, style verre.
private struct ChatBubble: View {
    let message: ChatMessage

    var body: some View {
        HStack {
            if message.role == .user { Spacer(minLength: 40) }
            VStack(alignment: .leading, spacing: 4) {
                Text(message.text)
                    .font(.subheadline)
                    .frame(maxWidth: .infinity, alignment: .leading)
                if message.role == .assistant {
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
        case "n1": "iPhone — 100 % local"
        case "n2": "Homelab — privé"
        case "n3": "Cloud — anonymisé"
        default: "Moteur — hors ligne"
        }
    }

    private var tierIcon: String {
        switch message.tier {
        case "n1": "iphone"
        case "n2": "server.rack"
        case "n3": "cloud"
        default: "function"
        }
    }
}
