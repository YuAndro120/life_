import Foundation

/// Сборка выпуска на устройстве: stories + laws × профиль × настройки → Edition.
enum FilterEngine {
    static let charsPerMinute = 1200

    static func edition(
        number: Int,
        feed: Feed,
        laws: [Law],
        profile: UserProfile,
        preferences: FilterPreferences,
        window: EditionWindow,
        today: CalendarDate
    ) -> Edition {
        let relevantLaws = AudienceMatcher.relevantLaws(laws, profile: profile, asOf: today)

        let inWindow = feed.stories.filter { story in
            story.updatedAt <= window.end && (window.start.map { story.updatedAt > $0 } ?? true)
        }
        let allowed = inWindow.filter { story in
            !preferences.stopTopics.contains(story.topic)
                && preferences.infoTypes.contains(story.infoType)
        }

        let heavy = allowed.filter { $0.heaviness == .heavy }.sorted(by: storyOrder)
        let regular = allowed.filter { $0.heaviness != .heavy }

        var shownHeavy: [Story] = []
        var folded: [Story] = []
        switch preferences.heavyMode {
        case .hide:
            break
        case .fold:
            folded = heavy
        case .show:
            let limit = max(0, preferences.maxHeavy)
            shownHeavy = Array(heavy.prefix(limit))
            folded = Array(heavy.dropFirst(limit))
        }

        let stories = (regular + shownHeavy).sorted(by: storyOrder)
        let kept = stories.count + folded.count

        let stats = EditionStats(
            aboutYou: relevantLaws.count,
            stories: stories.count,
            readingMinutes: readingMinutes(laws: relevantLaws, stories: stories),
            postsTotal: feed.stats.postsTotal,
            adsHidden: feed.stats.adsHidden,
            filteredOut: inWindow.count - kept
        )
        return Edition(number: number, laws: relevantLaws, stories: stories, foldedHeavy: folded, stats: stats)
    }

    /// Официальное выше фактов, больше источников выше, затем свежее; `id` делает порядок детерминированным.
    static func storyOrder(_ a: Story, _ b: Story) -> Bool {
        let (ra, rb) = (rank(a.infoType), rank(b.infoType))
        if ra != rb { return ra < rb }
        if a.sourceCount != b.sourceCount { return a.sourceCount > b.sourceCount }
        if a.updatedAt != b.updatedAt { return a.updatedAt > b.updatedAt }
        return a.id < b.id
    }

    private static func rank(_ type: InfoType) -> Int {
        switch type {
        case .official: 0
        case .fact: 1
        case .opinion: 2
        case .forecast: 3
        case .rumor: 4
        }
    }

    static func readingMinutes(laws: [Law], stories: [Story]) -> Int {
        let lawChars = laws.reduce(0) { sum, l in
            sum + l.title.count + l.whatChanged.count + l.whoAffected.count + l.actions.reduce(0) { $0 + $1.count }
        }
        let storyChars = stories.reduce(0) { sum, s in
            sum + s.title.count + s.summary.count + (s.meaning?.count ?? 0)
        }
        let total = lawChars + storyChars
        guard total > 0 else { return 0 }
        return (total + charsPerMinute - 1) / charsPerMinute
    }
}
