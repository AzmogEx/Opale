import SwiftUI

/// Presented only within an authenticated, unlocked profile.
struct AlertsInboxView: View {
    @Environment(SessionStore.self) private var session
    @Environment(\.dismiss) private var dismiss
    @State private var alerts: [OpaleAlert] = []
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        NavigationStack {
            List {
                if loading { ProgressView() }
                else if alerts.isEmpty, error == nil {
                    ContentUnavailableView("Aucune alerte active", systemImage: "checkmark.shield")
                }
                ForEach(alerts) { alert in
                    VStack(alignment: .leading, spacing: 8) {
                        Label(alert.title, systemImage: alert.severity == "critical" ? "exclamationmark.octagon" : "bell")
                            .font(.headline)
                        Text(alert.detail).font(.subheadline)
                    }.sensitive()
                }
                ToolError(message: error)
                Section { NavigationLink("Gérer mes seuils") { CustomAlertsView() } }
            }
            .navigationTitle("Alertes")
            .toolbar { ToolbarItem(placement: .confirmationAction) { Button("Fermer") { dismiss() } } }
            .task { await load() }
            .refreshable { await load() }
        }
    }
    private func load() async {
        let profile = session.profileKey
        loading = true
        defer { loading = false }
        do {
            let fetched = try await session.api.alerts()
            guard profile == session.profileKey, !Task.isCancelled else { return }
            alerts = fetched
            error = nil
        } catch { if !Task.isCancelled { self.error = error.localizedDescription } }
    }
}
