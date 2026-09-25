import Foundation
import SwiftData
import Testing
@testable import Life

/// Заглушка сети: обработчик выбирается по хосту, поэтому параллельные тесты не мешают друг другу.
final class StubProtocol: URLProtocol, @unchecked Sendable {
    typealias Handler = @Sendable (URLRequest) throws -> (HTTPURLResponse, Data)
    private static let lock = NSLock()
    nonisolated(unsafe) private static var handlers: [String: Handler] = [:]

    static func register(host: String, _ handler: @escaping Handler) {
        lock.withLock { handlers[host] = handler }
    }

    override class func canInit(with request: URLRequest) -> Bool { true }
    override class func canonicalRequest(for request: URLRequest) -> URLRequest { request }

    override func startLoading() {
        let handler = Self.lock.withLock { Self.handlers[request.url?.host() ?? ""] }
        guard let handler else {
            client?.urlProtocol(self, didFailWithError: URLError(.cannotConnectToHost))
            return
        }
        do {
            let (response, data) = try handler(request)
            client?.urlProtocol(self, didReceive: response, cacheStoragePolicy: .notAllowed)
            client?.urlProtocol(self, didLoad: data)
            client?.urlProtocolDidFinishLoading(self)
        } catch {
            client?.urlProtocol(self, didFailWithError: error)
        }
    }

    override func stopLoading() {}

    static func session() -> URLSession {
        let config = URLSessionConfiguration.ephemeral
        config.protocolClasses = [StubProtocol.self]
        return URLSession(configuration: config)
    }

    static func response(_ url: URL, status: Int = 200) -> HTTPURLResponse {
        HTTPURLResponse(url: url, statusCode: status, httpVersion: "HTTP/1.1", headerFields: ["Content-Type": "application/json"])!
    }
}

/// Ответ реального сервера (`curl localhost:8080/v1/feed`): даты с микросекундами, null в необязательных полях.
private let serverFeedJSON = """
{"generated_at":"2026-09-25T07:04:42.516855Z","stories":[{"id":"st_1","topic":"economy","info_type":"official",
"heaviness":"neutral","title":"Банк России объявил решение по ключевой ставке","meaning":"Условия не изменятся.",
"summary":"Совет директоров принял решение.","post_count":9,"source_count":5,
"sources":[{"title":"Банк России","url":"https://www.cbr.ru/#post-1"}],"region_code":null,
"updated_at":"2026-09-25T07:04:42.516855Z"}],"stats":{"posts_total":57,"ads_hidden":7}}
"""

private let serverLawsJSON = """
{"laws":[{"id":"lw_1","title":"Меняется срок уведомлений","what_changed":"Что","who_affected":"Кого",
"actions":["Проверить автоплатёж"],"audience_tags":["work:ip_usn"],"region_code":null,"status":"signed",
"dates":{"introduced":"2026-06-10","passed":"2026-08-05","signed":"2026-08-20","effective":"2026-10-01"},
"official_url":"https://publication.pravo.gov.ru/","bill_url":null,"act_number":"№ 000-ФЗ",
"verified_at":"2026-09-25T08:52:42.516855Z"}]}
"""

@Suite struct APIClientTests {
    private func client(host: String, _ handler: @escaping StubProtocol.Handler) -> APIClient {
        StubProtocol.register(host: host, handler)
        return APIClient(baseURL: URL(string: "http://\(host):8080")!, session: StubProtocol.session())
    }

    @Test func decodesRealServerFeed() async throws {
        let api = client(host: "feed-ok.test") { req in (StubProtocol.response(req.url!), Data(serverFeedJSON.utf8)) }
        let feed = try await api.feed()
        #expect(feed.stories.count == 1)
        #expect(feed.stories[0].sourceCount == 5)
        #expect(feed.stories[0].regionCode == nil)
        #expect(feed.stats.adsHidden == 7)
        #expect(feed.generatedAt == feed.stories[0].updatedAt)
    }

    @Test func decodesRealServerLaws() async throws {
        let api = client(host: "laws-ok.test") { req in (StubProtocol.response(req.url!), Data(serverLawsJSON.utf8)) }
        let laws = try await api.laws()
        #expect(laws.count == 1)
        #expect(laws[0].dates.effective == CalendarDate(year: 2026, month: 10, day: 1))
        #expect(laws[0].billUrl == nil)
        #expect(laws[0].verifiedAt != nil)
    }

    @Test func requestsExpectedPaths() async throws {
        let seen = LockedBox<[String]>([])
        let api = client(host: "paths.test") { req in
            seen.mutate { $0.append(req.url?.path() ?? "") }
            let body = req.url?.path().hasSuffix("laws") == true ? serverLawsJSON : serverFeedJSON
            return (StubProtocol.response(req.url!), Data(body.utf8))
        }
        _ = try await api.feed()
        _ = try await api.laws()
        #expect(seen.value == ["/v1/feed", "/v1/laws"])
    }

    @Test func requestCarriesNoUserData() async throws {
        let captured = LockedBox<URLRequest?>(nil)
        let api = client(host: "privacy.test") { req in
            captured.mutate { $0 = req }
            return (StubProtocol.response(req.url!), Data(serverFeedJSON.utf8))
        }
        _ = try await api.feed()
        let req = try #require(captured.value)
        #expect(req.url?.query == nil)
        #expect(req.httpBody == nil)
        #expect(req.value(forHTTPHeaderField: "Authorization") == nil)
        #expect(req.value(forHTTPHeaderField: "Cookie") == nil)
    }

    @Test func serverErrorThrowsBadStatus() async {
        let api = client(host: "err500.test") { req in (StubProtocol.response(req.url!, status: 500), Data()) }
        await #expect(throws: APIError.badStatus(500)) { try await api.feed() }
    }

    @Test func networkFailureThrowsTransport() async {
        // Хост без зарегистрированного обработчика.
        let api = APIClient(baseURL: URL(string: "http://nobody.test:8080")!, session: StubProtocol.session())
        await #expect(throws: APIError.transport) { try await api.feed() }
    }

    @Test func garbageThrowsDecoding() async {
        let api = client(host: "garbage.test") { req in (StubProtocol.response(req.url!), Data("не json".utf8)) }
        await #expect(throws: APIError.decoding) { try await api.feed() }
    }
}

/// Потокобезопасная коробка для значений, которые меняет заглушка сети.
final class LockedBox<T>: @unchecked Sendable {
    private let lock = NSLock()
    private var stored: T
    init(_ value: T) { stored = value }
    var value: T { lock.withLock { stored } }
    func mutate(_ change: (inout T) -> Void) { lock.withLock { change(&stored) } }
}

// MARK: - кэш и офлайн

struct FailingSource: ContentSource {
    func feed() async throws -> Feed { throw APIError.transport }
    func laws() async throws -> [Law] { throw APIError.transport }
}

@Suite @MainActor struct OfflineTests {
    private func container() throws -> ModelContainer { try PersistenceStack.makeContainer(inMemory: true) }

    @Test func cacheRoundTrip() throws {
        let container = try container()
        let context = ModelContext(container)
        let cache = ContentCache(context: context)
        let feed = try Fixtures.feed()
        let laws = try Fixtures.laws()
        let fetched = Date(timeIntervalSince1970: 1_790_000_000)
        cache.save(feed: feed, laws: laws, fetchedAt: fetched)

        let loaded = try #require(cache.load())
        #expect(Set(loaded.feed.stories.map(\.id)) == Set(feed.stories.map(\.id)))
        #expect(loaded.feed.stats == feed.stats)
        #expect(loaded.laws.count == laws.count)
        #expect(loaded.fetchedAt == fetched)
        let story = try #require(loaded.feed.stories.first { $0.id == "st_01" })
        #expect(story.sources.first?.url.absoluteString == "https://www.cbr.ru/")
        #expect(story.updatedAt == feed.stories.first { $0.id == "st_01" }?.updatedAt)
        #expect(loaded.laws.first { $0.id == "lw_01" }?.dates.effective == CalendarDate(year: 2026, month: 10, day: 1))
    }

    @Test func emptyCacheLoadsNothing() throws {
        let container = try container()
        #expect(ContentCache(context: ModelContext(container)).load() == nil)
    }

    @Test func saveReplacesPreviousSnapshot() throws {
        let container = try container()
        let context = ModelContext(container)
        let cache = ContentCache(context: context)
        let feed = try Fixtures.feed()
        cache.save(feed: feed, laws: try Fixtures.laws(), fetchedAt: .now)
        let smaller = Feed(generatedAt: feed.generatedAt, stories: Array(feed.stories.prefix(2)), stats: feed.stats)
        cache.save(feed: smaller, laws: [], fetchedAt: .now)
        let loaded = try #require(cache.load())
        #expect(loaded.feed.stories.count == 2)
        #expect(loaded.laws.isEmpty)
    }

    @Test func modelShowsCachedEditionWhenServerIsDown() async throws {
        let container = try container()
        let context = container.mainContext

        let online = AppModel(context: context, source: FixtureContentSource())
        await online.refresh()
        #expect(online.edition != nil)
        #expect(!online.isOffline)

        // «Перезапуск приложения» без сети: тот же диск, источник недоступен.
        let offline = AppModel(context: context, source: FailingSource())
        #expect(offline.edition != nil, "кэш должен подхватиться сразу при старте")
        await offline.refresh()
        #expect(offline.isOffline)
        #expect(offline.loadError == nil)
        #expect(offline.edition?.stories.isEmpty == false)
        #expect(offline.lastSyncedAt != nil)
    }

    @Test func firstLaunchWithoutNetworkShowsError() async throws {
        let container = try container()
        let model = AppModel(context: container.mainContext, source: FailingSource())
        await model.refresh()
        #expect(model.edition == nil)
        #expect(model.loadError != nil)
        #expect(!model.isOffline)
    }

    @Test func firstLaunchWithoutNetworkUsesFallbackButDoesNotCacheIt() async throws {
        let container = try container()
        let model = AppModel(context: container.mainContext, source: FailingSource(), fallback: FixtureContentSource())
        await model.refresh()
        #expect(model.edition != nil)
        #expect(model.isDemoData)
        #expect(model.loadError == nil)
        #expect(ContentCache(context: container.mainContext).load() == nil, "демо-данные не должны попасть в кэш")

        // Сервер появился: демо-режим выключается.
        let live = AppModel(context: container.mainContext, source: FixtureContentSource(), fallback: FixtureContentSource())
        await live.refresh()
        #expect(!live.isDemoData)
    }

    @Test func fallbackIsNotUsedWhenCacheExists() async throws {
        let container = try container()
        let seeded = AppModel(context: container.mainContext, source: FixtureContentSource())
        await seeded.refresh()
        let model = AppModel(context: container.mainContext, source: FailingSource(), fallback: FixtureContentSource())
        await model.refresh()
        #expect(model.isOffline)
        #expect(!model.isDemoData)
    }

    @Test func successfulRefreshClearsOfflineFlag() async throws {
        let container = try container()
        let context = container.mainContext
        let seeded = AppModel(context: context, source: FixtureContentSource())
        await seeded.refresh()
        let model = AppModel(context: context, source: FailingSource())
        await model.refresh()
        #expect(model.isOffline)
        let recovered = AppModel(context: context, source: FixtureContentSource())
        await recovered.refresh()
        #expect(!recovered.isOffline)
    }
}
