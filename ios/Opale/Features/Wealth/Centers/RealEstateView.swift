import SwiftUI

/// Centre immobilier (EF-033) : chaque bien avec ses indicateurs calculés
/// par le moteur — rendement brut, cashflow, plus-value, part possédée.
struct RealEstateView: View {
    @Environment(SessionStore.self) private var session

    @State private var properties: [PropertyStatus] = []
    @State private var liabilities: [Liability] = []
    @State private var editing: PropertyStatus?
    @State private var loaded = false

    var body: some View {
        List {
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
                    AmountText(cents: p.asset.latestValue ?? .zero, style: .whole)
                        .font(.title2.weight(.bold))
                }
                Spacer()
                if p.details.purchasePrice.raw > 0 {
                    VStack(alignment: .trailing, spacing: 2) {
                        Text("Plus-value")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                        AmountText(cents: p.capitalGain, style: .signedDelta)
                            .font(.headline)
                            .foregroundStyle(p.capitalGain.raw < 0 ? OpaleTheme.loss : OpaleTheme.gain)
                    }
                }
            }

            if p.details.purchasePrice.raw > 0 {
                Grid(alignment: .leading, horizontalSpacing: 16, verticalSpacing: 8) {
                    GridRow {
                        indicator("Rendement brut", percentLabel(p.grossYieldBps))
                        indicator("Cashflow/mois", MoneyFormat.signedEurosWhole(p.monthlyCashflow),
                                  color: p.monthlyCashflow.raw < 0 ? OpaleTheme.loss : OpaleTheme.gain)
                    }
                    GridRow {
                        indicator("Part possédée", MoneyFormat.eurosWhole(p.equity))
                        if let remaining = p.loanRemaining {
                            indicator("Crédit restant", MoneyFormat.eurosWhole(remaining), color: OpaleTheme.loss)
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
            Text(value).font(.subheadline.weight(.semibold)).foregroundStyle(color)
        }
        .gridColumnAlignment(.leading)
    }

    private func percentLabel(_ bps: Int) -> String {
        String(format: "%d,%02d %%", bps / 100, bps % 100)
    }

    private func load() async {
        properties = (try? await session.api.realEstate()) ?? []
        liabilities = (try? await session.api.listLiabilities()) ?? []
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
                    TextField("Prix d'achat (€)", text: $purchaseText)
                        .keyboardType(.decimalPad)
                    Toggle("Date d'achat", isOn: $hasDate)
                    if hasDate {
                        DatePicker("Le", selection: $purchaseDate, displayedComponents: .date)
                    }
                }
                Section("Location (si locatif)") {
                    TextField("Loyer mensuel (€)", text: $rentText)
                        .keyboardType(.decimalPad)
                    TextField("Charges mensuelles (€)", text: $chargesText)
                        .keyboardType(.decimalPad)
                    TextField("Taxe foncière annuelle (€)", text: $taxText)
                        .keyboardType(.decimalPad)
                }
                Section("Crédit adossé") {
                    Picker("Crédit", selection: $loanID) {
                        Text("Aucun").tag("")
                        ForEach(liabilities) { l in
                            Text(l.name).tag(l.id)
                        }
                    }
                    TextField("Mensualité (€)", text: $loanPaymentText)
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
        if d.purchasePrice.raw > 0 { purchaseText = String(d.purchasePrice.raw / 100) }
        if let date = d.purchaseDate {
            hasDate = true
            purchaseDate = date
        }
        if d.monthlyRent.raw > 0 { rentText = String(d.monthlyRent.raw / 100) }
        if d.monthlyCharges.raw > 0 { chargesText = String(d.monthlyCharges.raw / 100) }
        if d.propertyTaxYearly.raw > 0 { taxText = String(d.propertyTaxYearly.raw / 100) }
        loanID = d.liabilityID ?? ""
        if d.monthlyLoanPayment.raw > 0 { loanPaymentText = String(d.monthlyLoanPayment.raw / 100) }
    }

    private func cents(_ text: String) -> Int64 {
        text.isEmpty ? 0 : (Cents.parse(text)?.raw ?? 0)
    }

    private func save() async {
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
            onSaved()
            dismiss()
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
