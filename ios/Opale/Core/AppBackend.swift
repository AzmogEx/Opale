import Foundation

/// Le service Opale est toujours hébergé par Vaycode, sans préférence utilisateur.
nonisolated enum AppBackend {
    static let productionURL = URL(string: "https://opale.vaycode.com")!

    static var url: URL {
        #if DEBUG && targetEnvironment(simulator)
        // Réservé aux recettes automatisées sur simulateur. Jamais compilé sur iPhone.
        if let local = simulatorTestURL(arguments: CommandLine.arguments) { return local }
        #endif
        return productionURL
    }

    #if DEBUG && targetEnvironment(simulator)
    static func simulatorTestURL(arguments: [String]) -> URL? {
        guard let index = arguments.firstIndex(of: "--base-url"), arguments.count > index + 1,
              let url = URL(string: arguments[index + 1]),
              url.scheme == "http", ["localhost", "127.0.0.1", "::1"].contains(url.host ?? ""),
              url.port != nil, url.user == nil, url.password == nil,
              url.path.isEmpty, url.query == nil, url.fragment == nil else { return nil }
        return url
    }
    #endif
}
