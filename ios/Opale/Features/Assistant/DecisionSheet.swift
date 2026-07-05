import SwiftUI

/// Mode Décision (EF-052) : « puis-je me permettre X ? »
/// Le moteur backend projette avec/sans la décision (3 scénarios) ;
/// l'IA explique le verdict quand un niveau de la cascade est disponible.
struct DecisionSheet: View {
    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss

    @State private var label = ""
    @State private var oneTimeText = ""
    @State private var monthlyText = ""
    @State private var result: DecisionResponse?
    @State private var isLoading = false
    @State private var errorMessage: String?

    private var oneTime: Cents? { oneTimeText.isEmpty ? Cents.zero : Cents.parse(oneTimeText) }
    private var monthly: Cents? { monthlyText.isEmpty ? Cents.zero : Cents.parse(monthlyText) }
    private var isValid: Bool {
        !label.trimmingCharacters(in: .whitespaces).isEmpty
            && ((oneTime?.raw ?? 0) != 0 || (monthly?.raw ?? 0) != 0)
    }

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    TextField("Quoi ? (ex. Acheter une voiture)", text: $label)
                    TextField("Coût immédiat (€)", text: $oneTimeText)
                        .keyboardType(.decimalPad)
                    TextField("Charge mensuelle (€)", text: $monthlyText)
                        .keyboardType(.decimalPad)
                } header: {
                    Text("La décision")
                } footer: {
                    Text("Laisse un champ vide s'il ne s'applique pas. Le moteur projette ta trajectoire avec et sans cette décision.")
                }

                if isLoading {
                    Section {
                        HStack {
                            ProgressView()
                            Text("Le moteur projette 10 ans…")
                                .font(.subheadline)
                                .foregroundStyle(.secondary)
                        }
                    }
                }

                if let result {
                    verdictSection(result)
                    scenariosSection(result.impact)
                    narrativeSection(result)
                }

                if let errorMessage {
                    Section { Text(errorMessage).foregroundStyle(OpaleTheme.loss) }
                }
            }
            .navigationTitle("Mode Décision")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("Évaluer") { Task { await evaluate() } }
                        .disabled(!isValid || isLoading)
                }
                ToolbarItem(placement: .cancellationAction) {
                    Button("Fermer") { dismiss() }
                }
            }
        }
    }

    // MARK: - Verdict

    @ViewBuilder
    private func verdictSection(_ result: DecisionResponse) -> some View {
        Section("Verdict") {
            HStack {
                Text("Niveau de risque")
                Spacer()
                Text(result.impact.riskLevel.capitalized)
                    .font(.subheadline.weight(.bold))
                    .padding(.horizontal, 10)
                    .padding(.vertical, 4)
                    .background(riskColor(result.impact.riskLevel).opacity(0.15), in: .capsule)
                    .foregroundStyle(riskColor(result.impact.riskLevel))
            }
            LabeledContent("Payable cash") {
                Image(systemName: result.impact.affordableCash ? "checkmark.circle.fill" : "xmark.circle.fill")
                    .foregroundStyle(result.impact.affordableCash ? OpaleTheme.gain : OpaleTheme.loss)
            }
            LabeledContent("Épargne mensuelle après") {
                AmountText(cents: result.impact.savingsAfter, style: .signedDelta)
                    .foregroundStyle(result.impact.savingsAfter.raw < 0 ? OpaleTheme.loss : OpaleTheme.gain)
            }
        }
    }

    private func riskColor(_ level: String) -> Color {
        switch level {
        case "élevé": OpaleTheme.loss
        case "modéré": .orange
        default: OpaleTheme.gain
        }
    }

    // MARK: - Scénarios

    @ViewBuilder
    private func scenariosSection(_ impact: DecisionImpact) -> some View {
        Section("Impact sur ton patrimoine") {
            ForEach(impact.scenarios) { s in
                VStack(alignment: .leading, spacing: 6) {
                    HStack {
                        Text(s.name.capitalized)
                            .font(.subheadline.weight(.semibold))
                        Text(percentLabel(s.returnBps))
                            .font(.caption)
                            .foregroundStyle(.tertiary)
                        Spacer()
                        if s.baselineReached, s.decisionReached, s.delayMonths != 0 {
                            Text(delayLabel(s.delayMonths))
                                .font(.caption.weight(.semibold))
                                .foregroundStyle(s.delayMonths > 0 ? .orange : OpaleTheme.gain)
                        }
                    }
                    HStack {
                        deltaColumn("À 5 ans", s.delta5y)
                        Divider()
                        deltaColumn("À 10 ans", s.delta10y)
                    }
                }
                .padding(.vertical, 2)
            }
        }
    }

    private func deltaColumn(_ title: String, _ delta: Cents) -> some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(title)
                .font(.caption2)
                .foregroundStyle(.secondary)
            AmountText(cents: delta, style: .signedDelta)
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(delta.raw < 0 ? OpaleTheme.loss : OpaleTheme.gain)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private func percentLabel(_ bps: Int) -> String {
        "\(bps / 100) %/an"
    }

    private func delayLabel(_ months: Int) -> String {
        months > 0 ? "Liberté +\(months) mois" : "Liberté \(months) mois"
    }

    // MARK: - Explication (IA ou moteur)

    @ViewBuilder
    private func narrativeSection(_ result: DecisionResponse) -> some View {
        Section {
            Text(result.narrative)
                .font(.subheadline)
        } header: {
            Text("Explication")
        } footer: {
            Label(tierFooter(result.narrativeTier), systemImage: "lock.shield")
                .font(.caption2)
        }
    }

    private func tierFooter(_ tier: String) -> String {
        switch tier {
        case "n2": "Analyse rédigée sur ton homelab — les données ne l'ont jamais quitté."
        case "n3": "Analyse rédigée par le modèle cloud à partir de données anonymisées."
        default: "Verdict du moteur déterministe — l'IA est hors ligne."
        }
    }

    // MARK: - Évaluation

    private func evaluate() async {
        guard let oneTime, let monthly else {
            errorMessage = "Montants invalides"
            return
        }
        isLoading = true
        errorMessage = nil
        defer { isLoading = false }
        do {
            result = try await session.api.evaluateDecision(.init(
                label: label.trimmingCharacters(in: .whitespaces),
                oneTimeCostCents: oneTime.raw,
                monthlyCostCents: monthly.raw,
                allowCloud: false
            ))
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
