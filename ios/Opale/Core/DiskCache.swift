import Foundation

/// Cache disque (mode hors-ligne) : le dernier état connu de chaque écran,
/// par profil. L'app s'ouvre instantanément sur ces données puis rafraîchit ;
/// API injoignable → on reste sur le cache avec un bandeau « hors ligne ».
/// Vidé à la déconnexion (rien ne survit à la session).
nonisolated enum DiskCache {
	/// Une valeur datée — pour afficher « données du … ».
	struct Stamped<T: Codable>: Codable {
		let value: T
		let at: Date
	}

	private static var directory: URL {
		let base = FileManager.default.urls(for: .applicationSupportDirectory,
		                                    in: .userDomainMask)[0]
		let dir = base.appendingPathComponent("OpaleCache", isDirectory: true)
		try? FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
		return dir
	}

	private static func url(_ key: String) -> URL {
		// Clé assainie pour le nom de fichier.
		let safe = key.replacingOccurrences(of: "/", with: "_")
		return directory.appendingPathComponent(safe + ".json")
	}

	static func save<T: Codable>(_ value: T, key: String) {
		let stamped = Stamped(value: value, at: .now)
		guard let data = try? JSONEncoder().encode(stamped) else { return }
		try? data.write(to: url(key), options: .atomic)
	}

	static func load<T: Codable>(_ type: T.Type, key: String) -> Stamped<T>? {
		guard let data = try? Data(contentsOf: url(key)) else { return nil }
		return try? JSONDecoder().decode(Stamped<T>.self, from: data)
	}

	/// Efface tout le cache (déconnexion).
	static func clear() {
		try? FileManager.default.removeItem(at: directory)
	}
}
