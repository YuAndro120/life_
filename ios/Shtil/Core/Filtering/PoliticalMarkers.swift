import Foundation

/// Запрет темы «Политика» ловит и сюжеты, которые модель отнесла к другой теме (например, «Путин выступил на форуме культур»).
/// Признак: в заголовке или пересказе названы главные политические фигуры и институты. Работает только на устройстве.
enum PoliticalMarkers {
    /// (основа слова, сколько букв окончания допускается). Короткий допуск отсекает «трамплин» и подобное.
    private static let markers: [(stem: String, tail: Int)] = [
        ("путин", 2), ("кремл", 2), ("песков", 2), ("лавров", 2), ("мишустин", 2), ("зеленск", 4),
        ("трамп", 2), ("госдум", 2), ("володин", 2), ("макрон", 2), ("шольц", 2), ("эрдоган", 2), ("байден", 2),
    ]

    static func matches(_ story: Story) -> Bool {
        let words = tokens(story.title + " " + story.summary)
        for word in words {
            for m in markers where word.hasPrefix(m.stem) && word.count - m.stem.count <= m.tail {
                return true
            }
        }
        return false
    }

    private static func tokens(_ text: String) -> [String] {
        text.lowercased()
            .replacingOccurrences(of: "ё", with: "е")
            .split(whereSeparator: { !$0.isLetter })
            .map(String.init)
    }
}
