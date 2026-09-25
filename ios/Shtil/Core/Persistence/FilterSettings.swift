import Foundation
import SwiftData

enum SchedulePreference: String, Codable, CaseIterable, Sendable {
    case both, am, pm
}

@Model
final class FilterSettings {
    var calmMode: Bool = true
    var infoTypes: [String] = [InfoType.fact.rawValue, InfoType.official.rawValue]
    var heavyMode: String = HeavyMode.fold.rawValue
    var maxHeavy: Int = 3
    var stopTopics: [String] = [Topic.politics.rawValue, Topic.crime.rawValue]
    var hideAds: Bool = true
    var schedule: String = SchedulePreference.both.rawValue
    /// Время выпусков в минутах от полуночи: 08:00 и 19:00.
    var morningMinutes: Int = 8 * 60
    var eveningMinutes: Int = 19 * 60
    var theme: String = ThemeChoice.paper.rawValue
    var autoDusk: Bool = true

    init() {}

    var preferences: FilterPreferences {
        get {
            FilterPreferences(
                calmMode: calmMode,
                infoTypes: Set(infoTypes.compactMap(InfoType.init(rawValue:))),
                heavyMode: HeavyMode(rawValue: heavyMode) ?? .fold,
                maxHeavy: maxHeavy,
                stopTopics: Set(stopTopics.compactMap(Topic.init(rawValue:))),
                hideAds: hideAds
            )
        }
        set {
            calmMode = newValue.calmMode
            infoTypes = newValue.infoTypes.map(\.rawValue).sorted()
            heavyMode = newValue.heavyMode.rawValue
            maxHeavy = newValue.maxHeavy
            stopTopics = newValue.stopTopics.map(\.rawValue).sorted()
            hideAds = newValue.hideAds
        }
    }

    var themeChoice: ThemeChoice {
        get { ThemeChoice(rawValue: theme) ?? .paper }
        set { theme = newValue.rawValue }
    }

    var schedulePreference: SchedulePreference {
        get { SchedulePreference(rawValue: schedule) ?? .both }
        set { schedule = newValue.rawValue }
    }
}
