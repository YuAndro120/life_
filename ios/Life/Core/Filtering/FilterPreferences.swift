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

    static let `default` = FilterPreferences(
        calmMode: true,
        infoTypes: [.fact, .official],
        heavyMode: .fold,
        maxHeavy: 3,
        stopTopics: [.politics, .crime],
        hideAds: true
    )
}
