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
                    .id(session.profileKey)
                    .safeAreaInset(edge: .top) {
                        if session.isOffline {
                            Label("Hors ligne · consultation du dernier état enregistré", systemImage: "wifi.slash")
                                .font(.caption).padding(8).frame(maxWidth: .infinity).background(.regularMaterial)
                        }
                    }
            }
        }
        .task { await session.bootstrap() }
    }
}

/// Les 5 onglets d'Opale : Accueil / Flux / Patrimoine / Projection / Assistant.
struct MainTabView: View {
    @Environment(SessionStore.self) private var session
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
        @Bindable var session = session
        TabView(selection: $selection) {
            Tab(value: .home) {
                HomeView()
            } label: {
                Label("Accueil", systemImage: "circle.hexagongrid.fill")
                    .symbolEffect(.bounce, value: selection == .home)
            }
            Tab(value: .flows) {
                FlowsView()
            } label: {
                Label("Flux", systemImage: "arrow.left.arrow.right")
                    .symbolEffect(.bounce, value: selection == .flows)
            }
            Tab(value: .wealth) {
                WealthView()
            } label: {
                Label("Patrimoine", systemImage: "building.columns.fill")
                    .symbolEffect(.bounce, value: selection == .wealth)
            }
            Tab(value: .projection) {
                ProjectionView()
            } label: {
                Label("Projection", systemImage: "chart.line.uptrend.xyaxis")
                    .symbolEffect(.bounce, value: selection == .projection)
            }
            Tab(value: .assistant) {
                AssistantView()
            } label: {
                Label("Assistant", systemImage: "sparkles")
                    .symbolEffect(.bounce, value: selection == .assistant)
            }
        }
        .sheet(isPresented: $session.showAlerts) { AlertsInboxView() }
        // Chaque changement d'onglet « clique » sous le doigt.
        .sensoryFeedback(.selection, trigger: selection)
        .onChange(of: session.selectedTab, initial: true) { _, tab in
            selection = switch tab { case "flows": .flows; case "wealth": .wealth; case "projection": .projection; case "assistant": .assistant; default: .home }
        }
    }
}

#Preview {
    RootView()
        .environment(SessionStore())
}
