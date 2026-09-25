import Foundation
import SwiftData

/// Последний ответ API в SwiftData: выпуск открывается офлайн (критерий готовности MVP).
@MainActor
struct ContentCache {
    struct Snapshot {
        var feed: Feed
        var laws: [Law]
        var fetchedAt: Date
    }

    let context: ModelContext

    func save(feed: Feed, laws: [Law], fetchedAt: Date) {
        let encoder = APICoding.encoder()
        do {
            try context.delete(model: CachedStory.self)
            try context.delete(model: CachedLaw.self)
            try context.delete(model: CachedMeta.self)
            for story in feed.stories {
                context.insert(CachedStory(id: story.id, updatedAt: story.updatedAt, payload: try encoder.encode(story)))
            }
            for law in laws {
                context.insert(CachedLaw(id: law.id, payload: try encoder.encode(law)))
            }
            context.insert(CachedMeta(
                feedGeneratedAt: feed.generatedAt,
                postsTotal: feed.stats.postsTotal,
                adsHidden: feed.stats.adsHidden,
                fetchedAt: fetchedAt
            ))
            try context.save()
        } catch {
            context.rollback()
        }
    }

    func load() -> Snapshot? {
        guard let meta = try? context.fetch(FetchDescriptor<CachedMeta>()).first else { return nil }
        let decoder = APICoding.decoder()
        let stories = ((try? context.fetch(FetchDescriptor<CachedStory>(sortBy: [SortDescriptor(\.updatedAt, order: .reverse)]))) ?? [])
            .compactMap { try? decoder.decode(Story.self, from: $0.payload) }
        let laws = ((try? context.fetch(FetchDescriptor<CachedLaw>())) ?? [])
            .compactMap { try? decoder.decode(Law.self, from: $0.payload) }
        let feed = Feed(
            generatedAt: meta.feedGeneratedAt,
            stories: stories,
            stats: FeedStats(postsTotal: meta.postsTotal, adsHidden: meta.adsHidden)
        )
        return Snapshot(feed: feed, laws: laws, fetchedAt: meta.fetchedAt)
    }
}
