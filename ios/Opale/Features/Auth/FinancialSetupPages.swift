import SwiftUI

/// Champs des six pages ; la sauvegarde est pilotée par FinancialSetupForm.
struct FinancialSetupPages: View {
    @Binding var draft: FinancialSetupDraft
    @Binding var step: Int
    @FocusState.Binding var focused: String?
    let accounts: [Asset]
    let accountsError: String?
    let titles: [String]
    let reloadAccounts: () -> Void

    var body: some View {
        switch step {
        case 0: incomePage
        case 1: accountPage
        case 2: expensesPage
        case 3: subscriptionsPage
        case 4: goalPage
        default: reviewPage
        }
    }

    // MARK: - Revenus et compte

    private var incomePage: some View {
        Section {
            Toggle("J’ai un revenu mensuel régulier", isOn: $draft.income.enabled)
                .accessibilityIdentifier("setup.income.enabled")
            if draft.income.enabled {
                moneyField("Salaire net reçu chaque mois (€)", value: $draft.income.amount, id: "setup.income.amount")
                DatePicker("Prochain versement", selection: dateBinding($draft.income.nextDate), in: Date.now..., displayedComponents: .date)
                    .accessibilityIdentifier("setup.income.date")
                Text("Indique le montant qui arrive réellement sur ton compte, après prélèvement à la source. Cette saisie crée une prévision mensuelle.")
                    .font(.caption).foregroundStyle(.secondary)
            } else {
                Text("Tu pourras ajouter des revenus irréguliers ou d’autres rentrées d’argent dans Flux.")
                    .font(.subheadline).foregroundStyle(.secondary)
            }
        } header: { Text("Revenu mensuel") }
    }

    private var accountPage: some View {
        Group {
            Section {
                Toggle("Renseigner mon compte principal", isOn: $draft.account.enabled)
                    .accessibilityIdentifier("setup.account.enabled")
                if draft.account.enabled {
                    if !accounts.isEmpty {
                        Picker("Compte en euros", selection: $draft.account.existingAssetID) {
                            Text("Créer un compte").tag("")
                            ForEach(accounts) { Text($0.name).tag($0.id) }
                        }.accessibilityIdentifier("setup.account.choice")
                    }
                    if draft.account.existingAssetID.isEmpty {
                        TextField("Nom du compte", text: $draft.account.name)
                            .focused($focused, equals: "accountName").accessibilityIdentifier("setup.account.name")
                        TextField("Solde à la date choisie (€), facultatif", text: $draft.account.balance)
                            .keyboardType(.numbersAndPunctuation).focused($focused, equals: "balance")
                            .accessibilityIdentifier("setup.account.balance")
                        if !draft.account.balance.trimmingCharacters(in: .whitespaces).isEmpty {
                            DatePicker("Solde à la fin du", selection: dateBinding($draft.account.balanceDate), in: ...Date.now, displayedComponents: .date)
                        }
                        Text("Tu peux laisser le solde vide. Si tu le renseignes, utilise le solde de clôture à la date choisie ; un découvert se saisit avec un signe −.")
                            .font(.caption).foregroundStyle(.secondary)
                    } else {
                        Text("Le solde de ce compte est conservé. Le salaire et les prélèvements prévus lui seront associés.")
                            .font(.caption).foregroundStyle(.secondary)
                    }
                }
                if let accountsError {
                    Text(accountsError).font(.caption).foregroundStyle(OpaleTheme.loss)
                    Button("Recharger mes comptes", action: reloadAccounts)
                }
            } header: { Text("Compte principal") }
            Section {
                Text("Un compte est nécessaire si tu déclares un salaire, des charges ou des abonnements. Tu pourras ajouter tes livrets, placements et crédits dans Patrimoine.")
                    .font(.caption).foregroundStyle(.secondary)
            }
        }
    }

    // MARK: - Charges et abonnements

    private var expensesPage: some View {
        Group {
            Section {
                Menu("Ajouter une charge", systemImage: "plus.circle") {
                    ForEach(["Loyer", "Électricité", "Gaz", "Eau", "Assurance habitation", "Assurance auto", "Téléphone", "Internet", "Transport", "Autre charge"], id: \.self) { label in
                        Button(label) { addEntry(label, subscription: false) }
                    }
                }.accessibilityIdentifier("setup.add.expense")
                if draft.expenses.isEmpty {
                    Text("Ajoute les charges que tu connais, ou continue pour les renseigner plus tard.")
                        .font(.subheadline).foregroundStyle(.secondary)
                }
            }
            ForEach($draft.expenses) { $entry in entrySection($entry, subscription: false) }
        }
    }

    private var subscriptionsPage: some View {
        Group {
            Section {
                Menu("Ajouter un abonnement", systemImage: "plus.circle") {
                    ForEach(["Netflix", "Spotify", "Apple Music", "iCloud", "Salle de sport", "Presse", "Logiciel", "Autre abonnement"], id: \.self) { label in
                        Button(label) { addEntry(label, subscription: true) }
                    }
                }.accessibilityIdentifier("setup.add.subscription")
                if draft.subscriptions.isEmpty {
                    Text("Renseigne le prix de ton offre actuelle. Aucun tarif n’est prérempli.")
                        .font(.subheadline).foregroundStyle(.secondary)
                } else if let amount = draft.summary.subscriptions {
                    LabeledContent("Équivalent mensuel") { AmountText(cents: amount) }
                }
            }
            ForEach($draft.subscriptions) { $entry in entrySection($entry, subscription: true) }
        }
    }

    private func entrySection(_ entry: Binding<FinancialSetupEntry>, subscription: Bool) -> some View {
        let id = entry.wrappedValue.id
        let prefix = subscription ? "subscription" : "expense"
        return Section {
            TextField("Nom", text: entry.label).focused($focused, equals: id + "label")
                .accessibilityIdentifier("setup.\(prefix).label.\(id)")
            moneyField("Montant prélevé (€)", value: entry.amount, id: "setup.\(prefix).amount.\(id)")
            Picker("Fréquence", selection: entry.frequency) {
                Text("Chaque mois").tag("monthly")
                Text("Chaque trimestre").tag("quarterly")
                Text("Chaque année").tag("yearly")
            }
            DatePicker("Prochain prélèvement", selection: dateBinding(entry.date), in: Date.now..., displayedComponents: .date)
            Button("Retirer", role: .destructive) {
                if subscription { draft.subscriptions.removeAll { $0.id == id } }
                else { draft.expenses.removeAll { $0.id == id } }
            }
        } header: { Text(entry.wrappedValue.label.isEmpty ? "Nouvelle dépense" : entry.wrappedValue.label) }
    }

    // MARK: - Objectif et récapitulatif

    private var goalPage: some View {
        Group {
            Section {
                moneyField("Budget variable mensuel (€), facultatif", value: $draft.variableBudget, id: "setup.variable")
                Text("Courses, sorties, achats occasionnels… une estimation suffit. Elle sert au récapitulatif de ce formulaire.")
                    .font(.caption).foregroundStyle(.secondary)
            } header: { Text("Vie quotidienne") }
            Section {
                Toggle("Créer un premier objectif d’épargne", isOn: $draft.goal.enabled)
                    .accessibilityIdentifier("setup.goal.enabled")
                if draft.goal.enabled {
                    TextField("Nom de ton projet", text: $draft.goal.name)
                        .focused($focused, equals: "goalName").accessibilityIdentifier("setup.goal.name")
                    moneyField("Montant à atteindre (€)", value: $draft.goal.target, id: "setup.goal.target")
                    Text("Par exemple, une réserve de sécurité ou un voyage. Le rythme d’épargne pourra être ajusté dans Projection, après tes premières opérations.")
                        .font(.caption).foregroundStyle(.secondary)
                }
            } header: { Text("Premier projet") }
        }
    }

    private var reviewPage: some View {
        Group {
            Section("Ton mois, en prévision") { FinancialSetupBudgetRows(summary: draft.summary) }
            Section("Ce qui sera enregistré") {
                if draft.account.enabled {
                    Label(draft.account.existingAssetID.isEmpty ? "Compte : \(draft.account.name)" : "Ton compte existant", systemImage: "building.columns")
                    if draft.account.existingAssetID.isEmpty, let balance = Cents.parse(draft.account.balance) {
                        LabeledContent("Solde au \(draft.account.balanceDate)") { AmountText(cents: balance) }
                    }
                }
                if draft.income.enabled { Label("Salaire mensuel à partir du \(draft.income.nextDate)", systemImage: "arrow.down.circle") }
                Label("\(draft.expenses.count) charge(s) fixe(s)", systemImage: "house")
                Label("\(draft.subscriptions.count) abonnement(s)", systemImage: "repeat")
                if draft.goal.enabled { Label("Objectif : \(draft.goal.name)", systemImage: "target") }
                Text("Les prochaines échéances apparaîtront dans ton calendrier. Elles ne modifieront pas ton solde comme si elles avaient déjà été payées.")
                    .font(.caption).foregroundStyle(.secondary)
            }
            Section("Ajuster avant de valider") {
                ForEach(0..<5, id: \.self) { index in
                    Button { step = index } label: { Label(titles[index], systemImage: "pencil") }
                }
            }
        }
    }

    // MARK: - Champs et sauvegarde

    private func moneyField(_ title: String, value: Binding<String>, id: String) -> some View {
        TextField(title, text: value).keyboardType(.decimalPad)
            .focused($focused, equals: id).accessibilityIdentifier(id)
    }

    private func dateBinding(_ value: Binding<String>) -> Binding<Date> {
        Binding(get: { Date.fromOpaleDay(value.wrappedValue) ?? .now }, set: { value.wrappedValue = $0.opaleDayString })
    }

    private func addEntry(_ label: String, subscription: Bool) {
        let entry = FinancialSetupEntry(id: UUID().uuidString, label: label, amount: "", date: Date.now.opaleDayString, frequency: "monthly")
        if subscription { draft.subscriptions.append(entry) }
        else { draft.expenses.append(entry) }
    }

}
