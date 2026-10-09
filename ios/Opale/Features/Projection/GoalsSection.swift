import SwiftUI

/// Objectifs de vie (EF-042) — section de l'onglet Projection.
struct GoalsSection: View {
    @Environment(SessionStore.self) private var session

    @State private var goals: [GoalStatus] = []
    @State private var showAdd = false
    @State private var editing: GoalStatus?
    @State private var errorMessage: String?

    var body: some View {
        GlassCard {
            VStack(alignment: .leading, spacing: 16) {
                HStack {
                    Text("Objectifs")
                        .font(.footnote.weight(.semibold))
                        .foregroundStyle(.secondary)
                        .textCase(.uppercase)
                    Spacer()
                    Button {
                        showAdd = true
                    } label: {
                        Image(systemName: "plus.circle.fill")
                    }
                }

                if goals.isEmpty {
                    Text("Donne un cap à ton épargne : apport immobilier, voyage, retraite…")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }

                if let errorMessage { Text(errorMessage).foregroundStyle(.red) }
                ForEach(goals) { goal in
                    GoalRow(goal: goal)
                        .onTapGesture { editing = goal }
                        .contextMenu {
                            Button(role: .destructive) {
                                Task {
                                    do { try await session.api.deleteGoal(id: goal.id); session.changed(); await load() } catch { errorMessage = error.localizedDescription }
                                }
                            } label: {
                                Label("Supprimer", systemImage: "trash")
                            }
                        }
                }
            }
        }
        .task(id: session.refreshID) { await load() }
        .sheet(item: $editing) { goal in GoalFormSheet(existing: goal) { Task { await load() } } }
        .sheet(isPresented: $showAdd) {
            GoalFormSheet { Task { await load() } }
                .presentationDetents([.medium])
        }
    }

    private func load() async {
        do { goals = try await session.api.listGoals(); errorMessage = nil } catch { errorMessage = error.localizedDescription }
    }
}

/// Un objectif : jauge de progression + trajectoire.
private struct GoalRow: View {
    let goal: GoalStatus

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack {
                Label(goal.name, systemImage: goal.icon)
                    .font(.body.weight(.medium))
                Spacer()
                if let onTrack = goal.onTrack {
                    Text(onTrack ? "En avance" : "En retard")
                        .font(.caption2.weight(.bold))
                        .padding(.horizontal, 8)
                        .padding(.vertical, 3)
                        .background(
                            (onTrack ? OpaleTheme.gain : OpaleTheme.loss).opacity(0.15),
                            in: .capsule
                        )
                        .foregroundStyle(onTrack ? OpaleTheme.gain : OpaleTheme.loss)
                }
            }
            if let remaining = goal.remaining { LabeledContent("Reste à épargner") { AmountText(cents: remaining) } }
            if let date = goal.estimatedDate { Text("Atteinte estimée : \(date.opaleFormatted(.dateTime.month(.wide).year()))") }
            else if let reason = goal.estimateReason { Text(reason).font(.caption).foregroundStyle(.secondary) }
            ProgressView(value: Double(min(100, max(0, goal.percent))), total: 100)
                .tint(OpaleTheme.accent)
            HStack {
                Text("\(MoneyFormat.eurosWhole(goal.progress)) / \(MoneyFormat.eurosWhole(goal.target))")
                    .sensitive()
                Spacer()
                Text("\(goal.percent) %")
                    .fontWeight(.semibold)
                if let date = goal.targetDate {
                    Text("· \(date.opaleFormatted(.dateTime.month(.abbreviated).year()))")
                }
            }
            .font(.caption)
            .foregroundStyle(.secondary)
        }
    }
}

/// Création d'un objectif : nom, montant, échéance, actif source.
struct GoalFormSheet: View {
    var existing: GoalStatus? = nil
    var onSaved: () -> Void

    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss

    @State private var name = ""
    @State private var targetText = ""
    @State private var savingsText = ""
    @State private var isSaving = false
    @State private var hasDate = false
    @State private var targetDate = Calendar.opale.date(byAdding: .year, value: 2, to: .now)!
    @State private var assetID = ""
    @State private var assets: [Asset] = []
    @State private var errorMessage: String?

    private var parsed: Cents? { Cents.parse(targetText) }
    private var isValid: Bool {
        !name.trimmingCharacters(in: .whitespaces).isEmpty && (parsed?.raw ?? 0) > 0 && (Cents.parse(savingsText.isEmpty ? "0" : savingsText)?.raw ?? -1) >= 0
    }

    var body: some View {
        NavigationStack {
            Form {
                Section("Objectif") {
                    TextField("Nom (ex. Apport immobilier)", text: $name)
                    TextField("Montant cible (ex. 30 000)", text: $targetText)
                        .keyboardType(.decimalPad)
                }
                Section("Épargne affectée") {
                    TextField("Montant par mois (€)", text: $savingsText).keyboardType(.decimalPad)
                    Text("Ce rythme est réservé à cet objectif. Le serveur contrôle la capacité disponible.").font(.caption)
                }
                Section("Échéance (optionnel)") {
                    Toggle("Date cible", isOn: $hasDate)
                    if hasDate {
                        DatePicker("Échéance", selection: $targetDate,
                                   in: Date.now..., displayedComponents: .date)
                    }
                }
                Section {
                    Picker("Suivi sur", selection: $assetID) {
                        Text("Patrimoine net").tag("")
                        ForEach(assets) { a in
                            Text(a.name).tag(a.id)
                        }
                    }
                } footer: {
                    Text("La progression est mesurée sur cet actif (ex. le livret dédié), ou sur ton patrimoine net global.")
                }
                if let errorMessage {
                    Text(errorMessage).foregroundStyle(OpaleTheme.loss)
                }
            }
            .navigationTitle(existing == nil ? "Nouvel objectif" : "Modifier l’objectif")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("Enregistrer") { Task { await save() } }.disabled(!isValid || isSaving)
                }
                ToolbarItem(placement: .cancellationAction) {
                    Button("Annuler") { dismiss() }
                }
            }
            .task {
                do { assets = try await session.api.listAssets() } catch { errorMessage = error.localizedDescription }
                if let existing { name = existing.name; targetText = "\(existing.target.decimalEuros)"; savingsText = "\((existing.monthlySavings ?? .zero).decimalEuros)"; hasDate = existing.targetDate != nil; targetDate = existing.targetDate ?? .now; assetID = existing.assetID ?? "" }
            }
        }
    }

    private func save() async {
        guard let parsed else { return }
        isSaving = true; defer { isSaving = false }
        do {
            let req = APIClient.CreateGoalRequest(name: name.trimmingCharacters(in: .whitespaces), icon: existing?.icon ?? "target", targetCents: parsed.raw, targetDate: hasDate ? targetDate.opaleDayString : "", assetID: assetID, monthlySavingsCents: Cents.parse(savingsText.isEmpty ? "0" : savingsText)?.raw ?? 0)
            if let existing { try await session.api.updateGoal(id: existing.id, request: req) }
            else { try await session.api.createGoal(req) }
            session.changed()
            onSaved()
            dismiss()
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
