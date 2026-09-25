import Foundation

/// Профиль из онбординга. Живёт только на устройстве и никогда не уходит на сервер.
struct UserProfile: Equatable, Sendable {
    enum Gender: String, CaseIterable, Sendable { case male, female }
    enum AgeBracket: String, CaseIterable, Sendable {
        case u20
        case a20to25 = "20_25"
        case a26to35 = "26_35"
        case a36to50 = "36_50"
        case a50plus = "50p"
    }
    enum Work: String, CaseIterable, Sendable { case employee, ip, selfemployed, student }
    enum Housing: String, CaseIterable, Sendable { case renter, owner, mortgage }

    var gender: Gender?
    var age: AgeBracket?
    var work: Set<Work>
    var housing: Set<Housing>
    /// `true` — «Вожу авто», `false` — «Не вожу», `nil` — не отвечал.
    var drives: Bool?
    var regionCode: String?

    init(
        gender: Gender? = nil,
        age: AgeBracket? = nil,
        work: Set<Work> = [],
        housing: Set<Housing> = [],
        drives: Bool? = nil,
        regionCode: String? = nil
    ) {
        self.gender = gender
        self.age = age
        self.work = work
        self.housing = housing
        self.drives = drives
        self.regionCode = regionCode
    }

    static let empty = UserProfile()
}
