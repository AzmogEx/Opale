import SwiftUI

/// Entry point for server-backed tools; no financial result is calculated by the client.
struct PilotToolsView: View {
    var body: some View {
        List {
            Section("Préparer une décision") {
                NavigationLink { TaxToolView() } label: { Label("Fiscalité & PER", systemImage: "percent") }
                NavigationLink { LoanToolView() } label: { Label("Crédit & capacité", systemImage: "building.columns") }
                NavigationLink { AllocationToolView() } label: { Label("Allocation cible", systemImage: "chart.pie") }
            }
            Section("Suivre et automatiser") {
                NavigationLink { CustomAlertsView() } label: { Label("Alertes personnalisées", systemImage: "bell.badge") }
                NavigationLink { WrappedToolView() } label: { Label("Bilan annuel", systemImage: "calendar") }
                NavigationLink { SnapshotsToolView() } label: { Label("Historique mensuel", systemImage: "clock") }
                NavigationLink { QuotesToolView() } label: { Label("Cours automatiques", systemImage: "chart.line.uptrend.xyaxis") }
                NavigationLink { CategoriesRulesView() } label: { Label("Catégories & règles", systemImage: "tag") }
            }
        }
        .navigationTitle("Pilote automatique").opaleList()
    }
}

struct ToolError: View {
    let message: String?
    var body: some View {
        if let message { Section { Label(message, systemImage: "exclamationmark.triangle").foregroundStyle(.red) } }
    }
}

struct TaxToolView: View {
    @Environment(SessionStore.self) private var session
    @State private var income = ""
    @State private var per = ""
    @State private var perCeiling = ""
    @State private var parts = 10
    @State private var result: APIClient.TaxEstimateResponse?
    @State private var deadlines: [TaxDeadline] = []
    @State private var error: String?
    @State private var busy = false
    var body: some View {
        Form {
            Section("Barème 2026 · revenus 2025") {
                TextField("Revenu (€)", text: $income).keyboardType(.decimalPad)
                Stepper("Parts fiscales : \(String(describing: Decimal(parts) / 10))", value: $parts, in: 10...100, step: 5)
                TextField("Versement PER (€)", text: $per).keyboardType(.decimalPad)
                TextField("Plafond PER disponible sur l’avis (€)", text: $perCeiling).keyboardType(.decimalPad)
                Button(busy ? "Calcul…" : "Estimer") { Task { await calculate() } }
                    .disabled(busy || Cents.parse(income) == nil)
            }
            if let result {
                Section("Estimation indicative") {
                    LabeledContent("Impôt") { AmountText(cents: result.estimate.tax) }
                    LabeledContent("Revenu après impôt") { AmountText(cents: result.estimate.netIncome) }
                    LabeledContent("Taux marginal", value: "\(result.estimate.marginalRateBps / 100) %")
                    if let effect = result.perEffect {
                        LabeledContent("Économie PER") { AmountText(cents: effect.savings) }
                        LabeledContent("Coût net du versement") { AmountText(cents: effect.realCost) }
                    }
                    if let deductible = result.deductible { LabeledContent("Versement déductible retenu") { AmountText(cents: deductible) } }
                    Text(result.disclaimer).font(.caption)
                    Text("Barème \(result.assessmentYear), revenus \(result.incomeYear) · vérifié le \(result.verifiedOn)").font(.caption)
                    if let url = URL(string: result.sourceURL) { Link("Consulter le barème officiel", destination: url) }
                }
            }
            if !deadlines.isEmpty { Section("Repères du calendrier fiscal") { ForEach(deadlines) { d in Text("\(d.day)/\(d.month) · \(d.title)") } } }
            ToolError(message: error)
        }.navigationTitle("Fiscalité & PER")
        .task { do { deadlines = try await session.api.taxDeadlines() } catch { self.error = error.localizedDescription } }
    }
    private func calculate() async {
        guard let income = Cents.parse(income), income.raw >= 0, let per = Cents.parse(per.isEmpty ? "0" : per), per.raw >= 0 else { error = "Montants positifs requis"; return }
        let ceiling = Cents.parse(perCeiling)
        guard per.raw == 0 || (ceiling?.raw ?? -1) >= 0 else { error = "Indique le plafond PER disponible sur ton avis d’impôt."; return }
        busy = true; defer { busy = false }
        do { result = try await session.api.taxEstimate(incomeCents: income.raw, partsTenths: parts, perCents: per.raw, perCeilingCents: ceiling?.raw); error = nil }
        catch { self.error = error.localizedDescription }
    }
}

struct LoanToolView: View {
    @Environment(SessionStore.self) private var session
    @State private var principal = ""
    @State private var payment = ""
    @State private var rate = 350
    @State private var months = 240
    @State private var result: LoanResult?
    @State private var capacity: Cents?
    @State private var error: String?
    @State private var busy = false
    var body: some View {
        Form {
            Section("Hypothèses") {
                TextField("Capital à emprunter (€)", text: $principal).keyboardType(.decimalPad)
                TextField("Mensualité disponible (€)", text: $payment).keyboardType(.decimalPad)
                Stepper("Taux annuel : \(String(describing: Decimal(rate) / 100)) %", value: $rate, in: 0...2500, step: 25)
                Stepper("Durée : \(months) mois", value: $months, in: 12...480, step: 12)
                Button(busy ? "Calcul…" : "Simuler") { Task { await calculate() } }.disabled(busy)
                Text("Taux fixe, hors assurance et frais. La capacité est mathématique ; elle ne constitue pas un accord de crédit.").font(.caption)
            }
            if let capacity { Section("Capacité estimée") { AmountText(cents: capacity) } }
            if let result {
                Section("Coût du crédit") {
                    LabeledContent("Mensualité") { AmountText(cents: result.monthlyPayment) }
                    LabeledContent("Intérêts") { AmountText(cents: result.totalInterest) }
                    LabeledContent("Total payé") { AmountText(cents: result.totalPaid) }
                }
                Section("Amortissement annuel") {
                    ForEach(result.schedule) { row in
                        VStack(alignment: .leading) {
                            Text("Année \(row.year)").font(.headline)
                            LabeledContent("Capital restant") { AmountText(cents: row.remaining) }
                            LabeledContent("Intérêts") { AmountText(cents: row.interest) }
                        }
                    }
                }
            }
            ToolError(message: error)
        }.navigationTitle("Crédit")
    }
    private func calculate() async {
        result = nil; capacity = nil; error = nil
        busy = true; defer { busy = false }
        do {
            if let value = Cents.parse(principal), value.raw > 0 { result = try await session.api.loanSimulate(principalCents: value.raw, annualRateBps: rate, months: months) }
            if let value = Cents.parse(payment), value.raw > 0 { capacity = try await session.api.loanCapacity(monthlyPaymentCents: value.raw, annualRateBps: rate, months: months) }
            if result == nil && capacity == nil { error = "Saisis un capital ou une mensualité positive." } else { error = nil }
        } catch { self.error = error.localizedDescription }
    }
}

struct AllocationToolView: View {
    @Environment(SessionStore.self) private var session
    @State private var result: AllocationStatus?
    @State private var targets: [String: Int] = [:]
    @State private var error: String?
    @State private var busy = false
    private let classes = [("stocks", "Actions"), ("real_estate", "Immobilier"), ("crypto", "Crypto"), ("cash", "Liquidités"), ("other", "Autres")]
    var body: some View {
        Form {
            if let result { Section("Allocation actuelle") {
                ForEach(result.lines) { line in
                    VStack(alignment: .leading) {
                        LabeledContent(line.label, value: "\(Decimal(line.actualBps) / 100) %")
                        LabeledContent("Valeur") { AmountText(cents: line.actual) }
                        if result.hasTargets { LabeledContent("Écart à la cible") { AmountText(cents: line.drift, style: .signedDelta) } }
                    }
                }
            } }
            Section("Cible · total 100 %") {
                ForEach(classes, id: \.0) { item in
                    Stepper("\(item.1) : \(targets[item.0, default: 0]) %", value: Binding(get: { targets[item.0, default: 0] }, set: { targets[item.0] = $0 }), in: 0...100, step: 5)
                }
                Text("Total : \(targets.values.reduce(0,+)) %")
                Button("Enregistrer") { Task { await save() } }.disabled(busy || targets.values.reduce(0,+) != 100)
            }
            ToolError(message: error)
        }.navigationTitle("Allocation").task { await load() }
    }
    private func load() async { do { result = try await session.api.allocation(); for line in result?.lines ?? [] { targets[line.class] = line.targetBps / 100 }; error = nil } catch { self.error = error.localizedDescription } }
    private func save() async { busy = true; defer { busy = false }; do { try await session.api.setAllocation(targetsBps: targets.mapValues { $0 * 100 }); await load() } catch { self.error = error.localizedDescription } }
}

struct CustomAlertsView: View {
    @Environment(SessionStore.self) private var session
    @State private var alerts: [CustomAlert] = []
    @State private var kind = "cash_below"
    @State private var threshold = ""
    @State private var error: String?
    @State private var busy = false
    @State private var editing: CustomAlert?
    var body: some View {
        Form {
            Section("Nouvelle alerte") {
                Picker("Condition", selection: $kind) { ForEach(CustomAlert.kinds, id: \.kind) { Text($0.label).tag($0.kind) } }
                TextField("Seuil (€)", text: $threshold).keyboardType(.decimalPad)
                Button("Créer") { Task { await create() } }.disabled(busy || Cents.parse(threshold) == nil)
            }
            Section("Mes alertes") {
                if alerts.isEmpty { Text("Aucune alerte personnalisée").foregroundStyle(.secondary) }
                ForEach(alerts) { alert in
                    Button { editing = alert } label: {
                        VStack(alignment: .leading) {
                            Text(alert.kindLabel)
                            HStack { AmountText(cents: alert.threshold); Spacer(); Text(alert.enabled ? "Active" : "Désactivée") }
                        }
                    }.buttonStyle(.plain)
                }.onDelete { index in Task { do { for i in index { try await session.api.deleteCustomAlert(id: alerts[i].id) }; await load() } catch { self.error = error.localizedDescription } } }
            }
            ToolError(message: error)
        }.navigationTitle("Alertes").task { await load() }
        .sheet(item: $editing) { alert in AlertEditSheet(alert: alert) { Task { await load() } } }
    }
    private func load() async { do { alerts = try await session.api.customAlerts(); error = nil } catch { self.error = error.localizedDescription } }
    private func create() async { guard let value = Cents.parse(threshold) else { return }; busy = true; defer { busy = false }; do { _ = try await session.api.createCustomAlert(kind: kind, thresholdCents: value.raw); threshold = ""; await load() } catch { self.error = error.localizedDescription } }
}
private struct AlertEditSheet: View {
    let alert: CustomAlert
    var saved: () -> Void
    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss
    @State private var threshold = ""
    @State private var enabled = true
    @State private var error: String?
    @State private var busy = false
    var body: some View {
        NavigationStack { Form {
            Section(alert.kindLabel) { TextField("Seuil (€)", text: $threshold).keyboardType(.decimalPad); Toggle("Active", isOn: $enabled) }
            ToolError(message: error)
        }.navigationTitle("Modifier l'alerte").toolbar {
            ToolbarItem(placement: .cancellationAction) { Button("Annuler") { dismiss() } }
            ToolbarItem(placement: .confirmationAction) { Button("Enregistrer") { Task { await save() } }.disabled(busy || Cents.parse(threshold) == nil) }
        }.onAppear { threshold = "\(alert.threshold.decimalEuros)"; enabled = alert.enabled } }
    }
    private func save() async { guard let value = Cents.parse(threshold) else { return }; busy = true; defer { busy = false }; do { try await session.api.updateCustomAlert(id: alert.id, thresholdCents: value.raw, enabled: enabled); saved(); dismiss() } catch { self.error = error.localizedDescription } }
}

struct WrappedToolView: View {
    @Environment(SessionStore.self) private var session
    @State private var year = Calendar.opale.component(.year, from: .now)
    @State private var result: Wrapped?
    @State private var error: String?
    var body: some View {
        Form {
            Section { Stepper("Année \(String(year))", value: $year, in: 2000...Calendar.opale.component(.year, from: .now)) }
            if let result {
                Section("Ton année") {
                    LabeledContent("Revenus") { AmountText(cents: result.income) }
                    LabeledContent("Dépenses") { AmountText(cents: result.expenses) }
                    LabeledContent("Épargne") { AmountText(cents: result.saved) }
                    LabeledContent("Opérations", value: "\(result.transactionCount)")
                    LabeledContent("Taux d'épargne", value: "\(Decimal(result.savingsRateBps) / 100) %")
                }
                Section("Principaux postes") { ForEach(Array(result.topCategories.enumerated()), id: \.offset) { _, row in LabeledContent(row.name) { AmountText(cents: row.total) } } }
                Section("Marchands") { ForEach(Array(result.topMerchants.enumerated()), id: \.offset) { _, row in LabeledContent(row.name) { AmountText(cents: row.total) } } }
                if let start = result.netWorthStart, let end = result.netWorthEnd { Section("Patrimoine") { LabeledContent("Début") { AmountText(cents: start) }; LabeledContent("Fin") { AmountText(cents: end) } } }
            } else if error == nil { ProgressView() }
            ToolError(message: error)
        }.navigationTitle("Bilan annuel").task(id: year) { result = nil; do { result = try await session.api.wrapped(year: year); error = nil } catch { self.error = error.localizedDescription } }
    }
}

struct QuotesToolView: View {
    @Environment(SessionStore.self) private var session
    struct Metadata: Decodable { let symbol: String; let quantity_micro: Int64; let source: String; let as_of: Date?; let last_error: String?; let enabled: Bool }
    @State private var metadata: Metadata?
    @State private var assets: [Asset] = []
    @State private var assetID = ""
    @State private var symbol = ""
    @State private var quantity = ""
    @State private var error: String?
    @State private var feedback = ""
    @State private var busy = false
    var body: some View {
        Form {
            Section("Associer un actif à un cours") {
                Picker("Actif", selection: $assetID) { Text("Choisir").tag(""); ForEach(assets.filter { !$0.archived && $0.kind == .crypto }) { Text($0.name).tag($0.id) } }
                TextField("Identifiant CoinGecko (ex. bitcoin)", text: $symbol).textInputAutocapitalization(.never).autocorrectionDisabled()
                TextField("Quantité", text: $quantity).keyboardType(.decimalPad)
                Text("La source doit être configurée sur le serveur. Une association concerne la valeur totale de cet actif ; utilise un actif distinct par position.").font(.caption)
                if let metadata {
                    Text("Source : \(metadata.source) · \(metadata.enabled ? "suivi actif" : "manuel")").font(.caption)
                    if let date = metadata.as_of { Text("Cours relevé le \(date.formatted(date: .abbreviated, time: .shortened))").font(.caption) }
                    else { Text("Aucun cours automatique relevé").font(.caption) }
                    if let error = metadata.last_error { Text(error).foregroundStyle(.orange) }
                }
                Button("Désactiver le suivi") { Task { await disable() } }.disabled(busy || assetID.isEmpty || metadata?.enabled != true)
                Button("Enregistrer l'association") { Task { await save() } }.disabled(busy || assetID.isEmpty)
            }
            Section {
                Button(busy ? "Actualisation…" : "Actualiser les cours") { Task { await refresh() } }.disabled(busy)
                if !feedback.isEmpty { Text(feedback) }
            }
            ToolError(message: error)
        }.navigationTitle("Cours automatiques").task(id: assetID) { await loadMetadata() }.task { do { assets = try await session.api.listAssets() } catch { self.error = error.localizedDescription } }
    }
    private func loadMetadata() async {
        metadata = nil
        guard !assetID.isEmpty else { return }
        do { let value: Metadata = try await session.api.request("GET", "/v1/assets/\(assetID)/quote"); metadata = value; symbol = value.symbol; quantity = NSDecimalNumber(decimal: Decimal(value.quantity_micro) / 1_000_000).stringValue; error = nil } catch { self.error = error.localizedDescription }
    }
    private func disable() async { busy = true; defer { busy = false }; do { try await session.api.setAssetQuote(assetID: assetID, symbol: "", quantityMicro: 0); await loadMetadata() } catch { self.error = error.localizedDescription } }
    private func save() async {
        guard quantity.range(of: #"^[0-9]+([.,][0-9]{1,6})?$"#, options: .regularExpression) != nil, let amount = Decimal(string: quantity.replacingOccurrences(of: ",", with: ".")), amount > 0, amount <= 1_000_000_000 else { error = "Quantité positive valide requise"; return }
        let micro = amount * 1_000_000
        let rounded = NSDecimalNumber(decimal: micro)
        guard rounded == rounded.rounding(accordingToBehavior: NSDecimalNumberHandler(roundingMode: .plain, scale: 0, raiseOnExactness: false, raiseOnOverflow: false, raiseOnUnderflow: false, raiseOnDivideByZero: false)) else { error = "Maximum six décimales"; return }
        busy = true; defer { busy = false }
        do { try await session.api.setAssetQuote(assetID: assetID, symbol: symbol.trimmingCharacters(in: .whitespaces), quantityMicro: rounded.int64Value); feedback = "Association enregistrée"; await loadMetadata(); error = nil } catch { self.error = error.localizedDescription }
    }
    private func refresh() async { busy = true; defer { busy = false }; do { try await session.api.refreshQuotes(); session.changed(); feedback = "Actualisation demandée. Consulte les valeurs et dates dans Patrimoine."; error = nil } catch { self.error = error.localizedDescription } }
}


struct SnapshotsToolView: View {
    @Environment(SessionStore.self) private var session
    @State private var snapshots: [MonthlySnapshot] = []
    @State private var error: String?
    @State private var loading = true
    var body: some View {
        List {
            Section { Text("Instantanés observés lors de la première exécution du mois. L’évolution recalculée est sur l’accueil. Une absence ne signifie pas une valeur nulle.").font(.caption) }
            if loading { ProgressView() }
            else if snapshots.isEmpty { ContentUnavailableView("Aucun instantané disponible", systemImage: "clock") }
            ForEach(snapshots, id: \.month) { snapshot in
                Section(snapshot.month.opaleFormatted(.dateTime.month(.wide).year())) {
                    if let recorded = snapshot.recordedAt { Text("Observé le \(recorded.opaleFormatted(.dateTime.day().month(.wide).year().hour().minute()))").font(.caption) }
                    LabeledContent("Actifs") { AmountText(cents: snapshot.assets) }
                    LabeledContent("Dettes") { AmountText(cents: snapshot.liabilities) }
                    LabeledContent("Patrimoine net") { AmountText(cents: snapshot.netWorth) }
                }
            }
            ToolError(message: error)
        }.navigationTitle("Historique mensuel").task { await load() }.refreshable { await load() }
    }
    private func load() async { loading = true; defer { loading = false }; do { snapshots = try await session.api.snapshots(); error = nil } catch { self.error = error.localizedDescription } }
}
