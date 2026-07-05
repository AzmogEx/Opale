import SwiftUI

/// Objectifs de vie (EF-042) — section de l'onglet Projection.
struct GoalsSection: View {
    @Environment(SessionStore.self) private var session

    @State private var goals: [GoalStatus] = []
    @State private var showAdd = false

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

                ForEach(goals) { goal in
                    GoalRow(goal: goal)
                        .contextMenu {
                            Button(role: .destructive) {
                                Task {
                                    try? await session.api.deleteGoal(id: goal.id)
                                    await load()
                                }
                            } label: {
                                Label("Supprimer", systemImage: "trash")
                            }
                        }
                }
            }
        }
        .task { await load() }
        .sheet(isPresented: $showAdd) {
            GoalFormSheet { Task { await load() } }
                .presentationDetents([.medium])
        }
    }

    private func load() async {
        goals = (try? await session.api.listGoals()) ?? []
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
            ProgressView(value: Double(goal.percent), total: 100)
                .tint(OpaleTheme.accent)
            HStack {
                Text("\(MoneyFormat.eurosWhole(goal.progress)) / \(MoneyFormat.eurosWhole(goal.target))")
                Spacer()
                Text("\(goal.percent) %")
                    .fontWeight(.semibold)
                if let date = goal.targetDate {
                    Text("· \(date.formatted(.dateTime.month(.abbreviated).year()))")
                }
            }
            .font(.caption)
            .foregroundStyle(.secondary)
        }
    }
}

/// Création d'un objectif : nom, montant, échéance, actif source.
private struct GoalFormSheet: View {
    var onSaved: () -> Void

    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss

    @State private var name = ""
    @State private var targetText = ""
    @State private var hasDate = false
    @State private var targetDate = Calendar.current.date(byAdding: .year, value: 2, to: .now)!
    @State private var assetID = ""
    @State private var assets: [Asset] = []
    @State private var errorMessage: String?

    private var parsed: Cents? { Cents.parse(targetText) }
    private var isValid: Bool {
        !name.trimmingCharacters(in: .whitespaces).isEmpty && (parsed?.raw ?? 0) > 0
    }

    var body: some View {
        NavigationStack {
            Form {
                Section("Objectif") {
                    TextField("Nom (ex. Apport immobilier)", text: $name)
                    TextField("Montant cible (ex. 30 000)", text: $targetText)
                        .keyboardType(.decimalPad)
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
            .navigationTitle("Nouvel objectif")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("Créer") { Task { await save() } }.disabled(!isValid)
                }
                ToolbarItem(placement: .cancellationAction) {
                    Button("Annuler") { dismiss() }
                }
            }
            .task {
                assets = (try? await session.api.listAssets()) ?? []
            }
        }
    }

    private func save() async {
        guard let parsed else { return }
        do {
            try await session.api.createGoal(.init(
                name: name.trimmingCharacters(in: .whitespaces),
                icon: "target",
                targetCents: parsed.raw,
                targetDate: hasDate ? targetDate.opaleDayString : "",
                assetID: assetID
            ))
            onSaved()
            dismiss()
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
