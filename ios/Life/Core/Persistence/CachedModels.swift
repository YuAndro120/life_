import Foundation
import SwiftData

/// Последний ответ API. Тело хранится JSON-ом в формате API, чтобы кэш переживал добавление полей.
@Model
final class CachedStory {
    @Attribute(.unique) var id: String
    var updatedAt: Date
    var payload: Data

    init(id: String, updatedAt: Date, payload: Data) {
        self.id = id
        self.updatedAt = updatedAt
        self.payload = payload
    }
}

@Model
final class CachedLaw {
    @Attribute(.unique) var id: String
    var payload: Data

    init(id: String, payload: Data) {
        self.id = id
        self.payload = payload
    }
}

/// Напоминание о вступлении закона в силу.
@Model
final class Reminder {
    @Attribute(.unique) var lawId: String
    var fireDate: Date

    init(lawId: String, fireDate: Date) {
        self.lawId = lawId
        self.fireDate = fireDate
    }
}

/// Локальный счётчик номера выпуска.
@Model
final class EditionCounter {
    var number: Int
    var lastEditionAt: Date?

    init(number: Int = 0, lastEditionAt: Date? = nil) {
        self.number = number
        self.lastEditionAt = lastEditionAt
    }
}

/// Метаданные последнего успешного ответа ленты. Одна запись.
@Model
final class CachedMeta {
    var feedGeneratedAt: Date
    var postsTotal: Int
    var adsHidden: Int
    /// Когда приложение получило эти данные с сервера.
    var fetchedAt: Date

    init(feedGeneratedAt: Date, postsTotal: Int, adsHidden: Int, fetchedAt: Date) {
        self.feedGeneratedAt = feedGeneratedAt
        self.postsTotal = postsTotal
        self.adsHidden = adsHidden
        self.fetchedAt = fetchedAt
    }
}
