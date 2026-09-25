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

enum Region {
    struct Item: Hashable, Identifiable, Sendable {
        let code: String
        let title: String
        var id: String { code }
    }

    /// Стартовый список для MVP; полный список и региональные источники — открытый вопрос плана.
    static let all: [Item] = [
        .init(code: "77", title: "Москва"),
        .init(code: "78", title: "Санкт-Петербург"),
        .init(code: "50", title: "Московская область"),
        .init(code: "47", title: "Ленинградская область"),
        .init(code: "23", title: "Краснодарский край"),
        .init(code: "16", title: "Татарстан"),
        .init(code: "66", title: "Свердловская область"),
        .init(code: "54", title: "Новосибирская область"),
        .init(code: "52", title: "Нижегородская область"),
        .init(code: "63", title: "Самарская область"),
    ]

    static func title(for code: String?) -> String? {
        code.flatMap { c in all.first { $0.code == c }?.title }
    }
}

/// Подписи аудитории закона для карточек, календаря и блока «Тебе».
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
                if tag.hasPrefix("age:"), let a = profile.age { add(a.title) }
                if tag.hasPrefix("region:"), let r = Region.title(for: profile.regionCode) { add(r) }
            }
        }
        guard !answers.isEmpty else { return "Подобрано по вашему профилю." }
        return "В профиле указано: " + answers.map { "«\($0)»" }.joined(separator: ", ") + "."
    }
}
