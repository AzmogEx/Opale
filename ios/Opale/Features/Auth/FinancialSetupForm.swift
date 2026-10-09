import SwiftUI

/// Six pages, un brouillon protégé et une validation finale atomique côté serveur.
struct FinancialSetupForm: View {
    let profileKey: String
    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @State private var draft: FinancialSetupDraft
    @State private var step: Int
    @State private var revision: Int
    @State private var accounts: [Asset] = []
    @State private var accountsError: String?
    @State private var errorMessage: String?
    @State private var busy = false
    @State private var completed = false
    @State private var conflict = false
    @State private var operation: Task<Void, Never>?
    @FocusState private var focused: String?

    init(initialState: FinancialSetupState, profileKey: String) {
        self.profileKey = profileKey
        let cached = FinancialSetupCache.load(profileKey: profileKey)
        let resume = cached?.revision == initialState.revision ? cached : nil
        _draft = State(initialValue: resume?.draft ?? initialState.draft)
        _step = State(initialValue: min(5, max(0, resume?.step ?? initialState.step)))
        _revision = State(initialValue: initialState.revision)
    }

    private let titles = ["Tes revenus", "Ton compte principal", "Tes charges fixes", "Tes abonnements", "Ton budget et ton objectif", "Ton point de départ"]
    private let subtitles = [
        "Commençons par ce qui entre sur ton compte. Tu pourras compléter le reste plus tard.",
        "Choisis le compte qui reçoit ton salaire et paie tes prélèvements. Pour commencer, les montants sont en euros.",
        "Logement, énergie, assurances… ajoute les dépenses qui reviennent régulièrement.",
        "Streaming, sport, logiciels… quelques petites sommes peuvent peser sur l’année.",
        "Prévois tes dépenses du quotidien, puis choisis un premier projet si tu le souhaites.",
        "Vérifie tes informations avant de les enregistrer. Tu pourras ensuite les modifier dans Opale."
    ]

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    VStack(alignment: .leading, spacing: 10) {
                        HStack {
                            Text("ÉTAPE \(step + 1) SUR 6").font(.caption.weight(.semibold)).foregroundStyle(OpaleTheme.accent)
                            Spacer()
                            Image(systemName: step == 5 ? "checkmark.seal" : "sparkles").foregroundStyle(OpaleTheme.accent)
                        }
                        ProgressView(value: Double(step + 1), total: 6).accessibilityLabel("Étape \(step + 1) sur 6")
                        Text(titles[step]).font(.title2.bold()).accessibilityAddTraits(.isHeader)
                        Text(subtitles[step]).font(.subheadline).foregroundStyle(.secondary)
                    }.padding(.vertical, 6)
                }
                FinancialSetupPages(draft: $draft, step: $step, focused: $focused, accounts: accounts, accountsError: accountsError, titles: titles) {
                    Task { await loadAccounts() }
                }
                if let errorMessage {
                    Section {
                        Label(errorMessage, systemImage: "exclamationmark.circle")
                            .foregroundStyle(OpaleTheme.loss).accessibilityIdentifier("setup.error")
                        if conflict {
                            Button("Recharger le brouillon du serveur") { start { await reloadAfterConflict() } }
                        }
                        Button("Fermer et conserver sur cet iPhone") { cache(); dismiss() }
                            .accessibilityIdentifier("setup.close.local")
                    }
                }
                if session.isOffline {
                    Section {
                        Text("Tu es hors ligne. Ton brouillon reste sur cet iPhone ; reconnecte-toi pour l’enregistrer sur le serveur.")
                            .font(.caption).foregroundStyle(.secondary)
                        Button("Reprendre plus tard sur cet iPhone") { cache(); dismiss() }
                    }
                }
                Section {
                    Label("Tu peux t’arrêter et reprendre depuis Réglages → Ma situation de départ.", systemImage: "lock.shield")
                        .font(.caption).foregroundStyle(.secondary)
                }
            }
            .opaleList()
            .scrollDismissesKeyboard(.interactively)
            .disabled(busy)
            .navigationTitle("Bienvenue dans Opale")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Plus tard") { start { await postpone() } }.disabled(busy)
                        .accessibilityIdentifier("setup.later")
                }
                ToolbarItemGroup(placement: .keyboard) {
                    Spacer()
                    Button("OK") { focused = nil }
                }
            }
            .safeAreaInset(edge: .bottom) { navigationButtons }
            .interactiveDismissDisabled()
            .task { await loadAccounts() }
            .onChange(of: draft) { _, _ in cache() }
            .onChange(of: step) { _, _ in cache() }
            .onDisappear { cache(); operation?.cancel() }
        }
    }

    private var navigationButtons: some View {
        HStack {
            if step > 0 {
                Button("Retour", systemImage: "chevron.left") {
                    focused = nil
                    errorMessage = nil
                    withAnimation(reduceMotion ? nil : .snappy) { step -= 1 }
                }.buttonStyle(.glass).disabled(busy).accessibilityIdentifier("setup.back")
            }
            Button {
                focused = nil
                start { await advance() }
            } label: {
                HStack {
                    if busy { ProgressView().tint(.white) }
                    Text(step == 5 ? "Valider ma situation" : "Continuer").font(.headline)
                }.frame(maxWidth: .infinity).padding(.vertical, 6)
            }
            .buttonStyle(.glassProminent)
            .disabled(busy || session.isOffline || conflict)
            .accessibilityIdentifier("setup.next")
        }
        .padding()
        .background(.regularMaterial)
    }

    // MARK: - Sauvegarde et reprise

    private func cache() {
        guard !completed, session.profileKey == profileKey else { return }
        FinancialSetupCache.save(draft: draft, step: step, revision: revision, profileKey: profileKey)
    }

    private func start(_ action: @escaping @MainActor () async -> Void) {
        guard !busy else { return }
        busy = true
        operation = Task {
            defer { busy = false }
            await action()
        }
    }

    private func loadAccounts() async {
        do {
            let loaded = try await session.api.listAssets()
            guard !Task.isCancelled, session.profileKey == profileKey else { return }
            accounts = loaded.filter { !$0.archived && $0.currency == "EUR" && ($0.kind == .checking || $0.kind == .savings) }
            accountsError = nil
        } catch { if !Task.isCancelled { accountsError = "Tes comptes n’ont pas pu être chargés. Réessaie avant de choisir un compte existant." } }
    }

    private func advance() async {
        let issues = draft.validationErrors(forStep: step)
        guard issues.isEmpty else { errorMessage = issues.joined(separator: "\n"); return }
        errorMessage = nil
        let api = session.api
        let next = min(5, step + 1)
        do {
            let saved = try await api.saveFinancialSetup(draft: draft, step: next, revision: revision)
            guard !Task.isCancelled, session.profileKey == profileKey else { return }
            revision = saved.revision
            if step == 5 {
                let result = try await api.completeFinancialSetup(revision: revision)
                guard !Task.isCancelled, session.profileKey == profileKey else { return }
                guard result.status == "completed" else { throw APIError.invalidResponse }
                completed = true
                FinancialSetupCache.clear(profileKey: profileKey)
                session.changed()
                dismiss()
            } else {
                withAnimation(reduceMotion ? nil : .snappy) { step = next }
                cache()
            }
        } catch { handle(error) }
    }

    private func postpone() async {
        focused = nil
        cache()
        let api = session.api
        do {
            let saved = try await api.saveFinancialSetup(draft: draft, step: step, revision: revision)
            guard !Task.isCancelled, session.profileKey == profileKey else { return }
            revision = saved.revision
            let skipped = try await api.skipFinancialSetup(revision: revision)
            guard !Task.isCancelled, session.profileKey == profileKey else { return }
            revision = skipped.revision
            cache()
            dismiss()
        } catch { handle(error) }
    }

    private func reloadAfterConflict() async {
        do {
            let fresh = try await session.api.fetchFinancialSetup()
            guard !Task.isCancelled, session.profileKey == profileKey else { return }
            if fresh.status == "completed" {
                completed = true
                FinancialSetupCache.clear(profileKey: profileKey)
                session.changed()
                dismiss()
                return
            }
            draft = fresh.draft
            step = min(5, max(0, fresh.step))
            revision = fresh.revision
            conflict = false
            errorMessage = nil
            cache()
        } catch { handle(error) }
    }

    private func handle(_ error: Error) {
        guard !Task.isCancelled, session.profileKey == profileKey else { return }
        cache()
        if case APIError.badStatus(409, _) = error {
            conflict = true
            errorMessage = "Ce formulaire a changé sur un autre appareil ou a déjà été validé. Recharge la version du serveur avant de continuer."
        } else { errorMessage = "\(error.localizedDescription) Ton brouillon reste conservé sur cet appareil." }
    }
}
