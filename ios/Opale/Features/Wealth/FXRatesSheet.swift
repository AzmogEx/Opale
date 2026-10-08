import SwiftUI

/// Taux manuels datés, prioritaires sur les taux de référence.
struct FXRatesSheet: View {
    var onChanged: () -> Void

    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss

    @State private var rates: [FXRate] = []
    @State private var unrated: [String] = []
    @State private var newCurrency = "USD"
    @State private var newRateText = ""
    @State private var asOf = Date.now
    @State private var busy = false
    private var parsedRate: Int64? {
        let text = newRateText.replacingOccurrences(of: ",", with: ".")
        guard text.range(of: #"^[0-9]+(\.[0-9]{1,6})?$"#, options: .regularExpression) != nil,
              let value = Decimal(string: text), value > 0, value <= Decimal(Int64.max) / 1_000_000 else { return nil }
        return NSDecimalNumber(decimal: value * 1_000_000).int64Value
    }
    @State private var errorMessage: String?

    var body: some View {
        NavigationStack {
            Form {
                if !unrated.isEmpty {
                    Section {
                        Label {
                            Text("Un taux applicable est nécessaire pour calculer les totaux de ces devises : \(unrated.joined(separator: ", "))")
                                .font(.subheadline)
                        } icon: {
                            Image(systemName: "exclamationmark.triangle.fill")
                                .foregroundStyle(.orange)
                        }
                    }
                }

                Section("Taux enregistrés") {
                    if rates.isEmpty {
                        Text("Aucun taux enregistré.")
                            .foregroundStyle(.secondary)
                    }
                    ForEach(rates) { rate in
                        LabeledContent("1 \(rate.currency)") {
                            Text(Self.euroLabel(rate.rateMicro)).fontWeight(.semibold)
                        }
                        Text("Au \(rate.asOf ?? "—") · source : \(rate.source ?? "manuelle")").font(.caption)
                        Button("Modifier ce taux") { newCurrency = rate.currency; newRateText = NSDecimalNumber(decimal: Decimal(rate.rateMicro) / 1_000_000).stringValue; asOf = rate.asOf.flatMap(Date.fromOpaleDay) ?? .now
                        }
                    }
                    .onDelete { indexSet in
                        Task {
                            do { for i in indexSet { try await session.api.deleteFXRate(currency: rates[i].currency) }; await load(); session.changed(); onChanged() }
                            catch { errorMessage = error.localizedDescription }
                        }
                    }
                }

                Section {
                    Picker("Devise", selection: $newCurrency) {
                        ForEach(candidateCurrencies, id: \.self) { c in
                            Text(c).tag(c)
                        }
                    }
                    TextField("Valeur d'1 \(newCurrency) en euros (ex. 0,92)", text: $newRateText)
                        .keyboardType(.decimalPad)
                    DatePicker("Date du taux", selection: $asOf, in: ...Date.now, displayedComponents: .date)
                    Button("Enregistrer le taux") {
                        Task { await save() }
                    }
                    .disabled(parsedRate == nil || busy)
                } header: {
                    Text("Nouveau taux")
                } footer: {
                    Text("Unité : 1 devise = X euros (six décimales maximum). Le taux manuel s’applique à partir de sa date ; les cours automatiques peuvent compléter les taux de référence si activés sur le serveur.")
                }

                if let errorMessage {
                    Text(errorMessage).foregroundStyle(OpaleTheme.loss)
                }
            }
            .navigationTitle("Devises & taux")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Fermer") { dismiss() }
                }
            }
            .task { await load() }
        }
    }

    /// Devises proposables : celles du formulaire d'actif + celles déjà en usage.
    private var candidateCurrencies: [String] {
        var set = Set(AssetFormSheet.currencies + unrated)
        set.remove("EUR")
        return set.sorted()
    }

    /// « 0,92 € » depuis des micro-euros.
    private static func euroLabel(_ rateMicro: Int64) -> String {
        NSDecimalNumber(decimal: Decimal(rateMicro) / 1_000_000).stringValue + " €"
    }

    private func load() async {
        do { let result = try await session.api.fxRates(); rates = result.rates; unrated = result.unrated; errorMessage = nil }
        catch { errorMessage = error.localizedDescription }
    }

    private func save() async {
        guard let micro = parsedRate else { return }
        busy = true; defer { busy = false }
        do {
            try await session.api.upsertFXRate(currency: newCurrency, rateMicro: micro, asOf: asOf.opaleDayString)
            newRateText = ""
            await load()
            session.changed()
            onChanged()
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
