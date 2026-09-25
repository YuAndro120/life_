import Foundation

enum APIError: Error, Equatable {
    case badStatus(Int)
    case transport
    case decoding
}

/// Клиент API v1. Запросы одинаковы для всех пользователей: ни профиля, ни идентификаторов не отправляется.
/// Кэширование делает URLCache: запрос с `.reloadRevalidatingCacheData` уходит с If-None-Match,
/// а ответ 304 подменяется сохранённым телом.
struct APIClient: ContentSource {
    let baseURL: URL
    let session: URLSession

    init(baseURL: URL, session: URLSession = APIClient.makeSession()) {
        self.baseURL = baseURL
        self.session = session
    }

    static func makeSession() -> URLSession {
        let config = URLSessionConfiguration.default
        config.requestCachePolicy = .reloadRevalidatingCacheData
        config.urlCache = URLCache(memoryCapacity: 4 << 20, diskCapacity: 20 << 20)
        config.timeoutIntervalForRequest = 15
        config.waitsForConnectivity = false
        config.httpAdditionalHeaders = ["Accept": "application/json"]
        return URLSession(configuration: config)
    }

    func feed() async throws -> Feed {
        try await get("v1/feed")
    }

    func laws() async throws -> [Law] {
        let response: LawsResponse = try await get("v1/laws")
        return response.laws
    }

    private func get<T: Decodable>(_ path: String) async throws -> T {
        let url = baseURL.appending(path: path)
        let data: Data
        let response: URLResponse
        do {
            (data, response) = try await session.data(from: url)
        } catch {
            throw APIError.transport
        }
        guard let http = response as? HTTPURLResponse else { throw APIError.transport }
        guard (200..<300).contains(http.statusCode) else { throw APIError.badStatus(http.statusCode) }
        do {
            return try APICoding.decoder().decode(T.self, from: data)
        } catch {
            throw APIError.decoding
        }
    }
}

/// Адрес API: из Info.plist (`ShtilAPIBaseURL`, build setting `API_BASE_URL`), в DEBUG можно переопределить `-shtilAPI <url>`.
enum AppConfig {
    static var apiBaseURL: URL? {
        if let override = DebugLaunch.apiOverride, let url = URL(string: override) { return url }
        guard let raw = Bundle.main.object(forInfoDictionaryKey: "ShtilAPIBaseURL") as? String else { return nil }
        return URL(string: raw)
    }
}
