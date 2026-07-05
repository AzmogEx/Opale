import SwiftUI

/// Racine de l'app : porte d'authentification puis les 5 onglets (EF-003).
struct RootView: View {
    @Environment(SessionStore.self) private var session

    var body: some View {
        Group {
            switch session.state {
            case .loading:
                ProgressView()
            case .loggedOut:
                ProfileGateView()
            case .loggedIn:
                MainTabView()
            }
        }
        .task { await session.bootstrap() }
    }
}

/// Les 5 onglets d'Opale : Accueil / Flux / Patrimoine / Projection / Assistant.
struct MainTabView: View {
    var body: some View {
        TabView {
            Tab("Accueil", systemImage: "circle.hexagongrid.fill") {
                HomeView()
            }
            Tab("Flux", systemImage: "arrow.left.arrow.right") {
                FlowsView()
            }
            Tab("Patrimoine", systemImage: "building.columns.fill") {
                WealthView()
            }
            Tab("Projection", systemImage: "chart.line.uptrend.xyaxis") {
                ProjectionView()
            }
            Tab("Assistant", systemImage: "sparkles") {
                AssistantView()
            }
        }
    }
}

#Preview {
    RootView()
        .environment(SessionStore())
}
