import Foundation

/// Фикстуры в формате API. Используются, пока нет сервера (фаза 1), и в тестах.
enum Fixtures {
    enum FixtureError: Error { case missing(String) }

    static func feed(bundle: Bundle = .main) throws -> Feed {
        try APICoding.decoder().decode(Feed.self, from: data("feed", bundle))
    }

    static func laws(bundle: Bundle = .main) throws -> [Law] {
        try APICoding.decoder().decode(LawsResponse.self, from: data("laws", bundle)).laws
    }

    private static func data(_ name: String, _ bundle: Bundle) throws -> Data {
        guard let url = bundle.url(forResource: name, withExtension: "json") else {
            throw FixtureError.missing(name)
        }
        return try Data(contentsOf: url)
    }
}
