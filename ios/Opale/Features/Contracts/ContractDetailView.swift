import SwiftUI

struct ContractDetailView: View {
    let contractID: String
    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss
    @State private var contract: FinancialContract?
    @State private var prices: [ContractPrice] = []
    @State private var alerts: [ContractAlert] = []
    @State private var editing: FinancialContract?
    @State private var error: String?
    @State private var busy = false
    @State private var confirmDelete = false
    @State private var confirmPrice = false
    private var reminders: [ContractAlert] { alerts.filter { $0.kind != "subscription_price_increase" } }

    var body: some View {
        List {
            if let c = contract {
                summary(c)
                if let observation = c.pendingPrice { priceSection(c, observation) }
                if !reminders.isEmpty { reminderSection }
                engagement(c)
                historySection
                actions(c)
            } else if error == nil { ProgressView() }
            if let error { Section { Text(error).foregroundStyle(OpaleTheme.loss); Button("Recharger") { Task { await load() } } } }
        }.opaleList().navigationTitle(contract?.name ?? "Contrat").navigationBarTitleDisplayMode(.inline)
        .toolbar { if let contract { Button("Modifier") { editing = contract }.disabled(busy) } }
        .sheet(item: $editing) { value in ContractEditSheet(initial: value) }
        .task(id: session.refreshID) { await load() }
        .refreshable { await load() }
        .alert("Utiliser ce nouveau tarif ?", isPresented: $confirmPrice) {
            Button("Confirmer") { Task { await resolve(accept: true) } }; Button("Annuler", role: .cancel) {}
        } message: { Text("Les prochaines prévisions utiliseront ce montant. Les paiements et anciennes prévisions sont conservés.") }
        .alert("Supprimer ce contrat ?", isPresented: $confirmDelete) {
            Button("Supprimer", role: .destructive) { Task { await remove() } }; Button("Annuler", role: .cancel) {}
        } message: { Text("Le suivi et son historique de tarifs seront supprimés. Aucune résiliation n’est envoyée au fournisseur.") }
    }
    private func summary(_ c: FinancialContract) -> some View {
        Section(c.active ? "Contrat actif" : "Contrat arrêté") {
            LabeledContent("Tarif") { AmountText(cents: c.amount, currency: c.currency).fontWeight(.semibold) }
            LabeledContent("Fréquence", value: FinancialTools.frequencyLabel(c.frequency))
            LabeledContent("Prochain prélèvement", value: FinancialTools.dayLabel(c.nextDueDate))
            Text(c.calendarRuleID.isEmpty ? "Suivi uniquement : aucune prévision ajoutée au calendrier." : "Prévision liée au calendrier. Relie chaque paiement à sa transaction pour éviter de le compter deux fois.")
                .font(.caption).foregroundStyle(.secondary)
            NavigationLink("Ouvrir le calendrier") { CalendarView() }
        }
    }
    private func priceSection(_ c: FinancialContract, _ p: ContractPriceObservation) -> some View {
        Section {
            LabeledContent("Ancien tarif") { AmountText(cents: p.previous, currency: c.currency) }
            LabeledContent("Dernier prélèvement") { AmountText(cents: p.amount, currency: c.currency) }
            LabeledContent("Surcoût annuel si confirmé") { AmountText(cents: p.annualDelta, currency: c.currency) }
            Text("Opération du \(FinancialTools.dayLabel(p.date)). Une facture exceptionnelle peut ressembler à une hausse : vérifie avant de confirmer.").font(.caption).foregroundStyle(.secondary)
            Button("Confirmer le nouveau tarif") { confirmPrice = true }.disabled(busy).accessibilityIdentifier("contract.acceptPrice")
            Button("Ignorer ce prélèvement") { Task { await resolve(accept: false) } }.disabled(busy)
        } header: { Label("Hausse à vérifier", systemImage: "arrow.up.right").foregroundStyle(.orange) }
    }
    private var reminderSection: some View {
        Section("Rappels") {
            ForEach(reminders) { reminder in
                Text("\(reminder.title) · \(FinancialTools.dayLabel(reminder.date))")
            }
        }
    }
    private func engagement(_ c: FinancialContract) -> some View {
        Section("Engagement") {
            if !c.trialEnd.isEmpty { LabeledContent("Fin d’essai", value: FinancialTools.dayLabel(c.trialEnd)) }
            if !c.commitmentEnd.isEmpty { LabeledContent("Fin d’engagement", value: FinancialTools.dayLabel(c.commitmentEnd)) }
            if !c.renewalDate.isEmpty { LabeledContent("Renouvellement", value: FinancialTools.dayLabel(c.renewalDate)) }
            LabeledContent("Reconduction automatique", value: c.autoRenew ? "Oui" : "Non")
            LabeledContent("Préavis déclaré", value: "\(c.noticeDays) jours")
            LabeledContent("Rappel avant l’échéance", value: "\(c.reminderDays) jours")
            if !c.merchantKey.isEmpty { LabeledContent("Marchand surveillé", value: c.merchantKey) }
            if !c.note.isEmpty { Text(c.note) }
        }
    }
    private var historySection: some View {
        Section("Historique des tarifs") {
            ForEach(prices) { price in
                LabeledContent {
                    AmountText(cents: price.amount, currency: price.currency)
                } label: {
                    VStack(alignment: .leading) {
                        Text(FinancialTools.dayLabel(price.effectiveOn))
                        Text(price.source == "observed" ? "Prélèvement confirmé" : "Tarif déclaré").font(.caption).foregroundStyle(.secondary)
                    }
                }
            }
        }
    }
    private func actions(_ c: FinancialContract) -> some View {
        Section {
            Button(c.active ? "Arrêter le suivi et les prévisions" : "Réactiver le contrat") { Task { await toggleActive() } }.disabled(busy)
            Text("L’arrêt du suivi ne résilie pas le contrat auprès du fournisseur.").font(.caption).foregroundStyle(.secondary)
            Button("Supprimer le contrat", role: .destructive) { confirmDelete = true }.disabled(busy)
        }
    }
    private func load() async {
        let key = session.profileKey, api = session.api
        do {
            let result = try await api.contracts()
            let history = try await api.contractPrices(contractID)
            guard key == session.profileKey, !Task.isCancelled else { return }
            contract = result.contracts.first { $0.id == contractID }; prices = history
            alerts = result.alerts.filter { $0.contractID == contractID }; error = nil
        } catch { guard key == session.profileKey, !Task.isCancelled else { return }; self.error = error.localizedDescription }
    }
    private func resolve(accept: Bool) async {
        guard let contract else { return }; busy = true; defer { busy = false }
        let key = session.profileKey, api = session.api
        do { try await api.resolvePrice(contract, accept: accept); guard key == session.profileKey else { return }; session.changed() }
        catch { guard key == session.profileKey else { return }; self.error = error.localizedDescription }
    }
    private func toggleActive() async {
        guard var value = contract else { return }; value.active.toggle(); busy = true; defer { busy = false }
        let key = session.profileKey, api = session.api
        do { let _ = try await api.saveContract(value); guard key == session.profileKey else { return }; session.changed() }
        catch { guard key == session.profileKey else { return }; self.error = error.localizedDescription }
    }
    private func remove() async {
        guard let contract else { return }; busy = true; defer { busy = false }
        let key = session.profileKey, api = session.api
        do { try await api.deleteContract(contract); guard key == session.profileKey else { return }; session.changed(); dismiss() }
        catch { guard key == session.profileKey else { return }; self.error = error.localizedDescription }
    }
}
