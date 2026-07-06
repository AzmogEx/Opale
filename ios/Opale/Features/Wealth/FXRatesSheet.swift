import SwiftUI

/// Devises & taux (EF-008) : taux de change manuels — aucune API externe,
/// la confidentialité d'abord. 1 unité de devise = X euros ; les actifs en
/// devise sans taux sont comptés 1:1 dans le patrimoine (et signalés ici).
struct FXRatesSheet: View {
    var onChanged: () -> Void

    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss

    @State private var rates: [FXRate] = []
    @State private var unrated: [String] = []
    @State private var newCurrency = "USD"
    @State private var newRateText = ""
    @State private var errorMessage: String?

    var body: some View {
        NavigationStack {
            Form {
                if !unrated.isEmpty {
                    Section {
                        Label {
                            Text("Sans taux, ces devises sont comptées 1 pour 1 dans ton patrimoine : \(unrated.joined(separator: ", "))")
                                .font(.subheadline)
                        } icon: {
                            Image(systemName: "exclamationmark.triangle.fill")
                                .foregroundStyle(.orange)
                        }
                    }
                }

                Section("Taux enregistrés") {
                    if rates.isEmpty {
                        Text("Aucun taux — tout est en euros.")
                            .foregroundStyle(.secondary)
                    }
                    ForEach(rates) { rate in
                        LabeledContent("1 \(rate.currency)") {
                            Text(Self.euroLabel(rate.rateMicro))
                                .fontWeight(.semibold)
                        }
                    }
                    .onDelete { indexSet in
                        Task {
                            for i in indexSet {
                                try? await session.api.deleteFXRate(currency: rates[i].currency)
                            }
                            await load()
                            onChanged()
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
                    Button("Enregistrer le taux") {
                        Task { await save() }
                    }
                    .disabled(Cents.parse(newRateText) == nil)
                } header: {
                    Text("Nouveau taux")
                } footer: {
                    Text("Saisie manuelle et privée : Opale n'appelle aucun service de change. Pense à rafraîchir de temps en temps.")
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
        let euros = rateMicro / 1_000_000
        let frac = (rateMicro % 1_000_000) / 10_000 // 2 décimales
        return String(format: "%d,%02d €", euros, frac)
    }

    private func load() async {
        if let result = try? await session.api.fxRates() {
            rates = result.rates
            unrated = result.unrated
        }
    }

    private func save() async {
        // Le taux saisi en euros (ex. « 0,92 ») devient des micro-euros via
        // Cents.parse (entier, pas de float) : centimes × 10 000.
        guard let cents = Cents.parse(newRateText), cents.raw > 0 else { return }
        do {
            try await session.api.upsertFXRate(currency: newCurrency, rateMicro: cents.raw * 10_000)
            newRateText = ""
            await load()
            onChanged()
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
