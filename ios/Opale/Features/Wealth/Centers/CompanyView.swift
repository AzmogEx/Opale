import SwiftUI

/// Module entrepreneur (EF-036) : la valeur de MA part de la société,
/// compte courant d'associé, dividendes nets — calculés par le moteur.
struct CompanyView: View {
    @Environment(SessionStore.self) private var session

    @State private var companies: [CompanyStatus] = []
    @State private var editing: CompanyStatus?
    @State private var loaded = false
    @State private var errorMessage: String?

    var body: some View {
        List {
            if let errorMessage { Text(errorMessage).foregroundStyle(.red) }
            if loaded && companies.isEmpty {
                ContentUnavailableView(
                    "Aucune société",
                    systemImage: "briefcase",
                    description: Text("Ajoute un actif de type « Parts de société » dans Patrimoine ; il apparaîtra ici.")
                )
            }
            ForEach(companies) { company in
                Section(company.asset.name) {
                    companyCard(company)
                    Button {
                        editing = company
                    } label: {
                        Label("Modifier les détails (parts, CCA, dividendes…)",
                              systemImage: "square.and.pencil")
                            .font(.subheadline)
                    }
                }
            }
        }
        .opaleList()
        .navigationTitle("Entreprise")
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
        .refreshable { await load() }
        .sheet(item: $editing) { company in
            CompanyFormSheet(company: company) {
                Task { await load() }
            }
        }
    }

    @ViewBuilder
    private func companyCard(_ c: CompanyStatus) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                VStack(alignment: .leading, spacing: 2) {
                    Text("Ma part (\(c.details.ownershipBps / 100) %)")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    Group { if let value = c.asset.latestValue { AmountText(cents: value, style: .whole, currency: c.asset.currency) } else { Text("Valeur non renseignée") } }
                        .font(.title2.weight(.bold))
                }
                Spacer()
                VStack(alignment: .trailing, spacing: 2) {
                    Text("Société entière (dérivée)")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    AmountText(cents: c.companyValue, style: .whole, currency: c.asset.currency)
                        .font(.headline)
                }
            }

            Grid(alignment: .leading, horizontalSpacing: 16, verticalSpacing: 8) {
                GridRow {
                    stat("Compte courant d'associé", MoneyFormat.amount(c.details.cca, currency: c.asset.currency))
                    stat("Ma part + CCA", MoneyFormat.amount(c.myTotal, currency: c.asset.currency), color: OpaleTheme.accent)
                }
                GridRow {
                    stat("Dividendes bruts/an", MoneyFormat.amount(c.details.annualDividends, currency: c.asset.currency))
                    stat("Nets après PFU 2026 · 31,4 %", MoneyFormat.amount(c.dividendsNet, currency: c.asset.currency), color: OpaleTheme.gain)
                }
                if c.details.monthlySalary.raw > 0 {
                    GridRow {
                        stat("Rémunération/mois", MoneyFormat.amount(c.details.monthlySalary, currency: c.asset.currency))
                        if !c.details.siren.isEmpty {
                            stat("SIREN", c.details.siren)
                        }
                    }
                }
            }
        }
        .padding(.vertical, 4)
    }

    private func stat(_ title: String, _ value: String, color: Color = .primary) -> some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(title).font(.caption2).foregroundStyle(.secondary)
            Text(value).sensitive().font(.subheadline.weight(.semibold)).foregroundStyle(color)
        }
        .gridColumnAlignment(.leading)
    }

    private func load() async {
        do { companies = try await session.api.companies(); errorMessage = nil } catch { errorMessage = error.localizedDescription }
        loaded = true
    }
}

/// Saisie des détails de la société.
private struct CompanyFormSheet: View {
    let company: CompanyStatus
    var onSaved: () -> Void

    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss

    @State private var siren = ""
    @State private var ownershipPercent = 100
    @State private var ccaText = ""
    @State private var ccaAssetID = ""
    @State private var assets: [Asset] = []
    @State private var dividendsText = ""
    @State private var salaryText = ""
    @State private var errorMessage: String?

    var body: some View {
        NavigationStack {
            Form {
                Section("La société") {
                    TextField("SIREN (optionnel)", text: $siren)
                        .keyboardType(.numberPad)
                    Stepper("Parts détenues : \(ownershipPercent) %",
                            value: $ownershipPercent, in: 1...100)
                }
                Section {
                    Picker("Créance CCA", selection: $ccaAssetID) {
                        Text("Créer automatiquement").tag("")
                        ForEach(assets.filter { $0.id != company.asset.id && $0.currency == company.asset.currency && !$0.archived }) { Text($0.name).tag($0.id) }
                    }
                    Text("La créance CCA est un actif distinct inclus dans le patrimoine. Sélectionne une créance existante pour éviter un doublon.").font(.caption)
                    TextField("Compte courant d'associé (\(company.asset.currency))", text: $ccaText)
                        .keyboardType(.decimalPad)
                    TextField("Dividendes annuels bruts (\(company.asset.currency))", text: $dividendsText)
                        .keyboardType(.decimalPad)
                    TextField("Rémunération mensuelle (\(company.asset.currency))", text: $salaryText)
                        .keyboardType(.decimalPad)
                } header: {
                    Text("Ma position")
                } footer: {
                    Text("Valorise l'actif à la valeur de TA part (c'est elle qui compte dans ton patrimoine) ; le moteur en déduit la société entière.")
                }
                if let errorMessage {
                    Text(errorMessage).foregroundStyle(OpaleTheme.loss)
                }
            }
            .navigationTitle(company.asset.name)
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
            .task { do { assets = try await session.api.listAssets() } catch { errorMessage = error.localizedDescription } }
        }
    }

    private func prefill() {
        siren = company.details.siren
        ccaAssetID = company.details.ccaAssetID ?? ""
        ownershipPercent = max(1, company.details.ownershipBps / 100)
        if company.details.cca.raw > 0 { ccaText = MoneyFormat.input(company.details.cca, currency: company.asset.currency) }
        if company.details.annualDividends.raw > 0 {
            dividendsText = MoneyFormat.input(company.details.annualDividends, currency: company.asset.currency)
        }
        if company.details.monthlySalary.raw > 0 {
            salaryText = MoneyFormat.input(company.details.monthlySalary, currency: company.asset.currency)
        }
    }

    private func cents(_ text: String) -> Int64 {
        text.isEmpty ? 0 : (Cents.parse(text, currency: company.asset.currency)?.raw ?? 0)
    }

    private func save() async {
        guard [ccaText, dividendsText, salaryText].allSatisfy({ $0.isEmpty || (Cents.parse($0, currency: company.asset.currency)?.raw ?? -1) >= 0 }) else { errorMessage = "Montants positifs valides requis."; return }
        do {
            try await session.api.upsertCompany(assetID: company.asset.id, .init(
                siren: siren.trimmingCharacters(in: .whitespaces),
                ownershipBps: ownershipPercent * 100,
                ccaCents: cents(ccaText),
                ccaAssetID: ccaAssetID.isEmpty ? nil : ccaAssetID,
                annualDividendsCents: cents(dividendsText),
                monthlySalaryCents: cents(salaryText)
            ))
            session.changed()
            onSaved()
            dismiss()
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
