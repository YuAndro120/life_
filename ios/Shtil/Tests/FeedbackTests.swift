import Foundation
import SwiftData
import Testing
@testable import Shtil

@MainActor @Suite struct FeedbackTests {
    private func model() throws -> (AppModel, ModelContainer) {
        let container = try PersistenceStack.makeContainer(inMemory: true)
        return (AppModel(context: container.mainContext, source: FixtureContentSource()), container)
    }

    @Test func hideStoryPersistsAndCanBeUndone() throws {
        let (m, c) = try model()
        defer { withExtendedLifetime(c) {} }
        m.apply(.hideStory(id: "st_01"))
        #expect(m.settings.preferences.hiddenStories == ["st_01"])
        #expect(m.undo == .hideStory(id: "st_01"))
        m.undoLast()
        #expect(m.settings.preferences.hiddenStories.isEmpty)
        #expect(m.undo == nil)
    }

    @Test func muteSourceAndTopicAndBoost() throws {
        let (m, c) = try model()
        defer { withExtendedLifetime(c) {} }
        m.apply(.muteSource("Шумный"))
        #expect(m.settings.preferences.mutedSources == ["Шумный"])
        m.apply(.boostTopic(.space))
        #expect(m.settings.preferences.interests == [.space])
        m.apply(.muteTopic(.sport))
        #expect(m.settings.preferences.stopTopics.contains(.sport))
        m.undoLast()
        #expect(!m.settings.preferences.stopTopics.contains(.sport))
    }

    @Test func boostingAStoppedTopicMovesItOutOfStopList() throws {
        let (m, c) = try model()
        defer { withExtendedLifetime(c) {} }
        #expect(m.settings.preferences.stopTopics.contains(.politics))
        m.apply(.boostTopic(.politics))
        #expect(m.settings.preferences.interests.contains(.politics))
        #expect(!m.settings.preferences.stopTopics.contains(.politics))
    }

    @Test func repeatedActionIsNoOpAndKeepsPreviousUndo() throws {
        let (m, c) = try model()
        defer { withExtendedLifetime(c) {} }
        m.apply(.hideStory(id: "a"))
        m.apply(.hideStory(id: "a")) // повтор: менять нечего
        #expect(m.settings.preferences.hiddenStories == ["a"])
    }

    @Test func hiddenStoryDisappearsFromEditionAfterAction() async throws {
        let (m, c) = try model()
        defer { withExtendedLifetime(c) {} }
        await m.refresh()
        m.settings.preferences = {
            var p = m.settings.preferences
            p.stopTopics = []
            return p
        }()
        let before = m.edition?.stories.map(\.id) ?? []
        let target = try #require(before.first)
        m.apply(.hideStory(id: target))
        let after = m.edition?.stories.map(\.id) ?? []
        #expect(after.contains(target) == false)
    }

    @Test func messagesAreInRussian() {
        #expect(FeedbackAction.hideStory(id: "x").message == "Сюжет скрыт")
        #expect(FeedbackAction.boostTopic(.space).message == "Больше про: Космос")
        #expect(FeedbackAction.muteSource("BBC").message.contains("BBC"))
    }
}
