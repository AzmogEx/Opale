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
    @State private var showAISettings = false
    @State private var mode: AssistantMode = .automatic
    @State private var pendingCloudIsRetry = false
    // EIA-022 : proposition d'escalade cloud en attente de consentement.
    @State private var pendingCloudQuestion: String?

    private let suggestions = [
        "Comment va mon épargne ?",
        "Quels sont mes risques ?",
        "Par où commencer ?",
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
                                    providerCard
                                    conversation
                                    DisclosureGroup("Outils et radar") {
                                        toolsRow
                                        if !risks.isEmpty { riskRadarCard }
                                    }

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
                        Button("Choisir mon IA") { showAISettings = true }
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
                mode = AssistantMode.current(session)
                messages = DiskCache.load([ChatMessage].self, key: "chat-" + session.profileKey)?.value ?? []
                await load()
            }
            .onDisappear { requestTask?.cancel() }
            .alert("Utiliser le cloud pour cette demande ?", isPresented: $confirmCloud) {
                Button("Envoyer les agrégats") { if let question = pendingCloudQuestion { ask(question, allowCloud: true, appendUser: !pendingCloudIsRetry) } }
                Button("Annuler", role: .cancel) {
                    if !pendingCloudIsRetry { draft = pendingCloudQuestion ?? ""; pendingCloudQuestion = nil }
                }
            } message: { Text("Seuls une intention structurée et des agrégats minimisés sont transmis. Ces montants restent sensibles. Les textes libres et documents ne sont pas transmis. Ton profil doit autoriser le cloud.") }
            .sheet(isPresented: $showDecision) {
                DecisionSheet()
            }
            .sheet(isPresented: $showAISettings) {
                NavigationStack {
                    AssistantSettingsView { mode = AssistantMode.current(session); pendingCloudQuestion = nil }
                        .toolbar { ToolbarItem(placement: .cancellationAction) { Button("Fermer") { showAISettings = false } } }
                }
            }
            .sheet(isPresented: $showReview) {
                ReviewSheet()
            }
        }
    }

    private var providerCard: some View {
        GlassCard {
            VStack(alignment: .leading, spacing: 8) {
                Label(mode.title, systemImage: mode == .cloud ? "cloud" : "server.rack").font(.subheadline.weight(.semibold))
                if let status {
                    Text(mode == .cloud ? (status.cloudConfigured ? "Cloud configuré · accord demandé avant l’envoi" : "Cloud à configurer") : (status.homelabAvailable ? "Modèle du PC joignable" : "Moteur Opale actif · PC non connecté"))
                        .font(.caption).foregroundStyle(.secondary)
                }
                Button("Choisir ou connecter mon IA") { showAISettings = true }.font(.footnote).accessibilityIdentifier("assistant.settings")
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
                VStack(alignment: .leading, spacing: 8) {
                    ChatBubble(message: message)
                    if let actions = message.actions {
                        ForEach(actions) { action in actionButton(action) }
                    }
                }
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
        if mode == .cloud {
            pendingCloudQuestion = question; pendingCloudIsRetry = false; confirmCloud = true
        } else { ask(question, allowCloud: false) }
    }

    private func ask(_ question: String, allowCloud: Bool, appendUser: Bool = true) {
        if appendUser {
            messages.append(ChatMessage(role: .user, text: question, tier: ""))
        }
        isThinking = true
        pendingCloudQuestion = nil
        let profileKey = session.profileKey
        let api = session.api
        let history = (appendUser ? messages.dropLast() : messages[...]).suffix(12).map { APIClient.HistoryMessage(role: $0.role == .user ? "user" : "assistant", text: $0.text) }
        requestTask = Task {
            defer { if session.profileKey == profileKey { isThinking = false; persistMessages() } }

            do {
                let resp = try await api.ask(question: question, allowCloud: allowCloud, history: history, provider: allowCloud ? "cloud" : mode.apiValue)
                guard !Task.isCancelled, session.profileKey == profileKey else { return }
                facts = resp.facts ?? []
                messages.append(ChatMessage(role: .assistant, text: resp.answer, tier: resp.tier, state: resp.state, providerState: resp.providerState, actions: resp.actions))
                // Repli moteur + cloud configuré → proposer l'escalade (EIA-021).
                if resp.cloudEligible == true, status?.cloudConfigured == true, !allowCloud {
                    pendingCloudQuestion = question; pendingCloudIsRetry = true
                }
            } catch {
                guard !Task.isCancelled, session.profileKey == profileKey else { return }
                messages.append(ChatMessage(role: .assistant,
                    text: "Impossible de répondre : \(error.localizedDescription)", tier: ""))
            }
        }
    }

    @ViewBuilder private func actionButton(_ action: AssistantAction) -> some View {
        switch action.id {
        case "journey": Button(action.title, systemImage: "list.number") { session.selectedTab = "home" }
        case "contracts": NavigationLink(action.title) { ContractsView() }
        case "variable_incomes": NavigationLink(action.title) { VariableIncomesView() }
        case "calendar": NavigationLink(action.title) { CalendarView() }
        case "goals": Button(action.title, systemImage: "target") { session.selectedTab = "projection" }
        case "investments": NavigationLink(action.title) { InvestmentExplorerView() }
        case "decision": Button(action.title, systemImage: "scalemass") { showDecision = true }
        case "ai_settings": Button(action.title, systemImage: "server.rack") { showAISettings = true }
        default: EmptyView()
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
    var actions: [AssistantAction]? = nil
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
                    if let provider = message.providerState, ["unavailable", "invalid", "invalid_response", "timeout"].contains(provider) { Text("IA indisponible ou réponse non vérifiée · repli sur le moteur").font(.caption2) }
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
        case "data": "Moteur Opale · données vérifiées"
        case "guide": "Guide Opale · explication intégrée"
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
