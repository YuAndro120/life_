import Foundation
import SwiftData
import Testing
@testable import Shtil

@MainActor @Suite struct ProfileBackupTests {
    private func model(store: any BackupStoring, completed: Bool = false) throws -> (AppModel, ModelContainer) {
        let container = try PersistenceStack.makeContainer(inMemory: true)
        let m = AppModel(context: container.mainContext, source: FixtureContentSource(), backup: store)
        if completed { m.profile.onboardingCompleted = true }
        return (m, container)
    }

    @Test func backupIsWrittenAfterOnboardingAndRestoredOnFreshInstall() async throws {
        let store = InMemoryBackupStore()
        let (m, c1) = try model(store: store, completed: true)
        defer { withExtendedLifetime(c1) {} }
        var p = m.profile.snapshot
        p.regionCode = "16"
        p.work = [.ip]
        p.occupations = [.trade]
        m.profile.snapshot = p
        var prefs = m.settings.preferences
        prefs.interests = [.space]
        prefs.blockedWords = ["сплетн"]
        prefs.hideWar = false
        m.settings.preferences = prefs
        m.settings.theme = ThemeChoice.sage.rawValue
        m.save()
        #expect(store.read() != nil)

        // «Переустановка»: новая база, тот же Keychain.
        let (fresh, c2) = try model(store: store)
        defer { withExtendedLifetime(c2) {} }
        #expect(fresh.pendingBackup != nil)
        #expect(await fresh.restoreBackup())
        #expect(fresh.profile.onboardingCompleted)
        #expect(fresh.profile.regionCode == "16")
        #expect(fresh.profile.snapshot.occupations == [.trade])
        #expect(fresh.settings.preferences.interests == [.space])
        #expect(fresh.settings.preferences.blockedWords == ["сплетн"])
        #expect(fresh.settings.preferences.hideWar == false)
        #expect(fresh.settings.theme == ThemeChoice.sage.rawValue)
        #expect(fresh.pendingBackup == nil)
    }

    @Test func emptyProfileDoesNotOverwriteAnExistingBackup() throws {
        let store = InMemoryBackupStore(Data("старая копия".utf8))
        let (m, c) = try model(store: store)
        defer { withExtendedLifetime(c) {} }
        m.save() // онбординг ещё не завершён
        #expect(store.read() == Data("старая копия".utf8))
    }

    @Test func hiddenStoriesAreNotBackedUp() throws {
        let store = InMemoryBackupStore()
        let (m, c) = try model(store: store, completed: true)
        defer { withExtendedLifetime(c) {} }
        m.apply(.hideStory(id: "st_01"))
        let backup = try #require(store.read().flatMap(ProfileBackup.decode))
        #expect(backup.preferences.hiddenStories.isEmpty)
    }

    @Test func disablingBackupDeletesItAndStopsWriting() throws {
        let store = InMemoryBackupStore()
        let (m, c) = try model(store: store, completed: true)
        defer { withExtendedLifetime(c) {} }
        m.save()
        #expect(store.read() != nil)
        m.setBackup(enabled: false)
        #expect(store.read() == nil)
        m.save()
        #expect(store.read() == nil)
        m.setBackup(enabled: true)
        #expect(store.read() != nil)
    }

    @Test func oldBackupWithoutNewFieldsStillDecodes() throws {
        let json = #"{"version":1,"savedAt":"2026-09-26T10:00:00Z","profile":{"work":["ip"]},"preferences":{"calmMode":true},"theme":"paper","autoDusk":true,"schedule":"both","morningMinutes":480,"eveningMinutes":1140}"#
        let backup = try #require(ProfileBackup.decode(Data(json.utf8)))
        #expect(backup.profile.work == [.ip])
        #expect(backup.profile.occupations.isEmpty)
        #expect(backup.preferences.hideWar)
        #expect(backup.preferences.countries == ["RU"])
    }

    @Test func backupFromNewerAppVersionIsIgnored() {
        let json = #"{"version":99,"savedAt":"2026-09-26T10:00:00Z","profile":{},"preferences":{},"theme":"paper","autoDusk":true,"schedule":"both","morningMinutes":480,"eveningMinutes":1140}"#
        #expect(ProfileBackup.decode(Data(json.utf8)) == nil)
    }

    @Test func keychainRoundTrip() {
        let store = KeychainBackupStore()
        let saved = store.read() // не затираем чужие данные при локальном запуске тестов
        defer { if let saved { store.write(saved) } else { store.delete() } }
        store.delete()
        #expect(store.read() == nil)
        #expect(store.write(Data("v1".utf8)))
        #expect(store.read() == Data("v1".utf8))
        #expect(store.write(Data("v2".utf8)))
        #expect(store.read() == Data("v2".utf8))
        store.delete()
        #expect(store.read() == nil)
    }
}
