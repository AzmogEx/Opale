import SwiftUI

/// Centre immobilier (EF-033) : chaque bien avec ses indicateurs calculés
/// par le moteur — rendement brut, cashflow, plus-value, part possédée.
struct RealEstateView: View {
    @Environment(SessionStore.self) private var session

    @State private var properties: [PropertyStatus] = []
    @State private var liabilities: [Liability] = []
    @State private var editing: PropertyStatus?
    @State private var loaded = false
    @State private var errorMessage: String?

    var body: some View {
        List {
            if let errorMessage { Text(errorMessage).foregroundStyle(.red) }
            if loaded && properties.isEmpty {
                ContentUnavailableView(
                    "Aucun bien immobilier",
                    systemImage: "house",
                    description: Text("Ajoute un actif de type « Immobilier » dans Patrimoine, il apparaîtra ici.")
                )
            }
            ForEach(properties) { property in
                Section(property.asset.name) {
                    propertyCard(property)
                    Button {
                        editing = property
                    } label: {
                        Label(property.details.purchasePrice.raw > 0 ? "Modifier les détails" : "Renseigner les détails (loyer, crédit, taxe…)",
                              systemImage: "square.and.pencil")
                            .font(.subheadline)
                    }
                }
            }
        }
        .opaleList()
        .navigationTitle("Immobilier")
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
        .refreshable { await load() }
        .sheet(item: $editing) { property in
            PropertyFormSheet(property: property, liabilities: liabilities) {
                Task { await load() }
            }
        }
    }

    @ViewBuilder
    private func propertyCard(_ p: PropertyStatus) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                VStack(alignment: .leading, spacing: 2) {
                    Text("Valeur estimée")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    Group { if let value = p.asset.latestValue { AmountText(cents: value, style: .whole, currency: p.asset.currency) } else { Text("Valeur non renseignée") } }
                        .font(.title2.weight(.bold))
                }
                Spacer()
                if p.details.purchasePrice.raw > 0 {
                    VStack(alignment: .trailing, spacing: 2) {
                        Text("Plus-value")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                        AmountText(cents: p.capitalGain, style: .signedDelta, currency: p.asset.currency)
                            .font(.headline)
                            .foregroundStyle(p.capitalGain.raw < 0 ? OpaleTheme.loss : OpaleTheme.gain)
                    }
                }
            }

            if p.details.purchasePrice.raw > 0 {
                Grid(alignment: .leading, horizontalSpacing: 16, verticalSpacing: 8) {
                    GridRow {
                        indicator("Rendement brut", percentLabel(p.grossYieldBps))
                        indicator("Cashflow/mois", MoneyFormat.amount(p.monthlyCashflow, currency: p.asset.currency),
                                  color: p.monthlyCashflow.raw < 0 ? OpaleTheme.loss : OpaleTheme.gain)
                    }
                    GridRow {
                        indicator("Part possédée", MoneyFormat.amount(p.equity, currency: p.asset.currency))
                        if let remaining = p.loanRemaining {
                            indicator("Crédit restant", MoneyFormat.amount(remaining, currency: p.asset.currency), color: OpaleTheme.loss)
                        } else {
                            indicator("Crédit", "Aucun")
                        }
                    }
                }
            } else {
                Text("Renseigne le prix d'achat et le loyer pour voir rendement, cashflow et plus-value.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .padding(.vertical, 4)
    }

    private func indicator(_ title: String, _ value: String, color: Color = .primary) -> some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(title).font(.caption2).foregroundStyle(.secondary)
            Text(value).sensitive().font(.subheadline.weight(.semibold)).foregroundStyle(color)
        }
        .gridColumnAlignment(.leading)
    }

    private func percentLabel(_ bps: Int) -> String {
        String(format: "%d,%02d %%", bps / 100, bps % 100)
    }

    private func load() async {
        do { properties = try await session.api.realEstate(); liabilities = try await session.api.listLiabilities(); errorMessage = nil } catch { errorMessage = error.localizedDescription }
        loaded = true
    }
}

/// Saisie des détails d'un bien : achat, loyer, charges, taxe, crédit.
private struct PropertyFormSheet: View {
    let property: PropertyStatus
    let liabilities: [Liability]
    var onSaved: () -> Void

    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss

    @State private var purchaseText = ""
    @State private var hasDate = false
    @State private var purchaseDate = Date.now
    @State private var rentText = ""
    @State private var chargesText = ""
    @State private var taxText = ""
    @State private var loanID = ""
    @State private var loanPaymentText = ""
    @State private var errorMessage: String?

    var body: some View {
        NavigationStack {
            Form {
                Section("Achat") {
                    TextField("Prix d'achat (\(property.asset.currency))", text: $purchaseText)
                        .keyboardType(.decimalPad)
                    Toggle("Date d'achat", isOn: $hasDate)
                    if hasDate {
                        DatePicker("Le", selection: $purchaseDate, displayedComponents: .date)
                    }
                }
                Section("Location (si locatif)") {
                    TextField("Loyer mensuel (\(property.asset.currency))", text: $rentText)
                        .keyboardType(.decimalPad)
                    TextField("Charges mensuelles (\(property.asset.currency))", text: $chargesText)
                        .keyboardType(.decimalPad)
                    TextField("Taxe foncière annuelle (\(property.asset.currency))", text: $taxText)
                        .keyboardType(.decimalPad)
                }
                Section("Crédit adossé") {
                    Picker("Crédit", selection: $loanID) {
                        Text("Aucun").tag("")
                        ForEach(liabilities.filter { !$0.archived && $0.currency == property.asset.currency }) { l in
                            Text(l.name).tag(l.id)
                        }
                    }
                    TextField("Mensualité (\(property.asset.currency))", text: $loanPaymentText)
                        .keyboardType(.decimalPad)
                }
                if let errorMessage {
                    Text(errorMessage).foregroundStyle(OpaleTheme.loss)
                }
            }
            .navigationTitle(property.asset.name)
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("Enregistrer") { Task { await save() } }
                }
                ToolbarItem(placement: .cancellationAction) {
                    Button("Annuler") { dismiss() }
                }
            }
            .onAppear(perform: prefill)
        }
    }

    private func prefill() {
        let d = property.details
        if d.purchasePrice.raw > 0 { purchaseText = MoneyFormat.input(d.purchasePrice, currency: property.asset.currency) }
        if let date = d.purchaseDate {
            hasDate = true
            purchaseDate = date
        }
        if d.monthlyRent.raw > 0 { rentText = MoneyFormat.input(d.monthlyRent, currency: property.asset.currency) }
        if d.monthlyCharges.raw > 0 { chargesText = MoneyFormat.input(d.monthlyCharges, currency: property.asset.currency) }
        if d.propertyTaxYearly.raw > 0 { taxText = MoneyFormat.input(d.propertyTaxYearly, currency: property.asset.currency) }
        loanID = d.liabilityID ?? ""
        if d.monthlyLoanPayment.raw > 0 { loanPaymentText = MoneyFormat.input(d.monthlyLoanPayment, currency: property.asset.currency) }
    }

    private func cents(_ text: String) -> Int64 {
        text.isEmpty ? 0 : (Cents.parse(text, currency: property.asset.currency)?.raw ?? 0)
    }

    private func save() async {
        guard [purchaseText, rentText, chargesText, taxText, loanPaymentText].allSatisfy({ $0.isEmpty || (Cents.parse($0, currency: property.asset.currency)?.raw ?? -1) >= 0 }) else { errorMessage = "Montants positifs valides requis."; return }
        do {
            try await session.api.upsertProperty(assetID: property.asset.id, .init(
                purchasePriceCents: cents(purchaseText),
                purchaseDate: hasDate ? purchaseDate.opaleDayString : "",
                monthlyRentCents: cents(rentText),
                monthlyChargesCents: cents(chargesText),
                propertyTaxYearlyCents: cents(taxText),
                liabilityID: loanID,
                monthlyLoanPaymentCents: cents(loanPaymentText)
            ))
            session.changed()
            onSaved()
            dismiss()
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
