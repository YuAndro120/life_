import Foundation
import SwiftData

/// Профиль пользователя. Хранится только локально. Значения enum лежат строками,
/// чтобы схема не ломалась при добавлении вариантов.
@Model
final class Profile {
    var gender: String?
    var ageBracket: String?
    var work: [String] = []
    var housing: [String] = []
    var occupations: [String] = []
    var sells: [String] = []
    var drives: Bool?
    var regionCode: String?
    var onboardingCompleted: Bool = false

    init() {}

    var snapshot: UserProfile {
        get {
            UserProfile(
                gender: gender.flatMap(UserProfile.Gender.init(rawValue:)),
                age: ageBracket.flatMap(UserProfile.AgeBracket.init(rawValue:)),
                work: Set(work.compactMap(UserProfile.Work.init(rawValue:))),
                housing: Set(housing.compactMap(UserProfile.Housing.init(rawValue:))),
                occupations: Set(occupations.compactMap(UserProfile.Occupation.init(rawValue:))),
                sells: Set(sells.compactMap(UserProfile.Sells.init(rawValue:))),
                drives: drives,
                regionCode: regionCode
            )
        }
        set {
            gender = newValue.gender?.rawValue
            ageBracket = newValue.age?.rawValue
            work = newValue.work.map(\.rawValue).sorted()
            housing = newValue.housing.map(\.rawValue).sorted()
            occupations = newValue.occupations.map(\.rawValue).sorted()
            sells = newValue.sells.map(\.rawValue).sorted()
            drives = newValue.drives
            regionCode = newValue.regionCode
        }
    }
}
