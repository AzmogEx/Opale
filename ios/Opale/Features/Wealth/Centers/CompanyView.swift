import SwiftUI

/// Module entrepreneur (EF-036) : la valeur de MA part de la société,
/// compte courant d'associé, dividendes nets — calculés par le moteur.
struct CompanyView: View {
    @Environment(SessionStore.self) private var session

    @State private var companies: [CompanyStatus] = []
    @State private var editing: CompanyStatus?
    @State private var loaded = false

    var body: some View {
        List {
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
                    AmountText(cents: c.asset.latestValue ?? .zero, style: .whole)
                        .font(.title2.weight(.bold))
                }
                Spacer()
                VStack(alignment: .trailing, spacing: 2) {
                    Text("Société entière (dérivée)")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    AmountText(cents: c.companyValue, style: .whole)
                        .font(.headline)
                }
            }

            Grid(alignment: .leading, horizontalSpacing: 16, verticalSpacing: 8) {
                GridRow {
                    stat("Compte courant d'associé", MoneyFormat.eurosWhole(c.details.cca))
                    stat("Ma part + CCA", MoneyFormat.eurosWhole(c.myTotal), color: OpaleTheme.accent)
                }
                GridRow {
                    stat("Dividendes bruts/an", MoneyFormat.eurosWhole(c.details.annualDividends))
                    stat("Nets après PFU 30 %", MoneyFormat.eurosWhole(c.dividendsNet), color: OpaleTheme.gain)
                }
                if c.details.monthlySalary.raw > 0 {
                    GridRow {
                        stat("Rémunération/mois", MoneyFormat.eurosWhole(c.details.monthlySalary))
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
            Text(value).font(.subheadline.weight(.semibold)).foregroundStyle(color)
        }
        .gridColumnAlignment(.leading)
    }

    private func load() async {
        companies = (try? await session.api.companies()) ?? []
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
                    TextField("Compte courant d'associé (€)", text: $ccaText)
                        .keyboardType(.decimalPad)
                    TextField("Dividendes annuels bruts (€)", text: $dividendsText)
                        .keyboardType(.decimalPad)
                    TextField("Rémunération mensuelle (€)", text: $salaryText)
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
        }
    }

    private func prefill() {
        siren = company.details.siren
        ownershipPercent = max(1, company.details.ownershipBps / 100)
        if company.details.cca.raw > 0 { ccaText = String(company.details.cca.raw / 100) }
        if company.details.annualDividends.raw > 0 {
            dividendsText = String(company.details.annualDividends.raw / 100)
        }
        if company.details.monthlySalary.raw > 0 {
            salaryText = String(company.details.monthlySalary.raw / 100)
        }
    }

    private func cents(_ text: String) -> Int64 {
        text.isEmpty ? 0 : (Cents.parse(text)?.raw ?? 0)
    }

    private func save() async {
        do {
            try await session.api.upsertCompany(assetID: company.asset.id, .init(
                siren: siren.trimmingCharacters(in: .whitespaces),
                ownershipBps: ownershipPercent * 100,
                ccaCents: cents(ccaText),
                annualDividendsCents: cents(dividendsText),
                monthlySalaryCents: cents(salaryText)
            ))
            onSaved()
            dismiss()
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
