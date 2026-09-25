import Foundation

struct EditionStats: Equatable, Sendable {
    /// Законов в блоке «Касается тебя».
    var aboutYou: Int
    /// Сюжетов в основной части выпуска.
    var stories: Int
    /// Оценка времени чтения: символы / 1200 в минуту, округление вверх.
    var readingMinutes: Int
    /// Всего постов, собранных сервером за сутки.
    var postsTotal: Int
    /// Скрыто рекламы на сервере.
    var adsHidden: Int
    /// Сюжетов убрано фильтрами на устройстве (стоп-темы, типы информации, страны, скрытые, режим тяжёлых).
    var filteredOut: Int
    /// Сюжетов, не вошедших в выпуск из-за лимита «сюжетов в выпуске».
    var trimmed: Int = 0
}

struct Edition: Equatable, Sendable {
    var number: Int
    var laws: [Law]
    var stories: [Story]
    /// Тяжёлые сюжеты, свёрнутые в одну сводку в конце выпуска.
    var foldedHeavy: [Story]
    /// Сюжеты выпуска, попавшие в «Мои интересы».
    var interestIDs: Set<String> = []
    var stats: EditionStats
}

/// Окно выпуска: (start, end]. Сюжеты, обновлённые позже `end`, попадут в следующий выпуск.
struct EditionWindow: Equatable, Sendable {
    var start: Date?
    var end: Date
}
