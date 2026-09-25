import Foundation
import SwiftData
import Testing
@testable import Shtil

@Suite struct PersistenceTests {
    @Test func profileRoundTripsThroughStorage() throws {
        let container = try PersistenceStack.makeContainer(inMemory: true)
        let context = ModelContext(container)
        let profile = Profile()
        let value = UserProfile(gender: .female, age: .a36to50, work: [.employee, .student], housing: [.mortgage], drives: false, regionCode: "77")
        profile.snapshot = value
        context.insert(profile)
        try context.save()

        let fetched = try #require(try ModelContext(container).fetch(FetchDescriptor<Profile>()).first)
        #expect(fetched.snapshot == value)
    }

    @Test func filterSettingsDefaultsMatchMockup() throws {
        let container = try PersistenceStack.makeContainer(inMemory: true)
        let context = ModelContext(container)
        let settings = FilterSettings()
        context.insert(settings)
        try context.save()

        #expect(settings.preferences == .default)
        #expect(settings.themeChoice == .paper)
        #expect(settings.autoDusk)
        #expect(settings.schedulePreference == .both)
        #expect(settings.morningMinutes == 480)
        #expect(settings.eveningMinutes == 1140)
    }

    @Test func preferencesWriteBackAndUnknownRawValuesAreIgnored() {
        let settings = FilterSettings()
        var prefs = settings.preferences
        prefs.stopTopics = [.sport, .crypto]
        prefs.heavyMode = .hide
        settings.preferences = prefs
        #expect(settings.preferences == prefs)

        settings.stopTopics = ["sport", "removed_topic"]
        #expect(settings.preferences.stopTopics == [.sport])
    }

    @Test func cachedLawIdIsUnique() throws {
        let container = try PersistenceStack.makeContainer(inMemory: true)
        let context = ModelContext(container)
        context.insert(CachedLaw(id: "lw_01", payload: Data("v1".utf8)))
        context.insert(CachedLaw(id: "lw_01", payload: Data("v2".utf8)))
        try context.save()
        #expect(try context.fetch(FetchDescriptor<CachedLaw>()).count == 1)
    }

    @Test func interestsAndMutesRoundTrip() {
        let settings = FilterSettings()
        var p = settings.preferences
        p.countries = ["RU", "US", "GB"]
        p.interests = [.space, .science]
        p.onlyInterests = true
        p.mutedSources = ["Шумный", "Другой"]
        p.hiddenStories = ["st_1", "st_2"]
        p.storyLimit = 8
        settings.preferences = p
        #expect(settings.preferences == p)
        #expect(settings.preferences.storyLimit == 8)
    }

    @Test func hiddenStoriesKeepOrderAndAreCapped() {
        let settings = FilterSettings()
        var p = settings.preferences
        p.hiddenStories = Set((1...(FilterSettings.hiddenLimit + 20)).map { "st_\($0)" })
        settings.preferences = p
        #expect(settings.hiddenStories.count == FilterSettings.hiddenLimit)
        var q = settings.preferences
        q.hiddenStories.insert("st_new")
        settings.preferences = q
        #expect(settings.hiddenStories.last == "st_new", "новое скрытие добавляется в конец")
        #expect(settings.hiddenStories.count == FilterSettings.hiddenLimit)
    }

    @Test func defaultsForNewFieldsAreConservative() {
        let s = FilterSettings()
        #expect(s.preferences.countries == ["RU"])
        #expect(s.preferences.interests.isEmpty && !s.preferences.onlyInterests)
        #expect(s.preferences.storyLimit == 15)
    }
}
