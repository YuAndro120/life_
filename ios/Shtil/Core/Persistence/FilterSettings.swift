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
    var countries: [String] = ["RU"]
    var interests: [String] = []
    var onlyInterests: Bool = false
    var mutedSources: [String] = []
    /// Идентификаторы скрытых сюжетов в порядке скрытия (хранятся последние 300).
    var hiddenStories: [String] = []
    var storyLimit: Int = 15
    var hideWar: Bool = true
    var blockedWords: [String] = []

    static let hiddenLimit = 300

    init() {}

    var preferences: FilterPreferences {
        get {
            FilterPreferences(
                calmMode: calmMode,
                infoTypes: Set(infoTypes.compactMap(InfoType.init(rawValue:))),
                heavyMode: HeavyMode(rawValue: heavyMode) ?? .fold,
                maxHeavy: maxHeavy,
                stopTopics: Set(stopTopics.compactMap(Topic.init(rawValue:))),
                hideAds: hideAds,
                countries: Set(countries),
                interests: Set(interests.compactMap(Topic.init(rawValue:))),
                onlyInterests: onlyInterests,
                mutedSources: Set(mutedSources),
                hiddenStories: Set(hiddenStories),
                hideWar: hideWar,
                blockedWords: blockedWords,
                storyLimit: storyLimit
            )
        }
        set {
            calmMode = newValue.calmMode
            infoTypes = newValue.infoTypes.map(\.rawValue).sorted()
            heavyMode = newValue.heavyMode.rawValue
            maxHeavy = newValue.maxHeavy
            stopTopics = newValue.stopTopics.map(\.rawValue).sorted()
            hideAds = newValue.hideAds
            countries = newValue.countries.sorted()
            interests = newValue.interests.map(\.rawValue).sorted()
            onlyInterests = newValue.onlyInterests
            mutedSources = newValue.mutedSources.sorted()
            // Порядок скрытия сохраняем: новые id добавляются в конец, лишние старые отбрасываются.
            let kept = hiddenStories.filter { newValue.hiddenStories.contains($0) }
            let added = newValue.hiddenStories.subtracting(kept).sorted()
            hiddenStories = Array((kept + added).suffix(Self.hiddenLimit))
            storyLimit = newValue.storyLimit
            hideWar = newValue.hideWar
            blockedWords = newValue.blockedWords
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
