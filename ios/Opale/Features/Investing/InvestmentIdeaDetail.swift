import SwiftUI

struct InvestmentIdeaDetail: View {
    let idea: InvestmentIdea
    @Binding var isSaved: Bool
    let horizon: Int
    var body: some View {
        List {
            Section {
                Text(idea.name).font(.title2.bold())
                Text(idea.identifier).font(.subheadline).foregroundStyle(.secondary).textSelection(.enabled)
                LabeledContent("Risque indicatif Opale", value: idea.risk.label)
                LabeledContent("Zone / pays", value: idea.geographies.joined(separator: ", "))
                LabeledContent("Accès à vérifier", value: idea.envelopes.joined(separator: " / "))
                LabeledContent("Horizon de préparation", value: "\(idea.horizonYears) ans ou plus")
                if horizon < idea.horizonYears { Label("Ton horizon est plus court que ce repère.", systemImage: "exclamationmark.triangle").foregroundStyle(.orange) }
                Toggle("Garder dans ma liste à étudier", isOn: $isSaved).accessibilityIdentifier("invest.save")
            }
            Section("Pourquoi l’étudier") { Text(idea.rationale) }
            Section("Risques à comprendre") { Text(idea.risks) }
            Section("Frais et fiscalité") { Text(idea.costs) }
            Section("Disponibilité de l’argent") { Text(idea.liquidity) }
            Section("Avant de décider") { Text(idea.check) }
            Section("Sources officielles") {
                ForEach(idea.sources, id: \.url) { source in
                    if let url = URL(string: source.url), url.scheme == "https" {
                        Link(destination: url) { Label(source.title, systemImage: "arrow.up.right.square") }
                    }
                }
                Text("Consultées le \(FinancialTools.dayLabel(InvestmentCatalog.checkedOn)). Vérifie les conditions, frais et documents actuels. Aucun achat n’est exécuté depuis cette page.").font(.caption).foregroundStyle(.secondary)
            }
        }.opaleList().navigationTitle("Piste d’investissement").navigationBarTitleDisplayMode(.inline)
    }
}
