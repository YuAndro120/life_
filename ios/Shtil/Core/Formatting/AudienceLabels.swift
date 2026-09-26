import Foundation

extension UserProfile.Gender {
    var title: String {
        switch self {
        case .male: "Мужчина"
        case .female: "Женщина"
        }
    }
}

extension UserProfile.AgeBracket {
    var title: String {
        switch self {
        case .u20: "До 20"
        case .a20to25: "20–25"
        case .a26to35: "26–35"
        case .a36to50: "36–50"
        case .a50plus: "50+"
        }
    }
}

extension UserProfile.Work {
    var title: String {
        switch self {
        case .employee: "По найму"
        case .ip: "ИП"
        case .selfemployed: "Самозанятый"
        case .student: "Студент"
        }
    }
}

extension UserProfile.Housing {
    var title: String {
        switch self {
        case .renter: "Снимаю"
        case .owner: "Своё"
        case .mortgage: "Ипотека"
        }
    }
}

/// Подписи аудитории закона для карточек, календаря и блока «Тебе».
extension UserProfile.Occupation {
    var title: String {
        switch self {
        case .it: "IT и разработка"
        case .trade: "Торговля"
        case .food: "Общепит и гостиницы"
        case .education: "Образование"
        case .health: "Медицина"
        case .construction: "Строительство и ремонт"
        case .transport: "Транспорт и логистика"
        case .industry: "Производство"
        case .agriculture: "Сельское хозяйство"
        case .finance: "Финансы и право"
        case .publicService: "Госслужба и бюджет"
        case .beauty: "Красота и услуги"
        case .creative: "Творчество и медиа"
        }
    }
}

extension UserProfile.Sells {
    var title: String {
        switch self {
        case .goods: "Товары"
        case .marked: "Маркированные товары"
        case .alcohol: "Алкоголь и табак"
        case .food: "Продукты и еда"
        case .online: "Через маркетплейсы"
        case .services: "Услуги"
        case .transport: "Перевозки"
        case .rent: "Аренда и недвижимость"
        }
    }
}

enum AudienceLabels {
    static func label(forTag tag: String) -> String {
        switch tag {
        case "all": "Всех"
        case "work:employee": "Работающие по найму"
        case "work:ip": "ИП"
        case "work:ip_usn": "ИП на УСН"
        case "work:selfemployed": "Самозанятые"
        case "work:student": "Студенты"
        case "housing:renter": "Снимаю жильё"
        case "housing:owner": "Владельцы жилья"
        case "housing:mortgage": "Ипотека"
        case "transport:driver": "Водители"
        case let t where t.hasPrefix("industry:"): UserProfile.Occupation(rawValue: String(t.dropFirst(9)))?.title ?? "Отрасль"
        case let t where t.hasPrefix("sells:"): UserProfile.Sells(rawValue: String(t.dropFirst(6))).map { "Продают: \($0.title.lowercased())" } ?? "Торговля"
        case "military:registered": "Воинский учёт"
        case "gender:male": "Мужчины"
        case "gender:female": "Женщины"
        case "age:u20": "До 20 лет"
        case "age:20_25": "20–25 лет"
        case "age:26_35": "26–35 лет"
        case "age:36_50": "36–50 лет"
        case "age:50p": "Старше 50"
        default:
            tag.hasPrefix("region:") ? "Регион" : tag
        }
    }

    /// Короткая подпись закона: первая метка, совпавшая с профилем, иначе первая содержательная.
    static func label(for law: Law, profileTags: Set<String>) -> String {
        let tags = law.audienceTags.filter { !$0.hasPrefix("region:") }
        let matched = tags.first { profileTags.contains($0) }
        return label(forTag: matched ?? tags.first ?? "all")
    }

    /// «Почему тебе»: ответы профиля, из-за которых закон попал в блок.
    static func reason(for law: Law, profile: UserProfile) -> String {
        let tags = AudienceMatcher.audienceTags(for: profile)
        let lawTags = Set(law.audienceTags)
        if lawTags.contains(AudienceMatcher.allTag) { return "Закон касается всех." }

        var answers: [String] = []
        func add(_ s: String) { if !answers.contains(s) { answers.append(s) } }
        for tag in law.audienceTags where tags.contains(tag) {
            switch tag {
            case "work:ip", "work:ip_usn": add("ИП")
            case "work:employee": add("По найму")
            case "work:selfemployed": add("Самозанятый")
            case "work:student": add("Студент")
            case "housing:renter": add("Снимаю")
            case "housing:owner": add("Своё жильё")
            case "housing:mortgage": add("Ипотека")
            case "transport:driver": add("Вожу авто")
            case "military:registered": add("Мужчина \(profile.age?.title ?? "")".trimmingCharacters(in: .whitespaces))
            case "gender:male": add("Мужчина")
            case "gender:female": add("Женщина")
            default:
                if tag.hasPrefix("industry:"), let o = UserProfile.Occupation(rawValue: String(tag.dropFirst(9))) { add(o.title) }
                if tag.hasPrefix("sells:"), let x = UserProfile.Sells(rawValue: String(tag.dropFirst(6))) { add("Продаю: \(x.title.lowercased())") }
                if tag.hasPrefix("age:"), let a = profile.age { add(a.title) }
                if tag.hasPrefix("region:"), let r = Region.title(for: profile.regionCode) { add(r) }
            }
        }
        guard !answers.isEmpty else { return "Подобрано по вашему профилю." }
        return "В профиле указано: " + answers.map { "«\($0)»" }.joined(separator: ", ") + "."
    }
}
