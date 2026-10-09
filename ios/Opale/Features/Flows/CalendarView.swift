import SwiftUI

struct CalendarRule: Codable, Identifiable {
    var id: String
    var asset_id: String
    var label: String
    var amount_cents: Cents
    var date: String
    var frequency: String
    var end_date: String?
    var active: Bool
    var merchant_key: String
    var managed_by: String?
    var managed_id: String?
}
struct CalendarOccurrence: Codable, Identifiable {
    var rule_id: String
    var asset_id: String
    var label: String
    var date: String
    var amount_cents: Cents
    var currency: String
    var status: String
    var transaction_id: String?
    var id: String { rule_id + date }
}
struct CalendarData: Decodable { var rules: [CalendarRule]; var occurrences: [CalendarOccurrence] }
struct CalendarView: View {
    @Environment(SessionStore.self) private var session
    @State private var data: CalendarData?
    @State private var error: String?
    @State private var showAdd = false
    @State private var editing: CalendarRule?
    @State private var occurrence: CalendarOccurrence?
    @State private var until = Calendar.opale.date(byAdding: .month, value: 3, to: .now)!
    var body: some View {
        List {
            Section { DatePicker("Prévoir jusqu'au", selection: $until, in: Date.now...Calendar.opale.date(byAdding: .year, value: 2, to: .now)!, displayedComponents: .date) }
            if let data {
                Section("Échéances") {
                    if data.occurrences.isEmpty { Text("Aucune échéance sur cette période") }
                    ForEach(data.occurrences) { item in
                        Button { occurrence = item } label: {
                            VStack(alignment: .leading) {
                                LabeledContent(item.label) { AmountText(cents: item.amount_cents, currency: item.currency) }
                                Text("\(item.date) · \(item.status == "realized" ? "Réalisé" : item.status == "excluded" ? "Exclu" : "Prévu")").font(.caption).foregroundStyle(.secondary)
                            }
                        }.buttonStyle(.plain)
                    }
                }
                Section("Séries & échéances ponctuelles") {
                    ForEach(data.rules) { rule in
                        Group {
                            if rule.managed_by == "contract", let id = rule.managed_id {
                                NavigationLink { ContractDetailView(contractID: id) } label: { Label(rule.label + " · contrat", systemImage: "doc.text") }
                            } else if rule.managed_by == "income" {
                                NavigationLink { VariableIncomesView() } label: { Label(rule.label + " · revenu variable", systemImage: "waveform.path") }
                            } else {
                                Button { editing = rule } label: { Label(rule.label + (rule.active ? "" : " · arrêtée"), systemImage: rule.frequency == "once" ? "calendar" : "repeat") }
                            }
                        }.deleteDisabled(!(rule.managed_by ?? "").isEmpty)
                    }.onDelete { indices in Task { do { for i in indices { let _: APIClient.EmptyResponse = try await session.api.request("DELETE", "/v1/calendar/\(data.rules[i].id)") }; await load(); session.changed() } catch { self.error = error.localizedDescription } } }
                }
            } else if error == nil { ProgressView() }
            ToolError(message: error)
        }.navigationTitle("Calendrier")
        .toolbar { Button("Ajouter", systemImage: "plus") { showAdd = true } }
        .task(id: until) { await load() }
        .sheet(isPresented: $showAdd) { CalendarRuleSheet { Task { await load() } } }
        .sheet(item: $editing) { rule in CalendarRuleSheet(existing: rule) { Task { await load() } } }
        .sheet(item: $occurrence) { item in CalendarOccurrenceSheet(item: item) { Task { await load() } } }
    }
    private func load() async {
        do { data = try await session.api.request("GET", "/v1/calendar", query: [URLQueryItem(name: "from", value: Date.now.opaleDayString), URLQueryItem(name: "until", value: until.opaleDayString)]); error = nil }
        catch { self.error = error.localizedDescription }
    }
}
struct CalendarRuleSheet: View {
    var existing: CalendarRule?
    var detected: RecurringFlow?
    var saved: () -> Void
    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss
    @State private var assets: [Asset] = []
    @State private var account = ""
    @State private var label = ""
    @State private var amount = ""
    @State private var date = Date.now
    @State private var end = Date.now
    @State private var hasEnd = false
    @State private var active = true
    @State private var frequency = "once"
    @State private var error: String?
    @State private var busy = false
    private var currency: String { assets.first { $0.id == account }?.currency ?? "EUR" }
    var body: some View {
        NavigationStack { Form {
            Section("Échéance") {
                TextField("Libellé", text: $label)
                Picker("Compte", selection: $account) { Text("Choisir").tag(""); ForEach(assets) { Text($0.name).tag($0.id) } }
                TextField("Montant signé en \(currency) (négatif = sortie)", text: $amount).keyboardType(.numbersAndPunctuation)
                DatePicker("Début", selection: $date, displayedComponents: .date)
                Picker("Fréquence", selection: $frequency) { Text("Choisir").tag(""); Text("Une fois").tag("once"); Text("Semaine").tag("weekly"); Text("Mois").tag("monthly"); Text("Trimestre").tag("quarterly"); Text("Année").tag("yearly") }
                Toggle("Date de fin", isOn: $hasEnd)
                if hasEnd { DatePicker("Fin", selection: $end, in: date..., displayedComponents: .date) }
                Toggle("Active", isOn: $active)
            }
            if let detected {
                Text("Détection : \(detected.periodicityLabel), \(MoneyFormat.euros(detected.amount)). Vérifie la fréquence ; pour un compte non EUR, ressaisis le montant dans sa devise.").font(.caption).sensitive()
            }
            Text("La modification porte sur la série. Pour une seule occurrence, ouvre-la dans le calendrier. Les jours absents d'un mois sont reportés à sa fin.").font(.caption)
            ToolError(message: error)
        }.navigationTitle(existing == nil ? "Nouvelle échéance" : "Modifier la série")
        .toolbar {
            ToolbarItem(placement: .cancellationAction) { Button("Annuler") { dismiss() } }
            ToolbarItem(placement: .confirmationAction) { Button("Enregistrer") { Task { await save() } }.disabled(busy || account.isEmpty || label.isEmpty || frequency.isEmpty) }
        }.task {
            do { assets = try await session.api.listAssets() } catch { self.error = error.localizedDescription }
            if let rule = existing { account = rule.asset_id; label = rule.label; amount = MoneyFormat.input(rule.amount_cents, currency: assets.first { $0.id == account }?.currency ?? "EUR"); date = Date.fromOpaleDay(rule.date) ?? .now; frequency = rule.frequency; active = rule.active; hasEnd = rule.end_date != nil; end = rule.end_date.flatMap(Date.fromOpaleDay) ?? .now }
            if let detected { label = detected.label; amount = "\(detected.amount.decimalEuros)"; date = detected.nextDate; frequency = ["weekly", "monthly", "quarterly", "yearly"].contains(detected.periodicity) ? detected.periodicity : "" }
        }
        .onChange(of: account) { old, new in
            guard detected != nil else { return }
            let previousCurrency = assets.first { $0.id == old }?.currency ?? "EUR"
            let nextCurrency = assets.first { $0.id == new }?.currency ?? "EUR"
            if previousCurrency != nextCurrency { amount = "" }
        } }
    }
    private func save() async {
        guard !frequency.isEmpty else { error = "Choisis explicitement une fréquence prise en charge"; return }
        guard let value = Cents.parse(amount, currency: currency), value.raw != 0 else { error = "Montant non nul valide requis"; return }
        busy = true; defer { busy = false }
        let rule = CalendarRule(id: existing?.id ?? "", asset_id: account, label: label, amount_cents: value, date: date.opaleDayString, frequency: frequency, end_date: hasEnd ? end.opaleDayString : nil, active: active, merchant_key: existing?.merchant_key ?? detected?.merchantKey ?? "")
        do { let _: APIClient.EmptyResponse = try await session.api.request(existing == nil ? "POST" : "PATCH", "/v1/calendar" + (existing.map { "/\($0.id)" } ?? ""), body: rule); session.changed(); saved(); dismiss() } catch { self.error = error.localizedDescription }
    }
}
private struct CalendarOccurrenceSheet: View {
    let item: CalendarOccurrence
    var saved: () -> Void
    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss
    @State private var state = "planned"
    @State private var amount = ""
    @State private var transactionID = ""
    @State private var transactions: [Transaction] = []
    @State private var search = ""
    @State private var loadingTransactions = false
    @State private var moreTransactions = false
    @State private var error: String?
    @State private var busy = false
    var body: some View {
        NavigationStack { Form {
            Section(item.label + " · " + item.date) {
                Picker("État", selection: $state) { Text("Prévu").tag("planned"); Text("Exclu").tag("excluded"); Text("Réalisé").tag("realized") }
                TextField("Montant pour cette occurrence", text: $amount).keyboardType(.numbersAndPunctuation)
                if state == "realized" {
                    TextField("Rechercher une transaction", text: $search)
                    Picker("Transaction réalisée", selection: $transactionID) {
                        Text("Choisir").tag("")
                        if !transactionID.isEmpty, !transactions.contains(where: { $0.id == transactionID }) { Text("Transaction actuellement liée").tag(transactionID) }
                        ForEach(transactions) { Text($0.label + " · " + $0.occurredOn.opaleDayString).tag($0.id) }
                    }
                    if moreTransactions { Button("Charger d’autres transactions") { Task { await loadTransactions(reset: false) } }.disabled(loadingTransactions) }
                    if loadingTransactions { ProgressView() }
                }
                Text("Relier une transaction empêche de compter à nouveau cette échéance dans les prévisions.").font(.caption)
            }
            ToolError(message: error)
        }.navigationTitle("Occurrence").toolbar {
            ToolbarItem(placement: .cancellationAction) { Button("Annuler") { dismiss() } }
            ToolbarItem(placement: .confirmationAction) { Button("Enregistrer") { Task { await save() } }.disabled(busy || (state == "realized" && transactionID.isEmpty)) }
        }.onAppear { state = item.status; amount = MoneyFormat.input(item.amount_cents, currency: item.currency); transactionID = item.transaction_id ?? "" }
        .task(id: search) { await loadTransactions(reset: true) } }
    }
    private func loadTransactions(reset: Bool) async {
        let requestedSearch = search
        loadingTransactions = true
        defer { loadingTransactions = false }
        do {
            let page = try await session.api.listTransactions(query: requestedSearch, assetID: item.asset_id, offset: reset ? 0 : transactions.count)
            guard requestedSearch == search, !Task.isCancelled else { return }
            transactions = reset ? page : transactions + page
            moreTransactions = page.count == 100
            error = nil
        } catch { if !Task.isCancelled { self.error = error.localizedDescription } }

    }
    private func save() async {
        struct Body: Encodable { let date: String; let status: String; let amount_cents: Int64?; let transaction_id: String? }
        guard let value = Cents.parse(amount, currency: item.currency), value.raw != 0 else { error = "Montant non nul valide requis"; return }
        busy = true; defer { busy = false }
        do { let _: APIClient.EmptyResponse = try await session.api.request("PUT", "/v1/calendar/\(item.rule_id)/occurrences", body: Body(date: item.date, status: state, amount_cents: value.raw, transaction_id: state == "realized" ? transactionID : nil)); session.changed(); saved(); dismiss() } catch { self.error = error.localizedDescription }
    }
}
