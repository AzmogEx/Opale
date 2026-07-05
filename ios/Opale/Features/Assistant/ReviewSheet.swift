import SwiftUI

/// Bilan mensuel intelligent (EF-062) : les chiffres du mois clos, rédigés
/// par l'IA quand elle est disponible, en gabarit déterministe sinon.
struct ReviewSheet: View {
    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss

    @State private var review: MonthlyReview?
    @State private var errorMessage: String?
    // Décalage en mois par rapport au mois précédent (0 = mois dernier).
    @State private var monthOffset = 0

    var body: some View {
        NavigationStack {
            Form {
                if let review {
                    Section {
                        monthHeader(review)
                    }

                    Section("Les chiffres") {
                        LabeledContent("Revenus") {
                            AmountText(cents: review.summary.income, style: .whole)
                        }
                        LabeledContent("Dépenses") {
                            AmountText(cents: review.summary.expenses, style: .whole)
                        }
                        LabeledContent("Mis de côté") {
                            AmountText(cents: review.summary.net, style: .signedDelta)
                                .foregroundStyle(review.summary.net.raw < 0 ? OpaleTheme.loss : OpaleTheme.gain)
                        }
                        LabeledContent("Taux d'épargne", value: "\(review.savingsRateBps / 100) %")
                        LabeledContent("Santé financière", value: "\(review.healthScore)/100")
                    }

                    if let top = review.topCategories, !top.isEmpty {
                        Section("Plus gros postes") {
                            ForEach(top) { c in
                                LabeledContent {
                                    AmountText(cents: c.total, style: .whole)
                                } label: {
                                    Label(c.name, systemImage: c.icon)
                                }
                            }
                        }
                    }

                    Section {
                        Text(review.narrative)
                            .font(.subheadline)
                    } header: {
                        Text("Le bilan")
                    } footer: {
                        Label(tierFooter(review.narrativeTier), systemImage: "lock.shield")
                            .font(.caption2)
                    }
                } else if let errorMessage {
                    ContentUnavailableView(
                        "Bilan indisponible",
                        systemImage: "doc.text.magnifyingglass",
                        description: Text(errorMessage)
                    )
                } else {
                    HStack {
                        ProgressView()
                        Text("Calcul du bilan…")
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                    }
                }
            }
            .navigationTitle("Bilan mensuel")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Fermer") { dismiss() }
                }
            }
            .task(id: monthOffset) { await load() }
        }
    }

    private func monthHeader(_ review: MonthlyReview) -> some View {
        HStack {
            Button {
                monthOffset -= 1
            } label: {
                Image(systemName: "chevron.left")
            }
            Spacer()
            Text(monthTitle(review))
                .font(.headline)
            Spacer()
            Button {
                monthOffset += 1
            } label: {
                Image(systemName: "chevron.right")
            }
            .disabled(monthOffset >= 0)
        }
        .buttonStyle(.borderless)
    }

    private func monthTitle(_ review: MonthlyReview) -> String {
        var comps = DateComponents()
        comps.year = review.year
        comps.month = review.month
        let date = Calendar.current.date(from: comps) ?? .now
        return date.formatted(.dateTime.month(.wide).year()).capitalized
    }

    private func tierFooter(_ tier: String) -> String {
        switch tier {
        case "n2": "Bilan rédigé sur ton homelab — les données ne l'ont jamais quitté."
        case "n3": "Bilan rédigé par le modèle cloud à partir de données anonymisées."
        default: "Bilan gabarit du moteur — l'IA est hors ligne."
        }
    }

    private func load() async {
        // Mois cible : (mois courant − 1) + décalage choisi.
        let target = Calendar.current.date(byAdding: .month, value: -1 + monthOffset, to: .now) ?? .now
        let comps = Calendar.current.dateComponents([.year, .month], from: target)
        do {
            review = try await session.api.monthlyReview(year: comps.year, month: comps.month)
            errorMessage = nil
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
