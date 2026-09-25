import Foundation

// DTO в формате API v1 (раздел 8 plan.md). Ответы одинаковы для всех пользователей.

struct SourceLink: Codable, Hashable, Sendable {
    let title: String
    let url: URL
}

struct Story: Codable, Hashable, Identifiable, Sendable {
    let id: String
    let topic: Topic
    let infoType: InfoType
    let heaviness: Heaviness
    let title: String
    let meaning: String?
    let summary: String
    let postCount: Int
    let sourceCount: Int
    let sources: [SourceLink]
    let regionCode: String?
    /// Страна большинства источников (RU, US, GB, EU). Нет у старых записей: считается «RU».
    let country: String?
    let lang: String?
    let updatedAt: Date

    var countryCode: String { country ?? "RU" }
}

struct FeedStats: Codable, Hashable, Sendable {
    let postsTotal: Int
    let adsHidden: Int
}

struct Feed: Codable, Sendable {
    let generatedAt: Date
    let stories: [Story]
    let stats: FeedStats

    init(generatedAt: Date, stories: [Story], stats: FeedStats) {
        self.generatedAt = generatedAt
        self.stories = stories
        self.stats = stats
    }

    /// Сюжет с неизвестным значением таксономии (сервер мог добавить тему) пропускается,
    /// а не ломает всю ленту.
    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        generatedAt = try c.decode(Date.self, forKey: .generatedAt)
        stats = try c.decode(FeedStats.self, forKey: .stats)
        stories = try c.decode([Lossy<Story>].self, forKey: .stories).compactMap(\.value)
    }

    enum CodingKeys: String, CodingKey { case generatedAt, stories, stats }
}

struct LawDates: Codable, Hashable, Sendable {
    let introduced: CalendarDate?
    let passed: CalendarDate?
    let signed: CalendarDate?
    let effective: CalendarDate?
}

struct Law: Codable, Hashable, Identifiable, Sendable {
    let id: String
    let title: String
    let whatChanged: String
    let whoAffected: String
    let actions: [String]
    let audienceTags: [String]
    let regionCode: String?
    let status: LawStatus
    let dates: LawDates
    let officialUrl: URL?
    let billUrl: URL?
    let actNumber: String?
    let verifiedAt: Date?
}

struct LawsResponse: Codable, Sendable {
    let laws: [Law]

    init(laws: [Law]) { self.laws = laws }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        laws = try c.decode([Lossy<Law>].self, forKey: .laws).compactMap(\.value)
    }

    enum CodingKeys: String, CodingKey { case laws }
}

/// Декодирует элемент, а при ошибке даёт nil.
struct Lossy<T: Decodable & Sendable>: Decodable, Sendable {
    let value: T?
    init(from decoder: Decoder) throws {
        value = try? T(from: decoder)
    }
}

enum APICoding {
    static func encoder() -> JSONEncoder {
        let e = JSONEncoder()
        e.keyEncodingStrategy = .convertToSnakeCase
        e.dateEncodingStrategy = .iso8601
        return e
    }

    static func decoder() -> JSONDecoder {
        let d = JSONDecoder()
        d.keyDecodingStrategy = .convertFromSnakeCase
        d.dateDecodingStrategy = .custom { decoder in
            let raw = try decoder.singleValueContainer().decode(String.self)
            let plain = ISO8601DateFormatter()
            let fractional = ISO8601DateFormatter()
            fractional.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
            guard let date = plain.date(from: raw) ?? fractional.date(from: raw) else {
                throw DecodingError.dataCorrupted(
                    .init(codingPath: decoder.codingPath, debugDescription: "Неверная дата: \(raw)")
                )
            }
            return date
        }
        return d
    }
}
