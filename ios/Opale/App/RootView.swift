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
    enum Section: Hashable {
        case home, flows, wealth, projection, assistant
    }

    /// Onglet initial — surchargable par argument de lancement (debug/tests) :
    /// `--tab projection` ouvre directement la Projection, etc.
    @State private var selection: Section = {
        if let idx = CommandLine.arguments.firstIndex(of: "--tab"),
           idx + 1 < CommandLine.arguments.count {
            switch CommandLine.arguments[idx + 1] {
            case "flows": return .flows
            case "wealth": return .wealth
            case "projection": return .projection
            case "assistant": return .assistant
            default: break
            }
        }
        return .home
    }()

    var body: some View {
        TabView(selection: $selection) {
            Tab("Accueil", systemImage: "circle.hexagongrid.fill", value: .home) {
                HomeView()
            }
            Tab("Flux", systemImage: "arrow.left.arrow.right", value: .flows) {
                FlowsView()
            }
            Tab("Patrimoine", systemImage: "building.columns.fill", value: .wealth) {
                WealthView()
            }
            Tab("Projection", systemImage: "chart.line.uptrend.xyaxis", value: .projection) {
                ProjectionView()
            }
            Tab("Assistant", systemImage: "sparkles", value: .assistant) {
                AssistantView()
            }
        }
    }
}

#Preview {
    RootView()
        .environment(SessionStore())
}
