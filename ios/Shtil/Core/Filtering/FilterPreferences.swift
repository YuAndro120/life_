import Foundation

enum HeavyMode: String, Codable, CaseIterable, Sendable {
    case hide, fold, show
}

/// Настройки, влияющие на состав выпуска. Тема и расписание сюда не входят.
struct FilterPreferences: Equatable, Sendable {
    var calmMode: Bool
    var infoTypes: Set<InfoType>
    var heavyMode: HeavyMode
    /// Сколько тяжёлых сюжетов показывать в режиме `.show`; остальные сворачиваются в сводку.
    var maxHeavy: Int
    var stopTopics: Set<Topic>
    var hideAds: Bool
    /// Откуда читать новости (коды стран изданий). По умолчанию только Россия.
    var countries: Set<String> = ["RU"]
    /// Темы, которые интересны: такие сюжеты идут первыми.
    var interests: Set<Topic> = []
    /// Показывать только сюжеты по интересам (если интересы выбраны).
    var onlyInterests: Bool = false
    /// Названия источников, скрытых пользователем («Не интересно» → источник).
    var mutedSources: Set<String> = []
    /// Сюжеты, скрытые пользователем.
    var hiddenStories: Set<String> = []
    /// Скрывать всё про СВО (кроме официального заявления об окончании). По умолчанию включено.
    var hideWar: Bool = true
    /// Слова и фразы пользователя: сюжеты с ними скрываются.
    var blockedWords: [String] = []
    /// Сколько сюжетов показывать в выпуске (выпуск можно дочитать до конца).
    var storyLimit: Int = 15

    mutating func toggleInterest(_ topic: Topic) {
        if interests.contains(topic) {
            interests.remove(topic)
        } else {
            interests.insert(topic)
            stopTopics.remove(topic) // тема не может быть одновременно интересной и скрытой
        }
    }

    mutating func toggleStopTopic(_ topic: Topic) {
        if stopTopics.contains(topic) {
            stopTopics.remove(topic)
        } else {
            stopTopics.insert(topic)
            interests.remove(topic)
        }
    }

    static let `default` = FilterPreferences(
        calmMode: true,
        infoTypes: [.fact, .official],
        heavyMode: .fold,
        maxHeavy: 3,
        stopTopics: [.politics, .crime],
        hideAds: true
    )
}
