import Testing
@testable import Shtil

@Suite struct FilterEngineTests {
    private func build(
        _ stories: [Story],
        laws: [Law] = [],
        profile: UserProfile = .empty,
        prefs: FilterPreferences = .default,
        window: EditionWindow = TestData.window
    ) -> Edition {
        FilterEngine.edition(
            number: 268, feed: TestData.feed(stories), laws: laws, profile: profile,
            preferences: prefs, window: window, today: TestData.today
        )
    }

    private func prefs(
        stop: Set<Topic> = [], types: Set<InfoType> = Set(InfoType.allCases),
        heavy: HeavyMode = .show, maxHeavy: Int = 3
    ) -> FilterPreferences {
        FilterPreferences(calmMode: true, infoTypes: types, heavyMode: heavy, maxHeavy: maxHeavy, stopTopics: stop, hideAds: true)
    }

    // MARK: стоп-темы и типы

    @Test func stopTopicsAreRemoved() {
        let e = build(
            [TestData.story("a", topic: .politics), TestData.story("b", topic: .economy)],
            prefs: prefs(stop: [.politics])
        )
        #expect(e.stories.map(\.id) == ["b"])
        #expect(e.stats.filteredOut == 1)
    }

    @Test func disabledInfoTypesAreRemoved() {
        let e = build(
            [TestData.story("f", type: .fact), TestData.story("r", type: .rumor), TestData.story("p", type: .forecast)],
            prefs: prefs(types: [.fact, .official])
        )
        #expect(e.stories.map(\.id) == ["f"])
        #expect(e.stats.filteredOut == 2)
    }

    @Test func stopTopicRemovesHeavyStoryEvenInShowMode() {
        let e = build(
            [TestData.story("h", topic: .crime, heaviness: .heavy)],
            prefs: prefs(stop: [.crime], heavy: .show)
        )
        #expect(e.stories.isEmpty)
        #expect(e.foldedHeavy.isEmpty)
    }

    // MARK: тяжёлые темы

    @Test func heavyHideDropsThemEntirely() {
        let e = build(
            [TestData.story("n"), TestData.story("h", heaviness: .heavy)],
            prefs: prefs(heavy: .hide)
        )
        #expect(e.stories.map(\.id) == ["n"])
        #expect(e.foldedHeavy.isEmpty)
        #expect(e.stats.filteredOut == 1)
    }

    @Test func heavyFoldMovesThemToSummary() {
        let e = build(
            [TestData.story("n"), TestData.story("h1", heaviness: .heavy), TestData.story("h2", heaviness: .heavy)],
            prefs: prefs(heavy: .fold)
        )
        #expect(e.stories.map(\.id) == ["n"])
        #expect(Set(e.foldedHeavy.map(\.id)) == ["h1", "h2"])
        #expect(e.stats.filteredOut == 0)
    }

    @Test func heavyShowRespectsMaxAndFoldsTheRest() {
        let heavy = (1...4).map { TestData.story("h\($0)", heaviness: .heavy, sources: 10 - $0) }
        let e = build(heavy, prefs: prefs(heavy: .show, maxHeavy: 2))
        #expect(e.stories.map(\.id) == ["h1", "h2"])
        #expect(e.foldedHeavy.map(\.id) == ["h3", "h4"])
    }

    @Test func tenseIsNotHeavy() {
        let e = build([TestData.story("t", heaviness: .tense)], prefs: prefs(heavy: .hide))
        #expect(e.stories.map(\.id) == ["t"])
    }

    @Test func negativeMaxHeavyBehavesLikeZero() {
        let e = build([TestData.story("h", heaviness: .heavy)], prefs: prefs(heavy: .show, maxHeavy: -1))
        #expect(e.stories.isEmpty)
        #expect(e.foldedHeavy.count == 1)
    }

    // MARK: порядок

    @Test func officialBeforeFactThenMoreSourcesThenFresher() {
        let e = build([
            TestData.story("fact-many", type: .fact, sources: 9),
            TestData.story("official-few", type: .official, sources: 2),
            TestData.story("fact-few-new", type: .fact, sources: 3, updated: "2026-09-25T03:00:00Z"),
            TestData.story("fact-few-old", type: .fact, sources: 3, updated: "2026-09-25T01:00:00Z"),
        ])
        #expect(e.stories.map(\.id) == ["official-few", "fact-many", "fact-few-new", "fact-few-old"])
    }

    @Test func orderIsDeterministicForTies() {
        let a = TestData.story("b"), b = TestData.story("a")
        #expect(build([a, b]).stories.map(\.id) == ["a", "b"])
        #expect(build([b, a]).stories.map(\.id) == ["a", "b"])
    }

    // MARK: окно выпуска

    @Test func windowIsExclusiveStartInclusiveEnd() {
        let e = build(
            [
                TestData.story("at-start", updated: "2026-09-24T19:00:00Z"),
                TestData.story("inside", updated: "2026-09-25T01:00:00Z"),
                TestData.story("at-end", updated: "2026-09-25T05:00:00Z"),
                TestData.story("after", updated: "2026-09-25T05:00:01Z"),
            ],
            prefs: prefs()
        )
        #expect(Set(e.stories.map(\.id)) == ["inside", "at-end"])
    }

    @Test func openStartWindowIncludesEverythingBeforeEnd() {
        let e = build(
            [TestData.story("old", updated: "2026-01-01T00:00:00Z")],
            prefs: prefs(),
            window: EditionWindow(start: nil, end: TestData.window.end)
        )
        #expect(e.stories.count == 1)
    }

    // MARK: статистика

    @Test func statsForMockupScenario() {
        let e = build(
            [TestData.story("a", sources: 5), TestData.story("b", sources: 4), TestData.story("c", sources: 3)],
            laws: [TestData.law("l1", tags: ["all"]), TestData.law("l2", tags: ["all"])]
        )
        #expect(e.stats.aboutYou == 2)
        #expect(e.stats.stories == 3)
        #expect(e.stats.postsTotal == 38)
        #expect(e.stats.adsHidden == 7)
        #expect(e.number == 268)
    }

    @Test func readingMinutesRoundUpAt1200CharsPerMinute() {
        func story(chars: Int) -> Story {
            TestData.story("s", title: "", summary: String(repeating: "я", count: chars))
        }
        #expect(FilterEngine.readingMinutes(laws: [], stories: []) == 0)
        #expect(FilterEngine.readingMinutes(laws: [], stories: [story(chars: 1)]) == 1)
        #expect(FilterEngine.readingMinutes(laws: [], stories: [story(chars: 1200)]) == 1)
        #expect(FilterEngine.readingMinutes(laws: [], stories: [story(chars: 1201)]) == 2)
    }

    @Test func emptyFeedGivesEmptyEdition() {
        let e = build([])
        #expect(e.stories.isEmpty && e.laws.isEmpty && e.foldedHeavy.isEmpty)
        #expect(e.stats.readingMinutes == 0)
    }

    @Test func fixtureFeedWithDefaultPreferences() throws {
        let feed = try Fixtures.feed()
        let e = FilterEngine.edition(
            number: 1, feed: feed, laws: try Fixtures.laws(),
            profile: UserProfile(gender: .male, age: .a20to25, work: [.ip], housing: [.renter], drives: true),
            preferences: .default, window: EditionWindow(start: nil, end: TestData.date("2026-09-25T05:00:00Z")),
            today: TestData.today
        )
        // По умолчанию: факты и решения, политика и криминал скрыты, тяжёлые свёрнуты.
        #expect(Set(e.stories.map(\.id)) == ["st_01", "st_02", "st_03", "st_09"])
        #expect(Set(e.foldedHeavy.map(\.id)) == ["st_06", "st_07"])
        #expect(e.stories.first?.id == "st_01")
        #expect(e.stats.aboutYou == 4)
    }
}
