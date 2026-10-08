import Foundation
import Security

/// Accès minimal au trousseau iOS pour le jeton de session (ENF-003/004 :
/// le jeton n'est jamais stocké en clair dans UserDefaults).
/// `nonisolated` : l'API Security est thread-safe, le client API l'appelle
/// hors de l'acteur principal.
nonisolated enum Keychain {
    private static let service = "app.opale.ios"

    @discardableResult
    static func set(_ value: String, forKey key: String) -> Bool {
        let data = Data(value.utf8)
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key,
        ]
        // Remplacement : suppression puis insertion.
        SecItemDelete(query as CFDictionary)
        var insert = query
        insert[kSecValueData as String] = data
        insert[kSecAttrAccessible as String] = kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly
        return SecItemAdd(insert as CFDictionary, nil) == errSecSuccess
    }

    static func get(_ key: String) -> String? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne,
        ]
        var result: AnyObject?
        guard SecItemCopyMatching(query as CFDictionary, &result) == errSecSuccess,
              let data = result as? Data
        else { return nil }
        return String(data: data, encoding: .utf8)
    }

    /// A credential is never sent to a different configured server.
    static func sessionToken(forServer server: String) -> String? {
        struct Binding: Decodable { let server: String }
        guard let saved = get("session.profile"), let data = saved.data(using: .utf8),
              let binding = try? JSONDecoder().decode(Binding.self, from: data),
              binding.server == server else { return nil }
        return get("session.token")
    }

    static func delete(_ key: String) {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key,
        ]
        SecItemDelete(query as CFDictionary)
    }
}
