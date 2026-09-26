import Foundation

/// Профиль из онбординга. Живёт только на устройстве и никогда не уходит на сервер.
struct UserProfile: Equatable, Codable, Sendable {
    enum Gender: String, CaseIterable, Codable, Sendable { case male, female }
    enum AgeBracket: String, CaseIterable, Codable, Sendable {
        case u20
        case a20to25 = "20_25"
        case a26to35 = "26_35"
        case a36to50 = "36_50"
        case a50plus = "50p"
    }
    enum Work: String, CaseIterable, Codable, Sendable { case employee, ip, selfemployed, student }
    enum Housing: String, CaseIterable, Codable, Sendable { case renter, owner, mortgage }
    /// Сфера работы: по ней подбираются законы для отрасли (теги `industry:*`).
    enum Occupation: String, CaseIterable, Codable, Sendable {
        case it, trade, food, education, health, construction, transport, industry, agriculture, finance, publicService, beauty, creative
    }
    /// Что продаёт ИП или самозанятый: товары и услуги, к которым есть особые правила (теги `sells:*`).
    enum Sells: String, CaseIterable, Codable, Sendable {
        case goods, marked, alcohol, food, online, services, transport, rent
    }

    var gender: Gender?
    var age: AgeBracket?
    var work: Set<Work>
    var housing: Set<Housing>
    var occupations: Set<Occupation>
    var sells: Set<Sells>
    /// `true` — «Вожу авто», `false` — «Не вожу», `nil` — не отвечал.
    var drives: Bool?
    var regionCode: String?

    init(
        gender: Gender? = nil,
        age: AgeBracket? = nil,
        work: Set<Work> = [],
        housing: Set<Housing> = [],
        occupations: Set<Occupation> = [],
        sells: Set<Sells> = [],
        drives: Bool? = nil,
        regionCode: String? = nil
    ) {
        self.gender = gender
        self.age = age
        self.work = work
        self.housing = housing
        self.occupations = occupations
        self.sells = sells
        self.drives = drives
        self.regionCode = regionCode
    }

    static let empty = UserProfile()
}
