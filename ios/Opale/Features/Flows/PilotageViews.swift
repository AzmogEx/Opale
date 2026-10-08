import SwiftUI

// MARK: - Enveloppes budgétaires (EF-028)

struct EnvelopesView: View {
    @Environment(SessionStore.self) private var session

    @State private var statuses: [EnvelopeStatus] = []
    @State private var categories: [Category] = []
    @State private var showAdd = false
    @State private var editing: EnvelopeStatus?
    @State private var errorMessage: String?

    var body: some View {
        List {
            if statuses.isEmpty {
                ContentUnavailableView(
                    "Aucune enveloppe",
                    systemImage: "envelope",
                    description: Text("Alloue un budget mensuel à une catégorie — chaque euro a un job.")
                )
                .listRowBackground(Color.clear)
            }
            ForEach(statuses) { st in
                EnvelopeRow(status: st).onTapGesture { editing = st }
            }
            .onDelete { indexSet in
                Task {
                    do { for i in indexSet { try await session.api.deleteEnvelope(id: statuses[i].id) }; session.changed(); await load() } catch { errorMessage = error.localizedDescription }
                }
            }
            if let errorMessage {
                Text(errorMessage).foregroundStyle(OpaleTheme.loss)
            }
            Section {
                Button {
                    showAdd = true
                } label: {
                    Label("Nouvelle enveloppe", systemImage: "plus.circle.fill")
                }
            }
        }
        .task { await load() }
        .scrollContentBackground(.hidden)
        .refreshable { await load() }
        .sheet(item: $editing) { item in EnvelopeFormSheet(categories: categories, existing: item) { Task { await load() } } }
        .sheet(isPresented: $showAdd) {
            EnvelopeFormSheet(
                categories: categories.filter { c in !statuses.contains { $0.categoryID == c.id } }
            ) {
                Task { await load() }
            }
            .presentationDetents([.medium])
        }
    }

    private func load() async {
        do {
            let comps = Calendar.opale.dateComponents([.year, .month], from: .now)
            statuses = try await session.api.envelopeStatuses(year: comps.year!, month: comps.month!)
            if categories.isEmpty {
                categories = try await session.api.listCategories()
            }
            errorMessage = nil
        } catch is CancellationError {
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}

/// Jauge d'enveloppe : consommation du mois vs budget.
private struct EnvelopeRow: View {
    let status: EnvelopeStatus

    private var overrun: Bool { status.remaining.raw < 0 }
    private var fraction: Double {
        guard status.budget.raw > 0 else { return 0 }
        return min(Double(status.spent.raw) / Double(status.budget.raw), 1)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Label(status.categoryName, systemImage: status.categoryIcon)
                    .font(.body.weight(.medium))
                Spacer()
                AmountText(cents: status.remaining, style: .whole)
                    .font(.callout.weight(.semibold))
                    .foregroundStyle(overrun ? OpaleTheme.loss : OpaleTheme.gain)
            }
            ProgressView(value: fraction)
                .tint(overrun ? OpaleTheme.loss : OpaleTheme.accent)
            HStack {
                Text("\(MoneyFormat.eurosWhole(status.spent)) dépensés")
                    .sensitive()
                Spacer()
                Text("sur \(MoneyFormat.eurosWhole(status.budget))")
                    .sensitive()
            }
            .font(.caption)
            .foregroundStyle(.secondary)
        }
        .padding(.vertical, 4)
    }
}

/// Création d'une enveloppe : catégorie + budget mensuel.
private struct EnvelopeFormSheet: View {
    let categories: [Category]
    var existing: EnvelopeStatus? = nil
    var onSaved: () -> Void

    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss

    @State private var categoryID = ""
    @State private var budgetText = ""
    @State private var busy = false
    @State private var errorMessage: String?

    private var parsed: Cents? { Cents.parse(budgetText) }
    private var isValid: Bool { !categoryID.isEmpty && (parsed?.raw ?? 0) > 0 }

    var body: some View {
        NavigationStack {
            Form {
                Picker("Catégorie", selection: $categoryID) {
                    Text("Choisir…").tag("")
                    ForEach(categories) { c in
                        Label(c.name, systemImage: c.icon).tag(c.id)
                    }
                }
                .disabled(existing != nil)
                TextField("Budget mensuel (ex. 250)", text: $budgetText)
                    .keyboardType(.decimalPad)
                if let errorMessage {
                    Text(errorMessage).foregroundStyle(OpaleTheme.loss)
                }
            }
            .navigationTitle(existing == nil ? "Nouvelle enveloppe" : "Modifier l’enveloppe")
            .onAppear { if let existing { categoryID = existing.categoryID; budgetText = MoneyFormat.input(existing.budget, currency: "EUR") } }
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("Enregistrer") { Task { await save() } }.disabled(!isValid || busy)
                }
                ToolbarItem(placement: .cancellationAction) {
                    Button("Annuler") { dismiss() }
                }
            }
        }
    }

    private func save() async {
        guard let parsed, !busy else { return }
        busy = true; defer { busy = false }
        do {
            try await session.api.upsertEnvelope(categoryID: categoryID, budgetCents: parsed.raw)
            session.changed()
            onSaved()
            dismiss()
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}

// MARK: - À venir : cashflow futur + abonnements (EF-025/026/027)

struct UpcomingView: View {
    @Environment(SessionStore.self) private var session

    @State private var projection: CashProjection?
    @State private var recurring: [RecurringFlow] = []
    @State private var errorMessage: String?

    @State private var horizon = 30
    @State private var confirming: RecurringFlow?

    var body: some View {
        List {
            Section {
                NavigationLink("Gérer le calendrier") { CalendarView() }
                Stepper("Horizon : \(horizon) jours", value: $horizon, in: 1...365, step: 7)
            }
            if let projection {
                Section {
                    cashflowCard(projection)
                        .listRowInsets(EdgeInsets())
                        .listRowBackground(Color.clear)
                }

                Section("Échéances des \(horizon) prochains jours") {
                    if projection.upcoming.isEmpty {
                        Text("Aucune échéance détectée — importe plus d'historique.")
                            .foregroundStyle(.secondary)
                    }
                    ForEach(projection.upcoming) { flow in
                        HStack {
                            Text(flow.date.opaleFormatted(.dateTime.day().month(.abbreviated)))
                                .font(.callout.weight(.semibold))
                                .frame(width: 64, alignment: .leading)
                                .foregroundStyle(.secondary)
                            Text(flow.label)
                            Spacer()
                            AmountText(cents: flow.amount, style: .full)
                                .font(.callout.weight(.medium))
                                .foregroundStyle(flow.amount.raw > 0 ? OpaleTheme.gain : .primary)
                        }
                    }
                }
            }

            Section("Abonnements & flux récurrents") {
                let active = recurring.filter { $0.active && $0.amount.raw < 0 }
                if active.isEmpty {
                    Text("Aucun abonnement détecté.")
                        .foregroundStyle(.secondary)
                }
                ForEach(active) { flow in
                    VStack(alignment: .leading, spacing: 2) {
                        HStack {
                            Text(flow.label).font(.body.weight(.medium))
                            Spacer()
                            AmountText(cents: flow.amount, style: .full)
                                .font(.callout.weight(.semibold))
                        }
                        HStack {
                            Button("Confirmer") { confirming = flow }
                            Button("Exclure") { Task { await exclude(flow) } }
                        }.buttonStyle(.borderless)
                        Text("\(flow.periodicityLabel) — prochain le \(flow.nextDate.opaleFormatted(.dateTime.day().month(.wide)))")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }
            }

            if let errorMessage {
                Text(errorMessage).foregroundStyle(OpaleTheme.loss)
            }
        }
        .task(id: horizon) { await load() }
        .sheet(item: $confirming) { detected in CalendarRuleSheet(detected: detected) { Task { await load() } } }
        .scrollContentBackground(.hidden)
        .refreshable { await load() }
    }

    private func exclude(_ flow: RecurringFlow) async {
        struct Body: Encodable { let merchant_key: String; let excluded: Bool }
        do { let _: APIClient.EmptyResponse = try await session.api.request("PUT", "/v1/recurring/exclusion", body: Body(merchant_key: flow.merchantKey, excluded: true)); await load() }
        catch { errorMessage = error.localizedDescription }
    }

    private func cashflowCard(_ p: CashProjection) -> some View {
        GlassCard {
            VStack(alignment: .leading, spacing: 8) {
                Text("Cash prévu au \(p.until.opaleFormatted(.dateTime.day().month(.wide)))")
                    .font(.footnote.weight(.semibold))
                    .foregroundStyle(.secondary)
                    .textCase(.uppercase)
                HStack(alignment: .firstTextBaseline, spacing: 12) {
                    AmountText(cents: p.endCash, style: .whole)
                        .font(.system(size: 32, weight: .bold, design: .rounded))
                        .foregroundStyle(p.endCash.raw < 0 ? AnyShapeStyle(OpaleTheme.loss) : AnyShapeStyle(OpaleTheme.iridescent))
                    Text("aujourd'hui : \(MoneyFormat.eurosWhole(p.startCash))")
                        .sensitive()
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }
        }
    }

    private func load() async {
        do {
            async let proj = session.api.cashflow(days: horizon)
            async let rec = session.api.recurringFlows()
            projection = try await proj
            recurring = try await rec
            errorMessage = nil
        } catch is CancellationError {
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
