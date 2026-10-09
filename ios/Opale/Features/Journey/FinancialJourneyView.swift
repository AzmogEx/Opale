import SwiftUI

/// One entry point for manual setup and subsequent reviews. Progress is owned by the profile on the server.
struct FinancialJourneyView: View {
    @Environment(SessionStore.self) private var session
    @State private var snapshot: FinancialJourneySnapshot?
    @State private var path: [JourneyStep] = []
    @State private var error: String?
    @State private var stale = false
    @State private var showSettings = false
    @State private var showDashboard = false
    private var cacheKey: String { "financial-journey-v1|" + session.profileKey }

    var body: some View {
        NavigationStack(path: $path) {
            List {
                if let snapshot {
                    welcome(snapshot)
                    if stale { Section { Label("Dernière lecture du \(snapshot.loadedAt.opaleFormatted(.dateTime.day().month().hour().minute())). Reconnecte-toi avant de valider une étape.", systemImage: "wifi.slash").font(.caption) } }
                    Section("Dans l’ordre, à ton rythme") {
                        ForEach(JourneyStep.allCases) { step in
                            NavigationLink(value: step) { stepRow(step, snapshot.state) }
                                .accessibilityIdentifier("journey.step.\(step.key)")
                        }
                    }
                    if !snapshot.alerts.isEmpty {
                        Section("À vérifier") {
                            ForEach(snapshot.alerts.prefix(3)) { alert in
                                NavigationLink { ContractDetailView(contractID: alert.contractID) } label: {
                                    Label { VStack(alignment: .leading) { Text(alert.title); Text(alert.detail).font(.caption).foregroundStyle(.secondary) } } icon: { Image(systemName: "bell.badge") }
                                }
                            }
                        }
                    }
                    Section("Après la saisie") {
                        NavigationLink(value: JourneyStep.review) { Label("Comprendre mon bilan et mes priorités", systemImage: "lightbulb") }
                        Button { showDashboard = true } label: { Label("Voir tous mes chiffres", systemImage: "chart.bar") }
                    }
                } else if error == nil {
                    ProgressView("Préparation de ton parcours…")
                }
                if let error {
                    Section {
                        Text(error).foregroundStyle(OpaleTheme.loss)
                        Button("Réessayer") { Task { await load() } }
                    }
                }
            }
            .opaleList()
            .navigationTitle("Mon parcours")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button {
                        session.discreetMode.toggle(); WidgetBridge.setDiscreet(session.discreetMode)
                    } label: { Image(systemName: session.discreetMode ? "eye.slash.fill" : "eye") }
                    .accessibilityLabel(session.discreetMode ? "Afficher les montants" : "Masquer les montants")
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button { showSettings = true } label: { Image(systemName: "gearshape") }.accessibilityLabel("Réglages")
                }
            }
            .navigationDestination(for: JourneyStep.self) { step in
                if let snapshot {
                    JourneyStepView(step: step, snapshot: snapshot, stale: stale) { skip, budget, currency in
                        try await advance(step, skip: skip, budget: budget, currency: currency)
                    } previous: {
                        path = step.rawValue > 0 ? [JourneyStep(rawValue: step.rawValue - 1)!] : []
                    }
                    .id(step)
                }
            }
            .sheet(isPresented: $showSettings) { SettingsView() }
            .sheet(isPresented: $showDashboard) { HomeView().safeAreaInset(edge: .bottom) { Button("Fermer le tableau de bord") { showDashboard = false }.buttonStyle(.glass).padding(8) } }
            .refreshable { await load() }
        }
        .task(id: session.refreshID) { await load() }
    }

    private func welcome(_ data: FinancialJourneySnapshot) -> some View {
        Section {
            VStack(alignment: .leading, spacing: 12) {
                Text(data.state.finished ? "Ta situation, au même endroit" : "Posons les choses, une par une")
                    .font(.title2.bold())
                Text("Saisis tes informations à la main, comprends ce qu’elles signifient, puis choisis ta prochaine action. Tu peux quitter et reprendre plus tard.")
                    .foregroundStyle(.secondary)
                if !data.assets.isEmpty && data.netWorth.complete != false {
                    LabeledContent("Patrimoine net enregistré") { AmountText(cents: data.netWorth.net, currency: data.netWorth.currency) }
                        .font(.subheadline)
                }
                ProgressView(value: Double(data.state.completedCount), total: Double(JourneyStep.allCases.count))
                    .tint(OpaleTheme.accent)
                Text("\(data.state.reviewed.count) étapes vérifiées · \(data.state.skipped.count) à reprendre")
                    .font(.caption).foregroundStyle(.secondary)
                NavigationLink(value: data.state.next) {
                    Label(data.state.finished ? "Relire mon bilan" : data.state.completedCount == 0 ? "Commencer mon parcours" : "Continuer mon parcours", systemImage: "arrow.right.circle.fill")
                        .frame(maxWidth: .infinity)
                }.buttonStyle(.glassProminent).accessibilityIdentifier("journey.continue")
                if data.state.completedCount > 0 && !data.state.finished {
                    Text("Prochaine étape : \(data.state.next.title)").font(.caption)
                }
            }.padding(.vertical, 8)
        }
    }
    private func stepRow(_ step: JourneyStep, _ state: FinancialJourneyState) -> some View {
        HStack(spacing: 12) {
            Text("\(step.rawValue + 1)").font(.headline.monospacedDigit()).foregroundStyle(OpaleTheme.accent).frame(width: 24)
            VStack(alignment: .leading, spacing: 3) {
                Text(step.title)
                Text(state.reviewed.contains(step.key) ? "Vérifié · tu peux modifier" : state.skipped.contains(step.key) ? "Passé pour l’instant" : "À compléter ou vérifier")
                    .font(.caption).foregroundStyle(.secondary)
            }
            Spacer(minLength: 0)
            if state.reviewed.contains(step.key) { Image(systemName: "checkmark.circle.fill").foregroundStyle(OpaleTheme.accent) }
        }.padding(.vertical, 4)
    }
    private func load() async {
        let key = session.profileKey, api = session.api
        if snapshot == nil, let cached = DiskCache.load(FinancialJourneySnapshot.self, key: cacheKey) {
            snapshot = cached.value; stale = true
        }
        do {
            let result = try await api.journeySnapshot()
            guard key == session.profileKey, !Task.isCancelled else { return }
            snapshot = result; stale = false; error = nil
            DiskCache.save(result, key: cacheKey)
        } catch {
            guard key == session.profileKey, !Task.isCancelled else { return }
            stale = true
            if case APIError.badStatus(404, _) = error {
                self.error = "Le serveur doit recevoir la mise à jour du parcours. Tes autres espaces restent accessibles."
            } else { self.error = error.localizedDescription }
        }
    }
    private func advance(_ step: JourneyStep, skip: Bool, budget: Cents?, currency: String) async throws {
        guard var data = snapshot, !stale else { throw APIError.invalidResponse }
        let key = session.profileKey
        if step == .budget && !skip { data.state.dailyBudget = budget; data.state.budgetCurrency = currency }
        data.state.mark(step, skipped: skip)
        do {
            data.state = try await session.api.saveFinancialJourney(data.state)
            guard key == session.profileKey, !Task.isCancelled else { return }
            snapshot = data; DiskCache.save(data, key: cacheKey)
            path = step == .review ? [] : [JourneyStep(rawValue: step.rawValue + 1)!]
        } catch {
            if case APIError.badStatus(409, _) = error { await load() }
            throw error
        }
    }
}

private struct JourneyStepView: View {
    let step: JourneyStep
    let snapshot: FinancialJourneySnapshot
    let stale: Bool
    var advance: (Bool, Cents?, String) async throws -> Void
    var previous: () -> Void
    @Environment(SessionStore.self) private var session
    @State private var sheet: JourneySheet?
    @State private var budget = ""
    @State private var currency = "EUR"
    @State private var busy = false
    @State private var error: String?
    private var draftKey: String { "journey-budget-draft|\(session.profileKey)|\(snapshot.state.revision)" }
    private struct BudgetDraft: Codable { var amount: String; var currency: String }
    private var parsedBudget: Cents? { FinancialTools.amount(budget, currency: currency, zeroAllowed: true) }

    var body: some View {
        List {
            Section {
                Text("ÉTAPE \(step.rawValue + 1) SUR 7").font(.caption.weight(.semibold)).foregroundStyle(OpaleTheme.accent)
                Text(step.title).font(.title2.bold())
                Text(step.introduction).foregroundStyle(.secondary)
                DisclosureGroup("Pourquoi on te le demande") { Text(step.explanation).font(.subheadline).padding(.vertical, 6) }
            }
            stepContent
            if let error { Section { Text(error).foregroundStyle(OpaleTheme.loss) } }
            if stale { Section { Label("Données en cache. La validation attend une connexion au serveur.", systemImage: "wifi.slash").font(.caption) } }
            Section {
                Button { Task { await save(skip: false) } } label: {
                    Label(step == .review ? "Terminer mon parcours" : "Valider et continuer", systemImage: "checkmark.circle")
                        .frame(maxWidth: .infinity)
                }.buttonStyle(.glassProminent)
                    .disabled(busy || stale || (step == .budget && parsedBudget == nil))
                    .accessibilityIdentifier("journey.validate")
                Button("Passer pour l’instant") { Task { await save(skip: true) } }.disabled(busy || stale).accessibilityIdentifier("journey.skip")
                if step.rawValue > 0 { Button("Étape précédente", action: previous).disabled(busy) }
                Text("Les éléments sont enregistrés quand tu les ajoutes. Valider indique que tu as vérifié cette étape. Passer ne supprime aucune donnée.")
                    .font(.caption).foregroundStyle(.secondary)
            }
        }
        .opaleList().scrollDismissesKeyboard(.interactively)
        .navigationTitle("\(step.rawValue + 1) / 7").navigationBarTitleDisplayMode(.inline)
        .id(step)
        .sheet(item: $sheet) { item in sheetContent(item) }
        .task(id: step) {
            if step == .budget {
                let draft = DiskCache.load(BudgetDraft.self, key: draftKey)?.value
                currency = draft?.currency ?? snapshot.state.budgetCurrency
                budget = draft?.amount ?? snapshot.state.dailyBudget.map { MoneyFormat.input($0, currency: currency) } ?? ""
            }
        }
        .onChange(of: budget) { _, _ in cacheBudget() }
        .onChange(of: currency) { _, _ in cacheBudget() }
    }

    @ViewBuilder private var stepContent: some View {
        switch step {
        case .accounts:
            Section("Tes comptes actuels") {
                ForEach(snapshot.accounts) { asset in assetLink(asset) }
                if snapshot.accounts.isEmpty { Text("Pas encore de compte. Un seul suffit pour commencer.").foregroundStyle(.secondary) }
                Button("Ajouter un compte ou un livret", systemImage: "plus") { sheet = .account }.accessibilityIdentifier("journey.addAccount")
            }
        case .income:
            Section("Revenus réguliers") {
                ForEach(snapshot.ongoingRules.filter { $0.amount_cents > .zero }) { rule in legacyRule(rule) }
                ForEach(snapshot.incomes.filter { $0.active && $0.isFixed }) { income in
                    Button { sheet = .fixed(income) } label: { moneyLine(income.name, income.usual, income.currency, FinancialTools.frequencyLabel(income.frequency)) }.buttonStyle(.plain)
                }
                Button("Ajouter un salaire ou revenu fixe", systemImage: "plus") { sheet = .fixed(nil) }.accessibilityIdentifier("journey.addSalary")
            }
            Section("Revenus qui changent selon le mois") {
                ForEach(snapshot.incomes.filter { $0.active && !$0.isFixed }) { income in
                    Button { sheet = .variable(income) } label: { moneyLine(income.name, income.usual, income.currency, "Habituel · " + FinancialTools.frequencyLabel(income.frequency)) }.buttonStyle(.plain)
                }
                Button("Ajouter un revenu variable", systemImage: "plus") { sheet = .variable(VariableIncome()) }
                Text("Renseigne le minimum, l’habituel et le maximum. Le bilan utilise ton choix prudent ou habituel ; une source sans prévision est exclue du budget.").font(.caption).foregroundStyle(.secondary)
            }
        case .expenses:
            Section("Charges déjà déclarées") {
                ForEach(snapshot.ongoingRules.filter { $0.amount_cents < .zero }) { rule in legacyRule(rule) }
                ForEach(snapshot.contracts.filter { $0.active && $0.category != "subscription" }) { contract in contractLink(contract) }
                Button("Ajouter une charge fixe", systemImage: "plus") {
                    var value = FinancialContract(); value.category = "housing"; sheet = .contract(value)
                }.accessibilityIdentifier("journey.addExpense")
            }
            Section { Text("Pense au loyer, à l’énergie, aux assurances, aux transports et aux mensualités de crédit. N’ajoute pas deux fois la même charge si elle est déjà dans la liste.").font(.subheadline) }
        case .subscriptions:
            Section("Abonnements déjà déclarés") {
                ForEach(snapshot.contracts.filter { $0.active && $0.category == "subscription" }) { contract in contractLink(contract) }
                Button("Ajouter un abonnement", systemImage: "plus") { sheet = .contract(FinancialContract()) }.accessibilityIdentifier("journey.addSubscription")
            }
            Section { Text("Tu peux commencer avec le nom et le tarif. Les dates d’engagement et la détection des hausses sont dans les options du contrat.").font(.subheadline) }
        case .wealth:
            Section("Biens et placements") {
                ForEach(snapshot.otherAssets) { asset in assetLink(asset) }
                Button("Ajouter un bien ou placement", systemImage: "plus") { sheet = .asset }
            }
            Section("Capital restant à rembourser") {
                ForEach(snapshot.liabilities.filter { !$0.archived }) { liability in
                    NavigationLink { LiabilityDetailView(liability: liability, onChanged: {}) } label: { holdingLine(liability.name, liability.latestValue, liability.currency) }
                }
                Button("Ajouter un crédit ou une dette", systemImage: "plus") { sheet = .liability }
            }
            Section { Text("Tu n’as ni placement ni crédit ? Tu peux valider cette étape sans ajouter d’élément.").font(.caption).foregroundStyle(.secondary) }
        case .budget:
            Section("Dépenses du quotidien par mois") {
                Picker("Devise du budget", selection: Binding(get: { currency }, set: { if currency != $0 { budget = "" }; currency = $0 })) { ForEach(FinancialTools.currencies, id: \.self) { Text($0).tag($0) } }
                TextField("Courses, sorties, quotidien (\(currency))", text: $budget).keyboardType(.decimalPad).accessibilityIdentifier("journey.dailyBudget")
                Text("Une estimation suffit. Mets 0 si tu ne réserves aucun montant ; laisse vide et passe l’étape si tu ne sais pas encore.").font(.caption).foregroundStyle(.secondary)
            }
            Section("Un projet concret") {
                ForEach(snapshot.goals) { goal in
                    Button { sheet = .goal(goal) } label: { moneyLine(goal.name, goal.target, "EUR", "Objectif · suivi dans Projection") }.buttonStyle(.plain)
                }
                Button("Ajouter un objectif d’épargne", systemImage: "plus") { sheet = .goal(nil) }
                Text("Commence par une réserve accessible pour les imprévus si tu n’en as pas. Les versements réservés aux objectifs ne sont pas déduits du reste ci-dessous.").font(.caption).foregroundStyle(.secondary)
            }
        case .review:
            JourneyReviewSections(snapshot: snapshot)
        }
    }
    @ViewBuilder private func sheetContent(_ item: JourneySheet) -> some View {
        switch item {
        case .account: AssetFormSheet(allowedKinds: [.checking, .savings], onSaved: {})
        case .asset: AssetFormSheet(allowedKinds: AssetKind.allCases.filter { ![.checking, .savings].contains($0) }, onSaved: {})
        case .liability: LiabilityFormSheet(onSaved: {})
        case .fixed(let value): FixedIncomeSheet(initial: value)
        case .variable(let value): VariableIncomeEditSheet(initial: value)
        case .contract(let value): ContractEditSheet(initial: value, guided: true)
        case .rule(let value): CalendarRuleSheet(existing: value, guided: true, saved: {})
        case .goal(let value): GoalFormSheet(existing: value, onSaved: {})
        }
    }
    private func assetLink(_ asset: Asset) -> some View {
        NavigationLink { AssetDetailView(asset: asset, onChanged: {}) } label: { holdingLine(asset.name, asset.currentValue ?? asset.latestValue, asset.currency) }
    }
    private func contractLink(_ contract: FinancialContract) -> some View {
        NavigationLink { ContractDetailView(contractID: contract.id) } label: { moneyLine(contract.name, contract.amount, contract.currency, FinancialTools.frequencyLabel(contract.frequency)) }
    }
    private func legacyRule(_ rule: CalendarRule) -> some View {
        Button { sheet = .rule(rule) } label: {
            moneyLine(rule.label, rule.amount_cents, snapshot.assets.first { $0.id == rule.asset_id }?.currency ?? "EUR", "Calendrier · " + FinancialTools.frequencyLabel(rule.frequency))
        }.buttonStyle(.plain)
    }
    private func holdingLine(_ name: String, _ amount: Cents?, _ currency: String) -> some View {
        HStack { Text(name); Spacer(); if let amount { AmountText(cents: amount, currency: currency) } else { Text("Valeur à renseigner").font(.caption).foregroundStyle(.orange) } }
    }
    private func moneyLine(_ name: String, _ amount: Cents, _ currency: String, _ subtitle: String) -> some View {
        HStack { VStack(alignment: .leading, spacing: 4) { Text(name); Text(subtitle).font(.caption).foregroundStyle(.secondary) }; Spacer(); AmountText(cents: amount, currency: currency) }
    }
    private func cacheBudget() { if step == .budget { DiskCache.save(BudgetDraft(amount: budget, currency: currency), key: draftKey) } }
    private func save(skip: Bool) async {
        busy = true; defer { busy = false }
        do { try await advance(skip, parsedBudget, currency); error = nil }
        catch {
            if case APIError.badStatus(409, _) = error { self.error = "Le parcours a changé sur un autre appareil. Les dernières données ont été rechargées ; vérifie puis valide à nouveau." }
            else { self.error = error.localizedDescription }
        }
    }
}

private enum JourneySheet: Identifiable {
    case account, asset, liability, fixed(VariableIncome?), variable(VariableIncome), contract(FinancialContract), rule(CalendarRule), goal(GoalStatus?)
    var id: String { switch self { case .account: "account"; case .asset: "asset"; case .liability: "liability"; case .fixed(let v): "fixed-" + (v?.id ?? "new"); case .variable(let v): "variable-" + v.id; case .contract(let v): "contract-" + v.id; case .rule(let v): "rule-" + v.id; case .goal(let v): "goal-" + (v?.id ?? "new") } }
}

private struct JourneyReviewSections: View {
    let snapshot: FinancialJourneySnapshot
    @Environment(SessionStore.self) private var session
    var body: some View {
        Section("Ton mois type déclaré") {
            ForEach(snapshot.currencyRows) { row in
                VStack(alignment: .leading, spacing: 10) {
                    Text(row.currency).font(.headline)
                    amountRow("Revenus prévus", snapshot.state.reviewed.contains("income") ? row.income : nil, row.currency)
                    amountRow("Charges fixes", snapshot.state.reviewed.contains("expenses") ? row.charges : nil, row.currency)
                    amountRow("Abonnements", snapshot.state.reviewed.contains("subscriptions") ? row.subscriptions : nil, row.currency)
                    amountRow("Budget du quotidien", snapshot.state.reviewed.contains("budget") ? row.daily : nil, row.currency)
                    amountRow("Reste avant projets et imprévus", row.remaining, row.currency)
                }.padding(.vertical, 5)
            }
            Text("Les fréquences sont ramenées à un mois moyen. Seules les séries actives encore en cours comptent, pas les montants ponctuels. Les devises restent séparées. Le reste apparaît après vérification des quatre étapes du budget ; il ne tient pas compte des sommes affectées aux objectifs.")
                .font(.caption).foregroundStyle(.secondary)
        }
        Section("Ce que tu possèdes aujourd’hui") {
            if snapshot.netWorth.complete == false {
                Label("Patrimoine incomplet : certaines valeurs ou conversions manquent.", systemImage: "exclamationmark.circle").foregroundStyle(.orange)
            } else {
                amountRow("Patrimoine net", snapshot.netWorth.net, snapshot.netWorth.currency)
            }
            Text("Ce calcul vient du serveur et des valeurs enregistrées. Un bien immobilier ne peut pas payer directement une facture ; garde une réserve disponible sur un compte ou un livret.")
                .font(.caption).foregroundStyle(.secondary)
        }
        Section("Ta prochaine action utile") {
            if !snapshot.state.skipped.isEmpty {
                Label("Reprends les étapes passées quand tu auras les informations. Le bilan restera partiel jusque-là.", systemImage: "list.number")
            }
            if snapshot.accounts.contains(where: { $0.currentValue == nil && $0.latestValue == nil }) {
                Text("Recopie les soldes manquants depuis tes relevés : un montant inconnu ne signifie pas 0.")
            }
            if snapshot.currencyRows.contains(where: { ($0.remaining?.raw ?? 0) < 0 }) {
                Text("Ton budget déclaré est négatif dans au moins une devise. Vérifie les doublons, puis les dépenses que tu peux réduire avant d’ajouter un projet.")
            }
            if !snapshot.alerts.isEmpty {
                NavigationLink { ContractsView() } label: { Label("Vérifier les tarifs et rappels de contrats", systemImage: "bell.badge") }
            }
            NavigationLink { CalendarView() } label: { Label("Comparer prévisions et opérations reçues", systemImage: "calendar") }
            NavigationLink { InvestmentExplorerView() } label: { Label("Comprendre les placements et leurs risques", systemImage: "book") }
            Text("Avant d’investir, précise ton horizon, tes besoins d’argent disponible et la perte que tu pourrais supporter. Les exemples de placements servent à comparer ; ce bilan seul ne suffit pas à choisir un produit.").font(.caption).foregroundStyle(.secondary)
            Button { session.selectedTab = "assistant" } label: { Label("Poser une question à l’assistant", systemImage: "sparkles") }
        }
    }
    private func amountRow(_ title: String, _ value: Cents?, _ currency: String) -> some View {
        LabeledContent(title) {
            if let value { AmountText(cents: value, currency: currency) }
            else { Text("À compléter").foregroundStyle(.secondary) }
        }
    }
}
