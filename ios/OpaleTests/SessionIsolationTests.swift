import XCTest
@testable import Opale

private final class SessionProtocol: URLProtocol, @unchecked Sendable {
    nonisolated(unsafe) static var handler: ((URLRequest) throws -> (Int, Data))?
    override class func canInit(with request: URLRequest) -> Bool { true }
    override class func canonicalRequest(for request: URLRequest) -> URLRequest { request }
    override func startLoading() {
        do {
            guard let handler = Self.handler else { throw URLError(.notConnectedToInternet) }
            let (status, data) = try handler(request)
            let response = HTTPURLResponse(url: request.url!, statusCode: status, httpVersion: nil, headerFields: ["Content-Type": "application/json"])!
            client?.urlProtocol(self, didReceive: response, cacheStoragePolicy: .notAllowed)
            client?.urlProtocol(self, didLoad: data)
            client?.urlProtocolDidFinishLoading(self)
        } catch { client?.urlProtocol(self, didFailWithError: error) }
    }
    override func stopLoading() {}
}

final class SessionIsolationTests: XCTestCase {
    @MainActor private func isolated(_ perform: (SessionStore) async throws -> Void) async throws {
        let previousToken = Keychain.get("session.token")
        let previousProfile = Keychain.get("session.profile")
        let previousBase = UserDefaults.standard.string(forKey: "opale.baseURL")
        defer {
            if let previousToken { Keychain.set(previousToken, forKey: "session.token") } else { Keychain.delete("session.token") }
            if let previousProfile { Keychain.set(previousProfile, forKey: "session.profile") } else { Keychain.delete("session.profile") }
            if let previousBase { UserDefaults.standard.set(previousBase, forKey: "opale.baseURL") } else { UserDefaults.standard.removeObject(forKey: "opale.baseURL") }
            SessionProtocol.handler = nil
        }
        let config = URLSessionConfiguration.ephemeral
        config.protocolClasses = [SessionProtocol.self]
        let session = SessionStore(networkSession: URLSession(configuration: config), clearCachedData: {})
        XCTAssertTrue(Keychain.set("synthetic-token", forKey: "session.token"))
        let profile = Profile(id: "synthetic-a", name: "Synthetic A", privacyDefault: "N1")
        struct Saved: Encodable { let server: String; let profile: Profile }
        let data = try JSONEncoder().encode(Saved(server: session.baseURLString, profile: profile))
        XCTAssertTrue(Keychain.set(String(decoding: data, as: UTF8.self), forKey: "session.profile"))
        try await perform(session)
    }
    @MainActor func testOfflineStartupKeepsSessionAndCorrectProfile() async throws {
        try await isolated { session in
            SessionProtocol.handler = { _ in throw URLError(.notConnectedToInternet) }
            await session.bootstrap()
            XCTAssertEqual(session.profileID, "synthetic-a")
            XCTAssertTrue(session.isOffline)
            XCTAssertEqual(Keychain.get("session.token"), "synthetic-token")
        }
    }
    @MainActor func testServerFailureDoesNotRevokeSession() async throws {
        try await isolated { session in
            SessionProtocol.handler = { _ in (503, Data()) }
            await session.bootstrap()
            XCTAssertEqual(session.profileID, "synthetic-a")
            XCTAssertTrue(session.isOffline)
            XCTAssertEqual(Keychain.get("session.token"), "synthetic-token")
        }
    }
    @MainActor func testCacheFromAnotherServerIsNotRestored() async throws {
        try await isolated { session in
            struct Saved: Encodable { let server: String; let profile: Profile }
            let data = try JSONEncoder().encode(Saved(server: "https://other.invalid", profile: Profile(id: "synthetic-a", name: "Synthetic A", privacyDefault: "N1")))
            Keychain.set(String(decoding: data, as: UTF8.self), forKey: "session.profile")
            SessionProtocol.handler = { _ in throw URLError(.timedOut) }
            await session.bootstrap()
            XCTAssertEqual(session.profileID, "anonyme")
            XCTAssertEqual(Keychain.get("session.token"), "synthetic-token")
            XCTAssertNil(Keychain.sessionToken(forServer: session.baseURLString))
            SessionProtocol.handler = { request in
                guard request.value(forHTTPHeaderField: "Authorization") == nil else { throw URLError(.userAuthenticationRequired) }
                return (204, Data())
            }
            let _: APIClient.EmptyResponse = try await session.api.request("GET", "/v1/public", authenticated: false)
            do {
                let _: APIClient.EmptyResponse = try await session.api.request("GET", "/v1/private")
                XCTFail("A credential from another server must not be sent")
            } catch APIError.notAuthenticated {
                // Re-authentication is required before contacting the other server.
            }
        }
    }
    @MainActor func testConfirmedUnauthorizedRevokesLocalSession() async throws {
        try await isolated { session in
            SessionProtocol.handler = { _ in (401, Data()) }
            await session.bootstrap()
            XCTAssertEqual(session.profileID, "anonyme")
            XCTAssertFalse(session.isOffline)
            XCTAssertNil(Keychain.get("session.token"))
            XCTAssertNil(Keychain.get("session.profile"))
        }
    }

    @MainActor func testLegacyServerPreferenceCannotRedirectRequests() async throws {
        try await isolated { _ in
            UserDefaults.standard.set("https://other.invalid", forKey: "opale.baseURL")
            let config = URLSessionConfiguration.ephemeral
            config.protocolClasses = [SessionProtocol.self]
            let fixed = SessionStore(networkSession: URLSession(configuration: config), clearCachedData: {})
            XCTAssertEqual(fixed.baseURLString, "https://opale.vaycode.com")
            XCTAssertNil(UserDefaults.standard.object(forKey: "opale.baseURL"))
            SessionProtocol.handler = { request in
                XCTAssertEqual(request.url?.host, "opale.vaycode.com")
                XCTAssertEqual(request.url?.scheme, "https")
                return (204, Data())
            }
            let _: APIClient.EmptyResponse = try await fixed.api.request("GET", "/v1/public", authenticated: false)
            let _: APIClient.EmptyResponse = try await fixed.api.request("GET", "/v1/private")
        }
    }

    #if DEBUG && targetEnvironment(simulator)
    func testSimulatorOverrideOnlyAcceptsLoopbackTestServer() {
        XCTAssertEqual(AppBackend.simulatorTestURL(arguments: ["Opale", "--base-url", "http://localhost:58088"])?.host, "localhost")
        for address in ["https://other.invalid", "http://other.invalid:58088", "http://localhost.evil.invalid:58088", "http://user:secret@localhost:58088", "http://localhost:58088/path", "http://localhost:58088?target=other", "http://localhost:58088#other"] {
            XCTAssertNil(AppBackend.simulatorTestURL(arguments: ["Opale", "--base-url", address]))
        }
    }
    #endif
    @MainActor func testClientCapturesOriginalTokenAcrossProfileChange() async throws {
        try await isolated { session in
            let clientA = session.api
            Keychain.set("token-b", forKey: "session.token")
            SessionProtocol.handler = { request in
                guard request.value(forHTTPHeaderField: "Authorization") == "Bearer synthetic-token" else { throw URLError(.userAuthenticationRequired) }
                return (204, Data())
            }
            let _: APIClient.EmptyResponse = try await clientA.request("GET", "/v1/noop")
        }
    }
    @MainActor func testPushCannotOpenAnotherProfileOrUnknownDestination() {
        XCTAssertNil(PushRoute(profileID: "a", destination: "home").destination(for: "b"))
        XCTAssertNil(PushRoute(profileID: "a", destination: "external-link").destination(for: "a"))
        XCTAssertNil(PushRoute(profileID: "a", destination: "alerts").destination(for: "b"))
        XCTAssertEqual(PushRoute(profileID: "a", destination: "alerts").destination(for: "a"), "alerts")
        let session = SessionStore(clearCachedData: {})
        session.openDestination("alerts")
        XCTAssertTrue(session.showAlerts)
        XCTAssertEqual(session.selectedTab, "home")
        XCTAssertEqual(PushRoute(profileID: "a", destination: "wealth").destination(for: "a"), "wealth")
    }
}
