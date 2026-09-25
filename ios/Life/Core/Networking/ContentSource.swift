import Foundation

/// Откуда берутся сюжеты и законы. В фазе 1 — фикстуры, в фазе 2 — API.
protocol ContentSource: Sendable {
    func feed() async throws -> Feed
    func laws() async throws -> [Law]
}

struct FixtureContentSource: ContentSource {
    func feed() async throws -> Feed { try Fixtures.feed() }
    func laws() async throws -> [Law] { try Fixtures.laws() }
}
