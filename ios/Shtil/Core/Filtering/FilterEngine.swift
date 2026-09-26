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
            isAllowed(story, preferences)
        }

        let heavy = allowed.filter { $0.heaviness == .heavy }.sorted { storyOrder($0, $1) }
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

        let ranked = (regular + shownHeavy).sorted { storyOrder($0, $1, interests: preferences.interests) }
        let limit = max(1, preferences.storyLimit)
        let stories = Array(ranked.prefix(limit))
        let trimmed = ranked.count - stories.count
        let kept = ranked.count + folded.count
        let interestIDs = Set(stories.filter { preferences.interests.contains($0.topic) }.map(\.id))

        let stats = EditionStats(
            aboutYou: relevantLaws.count,
            stories: stories.count,
            readingMinutes: readingMinutes(laws: relevantLaws, stories: stories),
            postsTotal: feed.stats.postsTotal,
            adsHidden: feed.stats.adsHidden,
            filteredOut: inWindow.count - kept,
            trimmed: trimmed
        )
        return Edition(number: number, laws: relevantLaws, stories: stories, foldedHeavy: folded, interestIDs: interestIDs, stats: stats)
    }

    /// Можно ли показывать сюжет при таких настройках: скрытые пользователем сюжеты, источники и слова, СВО, тема, тип, страна, «только интересы».
    static func isAllowed(_ story: Story, _ p: FilterPreferences) -> Bool {
        guard !p.hiddenStories.contains(story.id) else { return false }
        if !story.sources.isEmpty, story.sources.allSatisfy({ p.mutedSources.contains($0.title) }) { return false }
        if WordFilter.matches(story, words: p.blockedWords) { return false }
        // Единственное исключение из «всё про СВО скрыто»: официальное заявление об окончании СВО показываем всегда.
        if p.hideWar, WarMarkers.isEndAnnouncement(story) { return true }
        if p.hideWar, WarMarkers.matches(story) { return false }
        guard !p.stopTopics.contains(story.topic), p.infoTypes.contains(story.infoType) else { return false }
        guard p.countries.contains(story.countryCode) else { return false }
        if p.stopTopics.contains(.politics), !p.interests.contains(story.topic), PoliticalMarkers.matches(story) { return false }
        if p.onlyInterests, !p.interests.isEmpty, !p.interests.contains(story.topic) { return false }
        return true
    }

    /// Интересные темы идут первыми; дальше официальное выше фактов, больше источников выше, затем свежее;
    /// `id` делает порядок детерминированным.
    static func storyOrder(_ a: Story, _ b: Story, interests: Set<Topic> = []) -> Bool {
        let (ia, ib) = (interests.contains(a.topic), interests.contains(b.topic))
        if ia != ib { return ia }
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
