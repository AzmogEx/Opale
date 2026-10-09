import SwiftUI

/// Le formulaire financier appartient au profil connecté, après déverrouillage.
struct FinancialSetupGate: View {
    @Environment(SessionStore.self) private var session
    @State private var checked = false
    @State private var presentation: SetupPresentation?

    private struct SetupPresentation: Identifiable {
        let id = UUID()
        let state: FinancialSetupState
    }

    var body: some View {
        Group {
            if checked || session.isOffline {
                MainTabView()
            } else {
                ProgressView("Préparation de ton espace…")
            }
        }
        .task(id: session.profileKey) {
            let key = session.profileKey
            defer { if key == session.profileKey { checked = true } }
            guard !session.isOffline else { return }
            do {
                let state = try await session.api.fetchFinancialSetup()
                guard !Task.isCancelled, key == session.profileKey else { return }
                if state.shouldPrompt { presentation = SetupPresentation(state: state) }
            } catch {
                // Une panne ou un serveur plus ancien ne bloque pas les fonctions existantes.
                // L'entrée des Réglages expose l'erreur et permet de réessayer.
            }
        }
        .fullScreenCover(item: $presentation) { item in
            FinancialSetupView(initialState: item.state)
                .id(session.profileKey)
        }
    }
}

/// Point d'entrée commun au premier lancement et aux Réglages.
struct FinancialSetupView: View {
    var initialState: FinancialSetupState? = nil
    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss
    @State private var state: FinancialSetupState?
    @State private var errorMessage: String?
    @State private var loading = false

    var body: some View {
        Group {
            if let state {
                if state.status == "completed" {
                    FinancialSetupReceipt(state: state)
                } else if state.canStart {
                    FinancialSetupForm(initialState: state, profileKey: session.profileKey)
                } else {
                    NavigationStack {
                        ContentUnavailableView("Espace de démonstration", systemImage: "sparkles", description: Text("La démonstration est déjà renseignée. Crée ton propre profil pour préparer ta situation de départ."))
                            .toolbar { ToolbarItem(placement: .cancellationAction) { Button("Fermer") { dismiss() } } }
                    }
                }
            } else {
                NavigationStack {
                    Group {
                        if let errorMessage {
                            ContentUnavailableView {
                                Label("Configuration indisponible", systemImage: "wifi.exclamationmark")
                            } description: {
                                Text(errorMessage)
                            } actions: {
                                Button("Réessayer") { Task { await load() } }.disabled(loading)
                            }
                        } else { ProgressView("Chargement de ton brouillon…") }
                    }
                    .navigationTitle("Ma situation de départ")
                    .toolbar { ToolbarItem(placement: .cancellationAction) { Button("Fermer") { dismiss() } } }
                }
            }
        }
        .task {
            if let initialState { state = initialState }
            else { await load() }
        }
    }

    private func load() async {
        let key = session.profileKey
        loading = true
        defer { loading = false }
        do {
            let result = try await session.api.fetchFinancialSetup()
            guard !Task.isCancelled, key == session.profileKey else { return }
            state = result
            errorMessage = nil
        } catch {
            guard !Task.isCancelled, key == session.profileKey else { return }
            if case APIError.badStatus(404, _) = error {
                errorMessage = "Le serveur Opale doit être mis à jour pour proposer ce formulaire. Tes autres fonctionnalités restent accessibles."
            } else { errorMessage = error.localizedDescription }
        }
    }
}

private struct FinancialSetupReceipt: View {
    let state: FinancialSetupState
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            List {
                Section {
                    Label("Ton point de départ est enregistré", systemImage: "checkmark.seal.fill")
                        .font(.headline).foregroundStyle(OpaleTheme.accent)
                    Text("Voici les informations déclarées lors de ta configuration. Pour suivre leur évolution, modifie désormais tes comptes, tes échéances et tes objectifs dans les espaces correspondants.")
                        .font(.subheadline).foregroundStyle(.secondary)
                }
                Section("Budget déclaré au départ") {
                    FinancialSetupBudgetRows(summary: state.draft.summary)
                }
                Section("Retrouver et modifier") {
                    NavigationLink { CalendarView() } label: { Label("Salaire, charges et abonnements", systemImage: "calendar") }
                    NavigationLink { WealthView() } label: { Label("Comptes et soldes", systemImage: "building.columns") }
                    NavigationLink { ProjectionView() } label: { Label("Objectif d’épargne", systemImage: "target") }
                }
            }
            .opaleList()
            .navigationTitle("Ma situation de départ")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar { ToolbarItem(placement: .confirmationAction) { Button("Terminé") { dismiss() } } }
        }
    }
}

struct FinancialSetupBudgetRows: View {
    let summary: FinancialSetupSummary

    var body: some View {
        budgetRow("Revenus mensuels", amount: summary.income)
        budgetRow("Charges fixes / mois", amount: summary.expenses)
        budgetRow("Abonnements / mois", amount: summary.subscriptions)
        budgetRow("Dépenses variables prévues", amount: summary.variableBudgetProvided ? summary.variableBudget : nil)
        if let remaining = summary.remaining {
            LabeledContent(summary.variableBudgetProvided ? "Reste mensuel estimé" : "Reste avant dépenses du quotidien") {
                AmountText(cents: remaining).foregroundStyle(remaining < .zero ? OpaleTheme.loss : OpaleTheme.accent)
            }.font(.headline)
        } else {
            Text("Renseigne tes revenus pour estimer ce qu’il te reste chaque mois.")
                .font(.caption).foregroundStyle(.secondary)
        }
        Text("Estimation à partir de tes déclarations. Les coûts trimestriels et annuels sont répartis sur l’année. Les dépenses oubliées et les imprévus ne sont pas inclus.")
            .font(.caption).foregroundStyle(.secondary)
    }

    private func budgetRow(_ title: String, amount: Cents?) -> some View {
        LabeledContent(title) {
            if let amount { AmountText(cents: amount) }
            else { Text("Non renseigné").foregroundStyle(.secondary) }
        }
    }
}
