import SwiftUI

// Onglets à venir dans les paliers suivants (P5) — placeholders
// volontairement sobres pour garder la navigation complète dès P1 (EF-003).

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
