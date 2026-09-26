import Foundation

/// Фильтр по словам пользователя: сюжет скрывается, если слово (или фраза) есть в заголовке, пересказе или «значит».
/// Слово ищется по началу слов, поэтому «футбол» найдёт и «футболист»; фраза из нескольких слов ищется как подстрока.
enum WordFilter {
    static let maxWords = 50
    static let maxLength = 40

    /// Приводит слово к виду для хранения: без пробелов по краям, строчные, «ё» → «е». Пустая строка, если слово не подходит.
    static func normalize(_ raw: String) -> String {
        let s = raw.trimmingCharacters(in: .whitespacesAndNewlines).lowercased().replacingOccurrences(of: "ё", with: "е")
        let squeezed = s.split(whereSeparator: \.isWhitespace).joined(separator: " ")
        return squeezed.count >= 2 && squeezed.count <= maxLength ? squeezed : ""
    }

    static func matches(_ story: Story, words: [String]) -> Bool {
        guard !words.isEmpty else { return false }
        let text = (story.title + " " + story.summary + " " + (story.meaning ?? "")).lowercased().replacingOccurrences(of: "ё", with: "е")
        let tokens = text.split(whereSeparator: { !$0.isLetter && !$0.isNumber }).map(String.init)
        return words.contains { word in
            word.contains(" ") ? text.contains(word) : tokens.contains { $0.hasPrefix(word) }
        }
    }
}
