import SwiftUI

// Onglets à venir dans les paliers suivants (P2/P3/P5) — placeholders
// volontairement sobres pour garder la navigation complète dès P1 (EF-003).

/// Onglet Flux — transactions, enveloppes, calendrier (P3/P4).
struct FlowsView: View {
    var body: some View {
        NavigationStack {
            ContentUnavailableView(
                "Flux",
                systemImage: "arrow.left.arrow.right",
                description: Text("Transactions, import bancaire et enveloppes arrivent au palier P3.")
            )
            .navigationTitle("Flux")
        }
    }
}

/// Onglet Projection — indépendance financière, simulateur (P2).
struct ProjectionView: View {
    var body: some View {
        NavigationStack {
            ContentUnavailableView(
                "Projection",
                systemImage: "chart.line.uptrend.xyaxis",
                description: Text("Date d'indépendance financière et simulateur arrivent au palier P2.")
            )
            .navigationTitle("Projection")
        }
    }
}

/// Onglet Assistant — IA patrimoniale, mode décision (P5).
struct AssistantView: View {
    @Environment(SessionStore.self) private var session

    var body: some View {
        NavigationStack {
            List {
                ContentUnavailableView(
                    "Assistant",
                    systemImage: "sparkles",
                    description: Text("L'IA patrimoniale (recherche naturelle, mode décision) arrive au palier P5.")
                )
                .listRowSeparator(.hidden)

                Section("Session") {
                    LabeledContent("Serveur", value: session.baseURLString)
                    Button("Se déconnecter", role: .destructive) {
                        Task { await session.logout() }
                    }
                }
            }
            .navigationTitle("Assistant")
        }
    }
}
