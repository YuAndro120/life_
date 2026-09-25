import Foundation
import SwiftData
import Testing
@testable import Life

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
}
