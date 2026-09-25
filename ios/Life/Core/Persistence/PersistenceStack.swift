import Foundation
import SwiftData

enum PersistenceStack {
    static let schema = Schema([
        Profile.self, FilterSettings.self, CachedStory.self, CachedLaw.self,
        Reminder.self, EditionCounter.self,
    ])

    static func makeContainer(inMemory: Bool = false) throws -> ModelContainer {
        let config = ModelConfiguration(schema: schema, isStoredInMemoryOnly: inMemory)
        return try ModelContainer(for: schema, configurations: [config])
    }
}
