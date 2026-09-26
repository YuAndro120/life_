import Foundation

/// Всё про СВО скрыто по умолчанию (можно включить в «Фильтрах»). Исключение одно: официальное заявление о том, что СВО закончилась.
/// Определение по словам в заголовке и пересказе; работает только на устройстве.
enum WarMarkers {
    /// (основа слова, сколько букв окончания допускается).
    private static let words: [(stem: String, tail: Int)] = [
        ("сво", 0), ("всу", 0), ("бпла", 1), ("беспилотн", 5), ("спецопераци", 3), ("донбасс", 3), ("днр", 0), ("лнр", 0),
        ("мобилизац", 3), ("обстрел", 3), ("фронт", 2), ("украин", 4), ("зеленск", 4), ("минобороны", 0),
    ]
    private static let phrases = ["специальной военной операции", "специальная военная операция", "вооруженные силы", "вооруженных сил", "боевые действия", "боевых действий"]

    static func matches(_ story: Story) -> Bool {
        let text = normalize(story.title + " " + story.summary)
        if phrases.contains(where: text.contains) { return true }
        return tokens(text).contains { word in words.contains { word.hasPrefix($0.stem) && word.count - $0.stem.count <= $0.tail } }
    }

    private static let lawPhrases = [
        "специальной военной операции", "боевых действий", "вооруженного вторжения", "погибших военнослужащих", "погибших участников",
        "погибших сотрудников", "участников сво", "участникам сво", "участники сво",
    ]

    /// Закон про участников СВО, ветеранов боевых действий и семьи погибших: скрывается тем же переключателем, что и новости.
    static func matches(_ law: Law) -> Bool {
        let text = normalize(law.title + " " + law.whatChanged + " " + law.whoAffected)
        if lawPhrases.contains(where: text.contains) { return true }
        return tokens(text).contains("сво")
    }

    private static let endWords = ["заверш", "окончен", "окончани", "прекращ"]

    /// Официальное заявление о том, что СВО закончилась: тип «официальное», упомянута СВО и слово об окончании.
    static func isEndAnnouncement(_ story: Story) -> Bool {
        guard story.infoType == .official else { return false }
        let text = normalize(story.title + " " + story.summary)
        let mentionsWar = text.contains("специальной военной операции") || text.contains("специальная военная операция") || tokens(text).contains("сво")
        guard mentionsWar else { return false }
        return tokens(text).contains { word in endWords.contains { word.hasPrefix($0) } }
    }

    private static func normalize(_ s: String) -> String {
        s.lowercased().replacingOccurrences(of: "ё", with: "е")
    }

    private static func tokens(_ text: String) -> [String] {
        text.split(whereSeparator: { !$0.isLetter }).map(String.init)
    }
}
