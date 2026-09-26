import Foundation

/// Разбор текста «расскажите о себе» на устройстве: регион, работа, жильё, авто, интересы, что не показывать.
/// Текст никуда не отправляется. Правила по словам; результат показывается плашками, и человек может любую убрать.
struct AboutParse: Equatable, Sendable {
    enum Kind: String, Sendable { case region, work, housing, drives, gender, interest, mute, word }

    struct Chip: Identifiable, Equatable, Sendable {
        let kind: Kind
        let key: String
        let title: String
        var id: String { "\(kind.rawValue):\(key)" }
    }

    var regionCode: String?
    var gender: UserProfile.Gender?
    var work: Set<UserProfile.Work> = []
    var housing: Set<UserProfile.Housing> = []
    var drives: Bool?
    var interests: Set<Topic> = []
    var mutedTopics: Set<Topic> = []
    var blockedWords: [String] = []

    var isEmpty: Bool { chips.isEmpty }

    /// Плашки для показа, в порядке: где, кто, что интересно, что скрыть.
    var chips: [Chip] {
        var out: [Chip] = []
        if let code = regionCode, let title = Region.title(for: code) { out.append(.init(kind: .region, key: code, title: title)) }
        if let gender { out.append(.init(kind: .gender, key: gender.rawValue, title: gender.title)) }
        for w in UserProfile.Work.allCases where work.contains(w) { out.append(.init(kind: .work, key: w.rawValue, title: w.title)) }
        for h in UserProfile.Housing.allCases where housing.contains(h) { out.append(.init(kind: .housing, key: h.rawValue, title: h.title)) }
        if let drives { out.append(.init(kind: .drives, key: "\(drives)", title: drives ? "Вожу авто" : "Не вожу")) }
        for t in Topic.allCases where interests.contains(t) { out.append(.init(kind: .interest, key: t.rawValue, title: t.title)) }
        for t in Topic.allCases where mutedTopics.contains(t) { out.append(.init(kind: .mute, key: t.rawValue, title: "Скрыть: \(t.title)")) }
        for w in blockedWords { out.append(.init(kind: .word, key: w, title: "Скрыть слово: \(w)")) }
        return out
    }

    /// Разбор без плашек из `dismissed` (их убрал пользователь).
    func without(_ dismissed: Set<String>) -> AboutParse {
        var r = self
        for chip in chips where dismissed.contains(chip.id) {
            switch chip.kind {
            case .region: r.regionCode = nil
            case .gender: r.gender = nil
            case .work: if let w = UserProfile.Work(rawValue: chip.key) { r.work.remove(w) }
            case .housing: if let h = UserProfile.Housing(rawValue: chip.key) { r.housing.remove(h) }
            case .drives: r.drives = nil
            case .interest: if let t = Topic(rawValue: chip.key) { r.interests.remove(t) }
            case .mute: if let t = Topic(rawValue: chip.key) { r.mutedTopics.remove(t) }
            case .word: r.blockedWords.removeAll { $0 == chip.key }
            }
        }
        return r
    }

    /// Дополняет профиль и настройки: найденное добавляется, ранее выбранное не стирается.
    func apply(to profile: inout UserProfile, preferences: inout FilterPreferences) {
        if let regionCode { profile.regionCode = regionCode }
        if let gender { profile.gender = gender }
        profile.work.formUnion(work)
        profile.housing.formUnion(housing)
        if let drives { profile.drives = drives }
        for t in interests { preferences.toggleInterestIfNeeded(t) }
        for t in mutedTopics where !preferences.interests.contains(t) { preferences.stopTopics.insert(t) }
        for w in blockedWords where !preferences.blockedWords.contains(w) && preferences.blockedWords.count < WordFilter.maxWords {
            preferences.blockedWords.append(w)
        }
    }
}

extension FilterPreferences {
    fileprivate mutating func toggleInterestIfNeeded(_ topic: Topic) {
        if !interests.contains(topic) { toggleInterest(topic) }
    }
}

enum AboutParser {
    static func parse(_ raw: String) -> AboutParse {
        let text = raw.lowercased().replacingOccurrences(of: "ё", with: "е")
        var r = AboutParse()
        guard !text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else { return r }

        // Отрицательные фразы разбираем отдельно, чтобы «не люблю футбол» не стало интересом «Спорт».
        let (positive, negative) = split(text)

        r.regionCode = region(in: positive, fullText: text)
        let words = tokens(positive)
        let has = { (stems: [String]) in words.contains { w in stems.contains { w.hasPrefix($0) } } }
        let exact = { (list: [String]) in words.contains { list.contains($0) } }

        if exact(["ип"]) || has(["предпринимател"]) { r.work.insert(.ip) }
        if has(["самозанят"]) { r.work.insert(.selfemployed) }
        if has(["студент", "учусь", "школьник"]) { r.work.insert(.student) }
        if positive.contains("по найму") || has(["наемн", "сотрудник"]) { r.work.insert(.employee) }

        if has(["снимаю", "аренд"]) { r.housing.insert(.renter) }
        if positive.contains("своя квартира") || positive.contains("свое жилье") || has(["собственник", "владелец"]) { r.housing.insert(.owner) }
        if has(["ипотек"]) { r.housing.formUnion([.mortgage]) }

        if text.contains("не вожу") || text.contains("без машины") || text.contains("без авто") || text.contains("нет машины") {
            r.drives = false
        } else if has(["вожу", "водител", "автомобил", "машин"]) || positive.contains("за рулем") {
            r.drives = true
        }

        if has(["мужчин", "парень"]) {
            r.gender = .male
        } else if has(["женщин", "девушк"]) || exact(["мама"]) {
            r.gender = .female
        }

        r.interests = topics(in: words)
        let neg = negativeTerms(negative)
        for term in neg {
            let matched = topics(in: [term])
            if matched.isEmpty {
                let w = WordFilter.normalize(stem(term))
                if !w.isEmpty, !r.blockedWords.contains(w) { r.blockedWords.append(w) }
            } else {
                r.mutedTopics.formUnion(matched)
                r.interests.subtract(matched)
            }
        }
        return r
    }

    // MARK: разбор

    private static let negativeCues = ["не люблю", "не хочу", "не интересует", "не интересно", "надоел", "надоела", "надоело", "надоели", "не читаю", "без ", "устал от", "устала от", "бесит", "бесят"]

    /// Делит текст на предложения; в отрицательные попадает то, что идёт после подсказок вроде «не люблю».
    private static func split(_ text: String) -> (positive: String, negative: String) {
        var positive: [String] = []
        var negative: [String] = []
        for sentence in text.split(whereSeparator: { ".!?;\n".contains($0) }).map(String.init) {
            var handled = false
            for cue in negativeCues {
                if let range = sentence.range(of: cue) {
                    positive.append(String(sentence[..<range.lowerBound]))
                    negative.append(String(sentence[range.upperBound...]))
                    handled = true
                    break
                }
            }
            if !handled { positive.append(sentence) }
        }
        return (positive.joined(separator: ". "), negative.joined(separator: ". "))
    }

    private static let stopWords: Set<String> = [
        "и", "а", "но", "или", "в", "на", "с", "по", "про", "о", "об", "от", "до", "для", "как", "что", "это", "все", "всё", "весь",
        "надоел", "надоела", "надоело", "надоели", "бесит", "бесят", "устал", "устала", "люблю", "хочу", "очень", "сильно", "новости", "новостей", "новость", "любые", "разные", "такие", "тоже", "уже", "просто", "чтобы", "когда",
    ]

    /// Слова после отрицательной подсказки до ближайшего «но»: «не люблю футбол и политику» → [футбол, политику].
    private static func negativeTerms(_ negative: String) -> [String] {
        let words = tokens(negative)
        var out: [String] = []
        for w in words {
            if w == "но" || w == "зато" { break }
            if stopWords.contains(w) || w.count < 3 { continue }
            if !out.contains(w) { out.append(w) }
            if out.count == 6 { break }
        }
        return out
    }

    /// Основа слова для поиска по началу слов: «политику» → «политик».
    private static func stem(_ word: String) -> String {
        word.count >= 6 ? String(word.dropLast(1)) : word
    }

    private static func region(in positive: String, fullText: String) -> String? {
        // «Живу в Казани, работаю в Москве»: предпочитаем место после «живу».
        for cue in ["живу", "проживаю", "из города", "родом из", "переехал в", "переехала в"] {
            if let range = positive.range(of: cue), let item = Region.find(in: String(positive[range.upperBound...])) { return item.code }
        }
        return Region.find(in: positive)?.code
    }

    private static let topicWords: [(Topic, [String])] = [
        (.space, ["космос", "космичес", "наса", "nasa", "роскосмос", "астроном", "ракет", "spacex"]),
        (.science, ["наук", "учен", "физик", "биолог", "хими", "исследован"]),
        (.techAi, ["технолог", "нейросет", "ии", "ai", "айти", "it", "программир", "гаджет", "стартап"]),
        (.sport, ["спорт", "футбол", "хоккей", "теннис", "баскетбол", "бокс", "формул"]),
        (.culture, ["кино", "фильм", "музык", "театр", "книг", "литератур", "искусств", "выставк", "культур"]),
        (.showbiz, ["шоубиз", "звезд", "знаменитост"]),
        (.crypto, ["крипт", "биткоин", "bitcoin", "блокчейн"]),
        (.health, ["здоровь", "медицин", "врач", "лекарств"]),
        (.education, ["образован", "школ", "университет", "егэ", "обучени"]),
        (.economy, ["экономик", "бизнес", "инвестиц", "рынк"]),
        (.finance, ["финанс", "вклад", "кредит", "акци", "курс"]),
        (.housing, ["недвижим", "жкх"]),
        (.transport, ["транспорт", "метро", "авиа", "самолет", "поезд"]),
        (.politics, ["политик", "выбор", "депутат", "власт"]),
        (.crime, ["криминал", "преступлен", "происшествия"]),
    ]

    private static func topics(in words: [String]) -> Set<Topic> {
        var out = Set<Topic>()
        for w in words {
            for (topic, stems) in topicWords {
                // Латинские и очень короткие основы («ии», «it») должны совпадать целиком, остальные — по началу слова.
                if stems.contains(where: { $0.count <= 3 ? w == $0 : w.hasPrefix($0) }) { out.insert(topic) }
            }
        }
        return out
    }

    private static func tokens(_ text: String) -> [String] {
        text.split(whereSeparator: { !$0.isLetter && !$0.isNumber }).map(String.init)
    }
}
