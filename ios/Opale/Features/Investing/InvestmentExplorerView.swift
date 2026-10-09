import SwiftUI

struct InvestmentExplorerView: View {
    @Environment(SessionStore.self) private var session
    @State private var risk: InvestmentRisk = .medium
    @State private var country = ""
    @State private var family = ""
    @State private var envelope = ""
    @State private var residence = "France"
    @State private var horizon = 8
    @State private var respectHorizon = true
    @State private var onlySaved = false
    @State private var query = ""
    @State private var savedIDs = Set<String>()
    private var cacheKey: String { "investment-watchlist-v1|" + session.profileKey }
    private var ideas: [InvestmentIdea] {
        InvestmentCatalog.ideas.filter { idea in
            (!onlySaved || savedIDs.contains(idea.id)) && idea.matches(risk: risk, country: country, family: family, envelope: envelope, residence: residence, horizon: respectHorizon ? horizon : nil, query: query)
        }
    }
    var body: some View {
        List {
            Section {
                Text("Des pistes à comparer").font(.title2.bold())
                Text("Explore selon ton projet. Ces exemples expliquent les supports, leurs risques et leurs sources ; ils ne tiennent pas compte de toute ta situation financière.")
                    .font(.subheadline).foregroundStyle(.secondary)
                Picker("Niveau de risque", selection: $risk) { ForEach(InvestmentRisk.allCases) { Text($0.label).tag($0) } }.pickerStyle(.segmented).accessibilityIdentifier("invest.risk")
                Text(risk.guidance).font(.caption).foregroundStyle(.secondary)
            }
            Section("Mon projet") {
                Picker("Résidence fiscale", selection: $residence) { ForEach(InvestmentCatalog.residences, id: \.self) { Text($0).tag($0) } }
                Picker("Pays / zone d’investissement", selection: $country) { Text("Toutes les zones").tag(""); ForEach(InvestmentCatalog.geographies, id: \.self) { Text($0).tag($0) } }.accessibilityIdentifier("invest.country")
                Picker("Type de placement", selection: $family) { Text("Tous les types").tag(""); ForEach(InvestmentFamily.allCases) { Text($0.label).tag($0.rawValue) } }.accessibilityIdentifier("invest.family")
                Picker("Enveloppe / accès", selection: $envelope) {
                    Text("Tous").tag("")
                    ForEach(residence == "France" ? ["PEA", "CTO", "Assurance-vie", "Livret", "Banque", "Direct"] : ["CTO", "Banque", "Direct"], id: \.self) { Text($0).tag($0) }
                }
                Stepper("Horizon : \(horizon) an\(horizon > 1 ? "s" : "")", value: $horizon, in: 0...30)
                Toggle("Respecter cet horizon", isOn: $respectHorizon)
                Toggle("Ma liste à étudier", isOn: $onlySaved)
                Text(residence == "France" ? "Le PEA et l’assurance-vie sont des enveloppes ; leur risque dépend des actifs choisis. Le pays du fonds ne décrit pas à lui seul son exposition." : "Le catalogue ne modélise pas la fiscalité de cette résidence. CTO désigne ici un compte d’investissement ordinaire ; vérifie les règles locales et l’accès aux produits.")
                    .font(.caption).foregroundStyle(.secondary)
            }
            Section("\(ideas.count) piste\(ideas.count > 1 ? "s" : "") à étudier") {
                ForEach(ideas) { idea in
                    NavigationLink { InvestmentIdeaDetail(idea: idea, isSaved: savedBinding(idea.id), horizon: horizon) } label: {
                        HStack(alignment: .top, spacing: 12) {
                            Image(systemName: idea.families.first?.icon ?? "chart.pie").foregroundStyle(OpaleTheme.accent).frame(width: 24)
                            VStack(alignment: .leading, spacing: 5) {
                                Text(idea.name).font(.body.weight(.medium))
                                Text(idea.identifier).font(.caption).foregroundStyle(.secondary)
                                Text(idea.geographies.joined(separator: " · ") + " · " + idea.envelopes.joined(separator: " / ")).font(.caption2).foregroundStyle(.secondary)
                            }
                            if savedIDs.contains(idea.id) { Spacer(minLength: 0); Image(systemName: "bookmark.fill").foregroundStyle(OpaleTheme.accent).accessibilityLabel("Dans ma liste") }
                        }.padding(.vertical, 3)
                    }.accessibilityIdentifier("invest.idea.\(idea.id)")
                }
                if ideas.isEmpty {
                    ContentUnavailableView("Aucune piste avec ces filtres", systemImage: "line.3.horizontal.decrease", description: Text("Essaie une autre zone ou un autre support. Un horizon court limite les placements proposés."))
                }
            }
            Section {
                Text("Repères de risque Opale, indicatifs, distincts du score réglementaire du DIC. Les horizons sont des repères de préparation, sans garantie de résultat. Consulte les documents actuels avant une décision.")
                Text("Sources consultées le \(FinancialTools.dayLabel(InvestmentCatalog.checkedOn)). Aucun cours ni classement de performance en direct.")
                if let checked = Date.fromOpaleDay(InvestmentCatalog.checkedOn), Date.now.timeIntervalSince(checked) > 90 * 86_400 {
                    Label("Catalogue à revérifier : les sources ont plus de 90 jours.", systemImage: "clock.badge.exclamationmark").foregroundStyle(.orange)
                }
            }.font(.caption).foregroundStyle(.secondary)
        }.opaleList().navigationTitle("Explorer les investissements").navigationBarTitleDisplayMode(.inline)
        .searchable(text: $query, prompt: "Nom, ISIN ou zone")
        .task(id: session.profileKey) { savedIDs = Set(DiskCache.load([String].self, key: cacheKey)?.value ?? []) }
        .onChange(of: residence) { _, _ in envelope = "" }
    }
    private func savedBinding(_ id: String) -> Binding<Bool> {
        Binding(get: { savedIDs.contains(id) }, set: { wanted in
            if wanted { savedIDs.insert(id) } else { savedIDs.remove(id) }
            DiskCache.save(savedIDs.sorted(), key: cacheKey)
        })
    }
}
