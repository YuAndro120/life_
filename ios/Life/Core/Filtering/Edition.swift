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
    /// Сюжетов убрано фильтрами на устройстве (стоп-темы, типы информации, режим тяжёлых).
    var filteredOut: Int
}

struct Edition: Equatable, Sendable {
    var number: Int
    var laws: [Law]
    var stories: [Story]
    /// Тяжёлые сюжеты, свёрнутые в одну сводку в конце выпуска.
    var foldedHeavy: [Story]
    var stats: EditionStats
}

/// Окно выпуска: (start, end]. Сюжеты, обновлённые позже `end`, попадут в следующий выпуск.
struct EditionWindow: Equatable, Sendable {
    var start: Date?
    var end: Date
}
