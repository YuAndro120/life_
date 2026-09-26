import Foundation

/// Подбор законов под профиль (раздел 7 plan.md). Чистые функции, без SwiftData и сети.
enum AudienceMatcher {
    static let allTag = "all"
    static let militaryRegistered = "military:registered"

    static func audienceTags(for profile: UserProfile) -> Set<String> {
        var tags = Set<String>()
        if let g = profile.gender { tags.insert("gender:\(g.rawValue)") }
        if let a = profile.age { tags.insert("age:\(a.rawValue)") }
        for w in profile.work {
            tags.insert("work:\(w.rawValue)")
            // В онбординге нет вопроса про налоговый режим, поэтому ИП получает и УСН-законы:
            // лучше показать лишнее, чем пропустить срок по налогам.
            if w == .ip { tags.insert("work:ip_usn") }
        }
        for h in profile.housing { tags.insert("housing:\(h.rawValue)") }
        for o in profile.occupations { tags.insert("industry:\(o.rawValue)") }
        for x in profile.sells { tags.insert("sells:\(x.rawValue)") }
        if profile.drives == true { tags.insert("transport:driver") }
        if let r = profile.regionCode { tags.insert("region:\(r)") }
        if isMilitaryRegistered(profile) { tags.insert(militaryRegistered) }
        return tags
    }

    /// Мужчина 18–30. Возрастные группы онбординга грубее (до 20, 20–25, 26–35),
    /// поэтому берём их целиком: возможен небольшой перебор, но не пропуск.
    static func isMilitaryRegistered(_ profile: UserProfile) -> Bool {
        guard profile.gender == .male, let age = profile.age else { return false }
        switch age {
        case .u20, .a20to25, .a26to35: return true
        case .a36to50, .a50plus: return false
        }
    }

    /// Закон касается пользователя, если региональность подходит и есть пересечение тегов или тег `all`.
    static func matches(_ law: Law, tags: Set<String>, regionCode: String?) -> Bool {
        if let lawRegion = law.regionCode, lawRegion != regionCode { return false }
        let lawTags = Set(law.audienceTags)
        // Отрасль и товары уточняют выбор: если закон про конкретную отрасль или торговлю, а человек эти ответы дал,
        // они должны совпасть. Если ответов нет, не отсекаем: лучше показать лишнее, чем пропустить нужное.
        for prefix in ["industry:", "sells:"] {
            let lawSpecific = lawTags.filter { $0.hasPrefix(prefix) }
            let mine = tags.filter { $0.hasPrefix(prefix) }
            if !lawSpecific.isEmpty, !mine.isEmpty, lawSpecific.isDisjoint(with: mine) { return false }
        }
        // Тег «all» решает, только когда других тегов у закона нет: модель ставит его вместе с конкретными, и тогда важнее они.
        let specific = lawTags.subtracting([allTag])
        if specific.isEmpty { return lawTags.contains(allTag) }
        return !specific.isDisjoint(with: tags)
    }

    /// Законы про пользователя, по возрастанию даты вступления в силу (без даты — в конце).
    /// `asOf` отсекает уже вступившие в силу до этой даты; `nil` — не отсекать.
    static func relevantLaws(
        _ laws: [Law],
        profile: UserProfile,
        asOf: CalendarDate? = nil
    ) -> [Law] {
        let tags = audienceTags(for: profile)
        return laws
            .filter { matches($0, tags: tags, regionCode: profile.regionCode) }
            .filter { law in
                guard let asOf, let effective = law.dates.effective else { return true }
                return effective >= asOf
            }
            .sorted(by: lawOrder)
    }

    static func lawOrder(_ a: Law, _ b: Law) -> Bool {
        switch (a.dates.effective, b.dates.effective) {
        case let (l?, r?) where l != r: return l < r
        case (nil, _?): return false
        case (_?, nil): return true
        default: return a.id < b.id
        }
    }
}
