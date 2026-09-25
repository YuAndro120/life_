import Testing
@testable import Shtil

@Suite struct BuildingPlanTests {
    private func plan(prefs: FilterPreferences = .default, profile: UserProfile = UserProfile(work: [.ip])) throws -> BuildingPlan {
        let feed = try Fixtures.feed()
        let e = FilterEngine.edition(
            number: 1, feed: feed, laws: try Fixtures.laws(), profile: profile, preferences: prefs,
            window: EditionWindow(start: nil, end: TestData.date("2026-09-25T05:00:00Z")), today: TestData.today
        )
        return BuildingPlan.make(edition: e, feed: feed)
    }

    @Test func numbersComeFromFeedAndEdition() throws {
        let p = try plan()
        #expect(p.totalPosts == 38)
        #expect(p.storyCount == 4)
        #expect(p.lawCount == 2)
        #expect(p.keptPosts <= p.totalPosts)
        #expect(p.hiddenPosts == p.totalPosts - p.keptPosts)
        #expect(p.sourceCount == 14)
        #expect(p.cards.count == 3)
    }

    @Test func slipsSplitIntoKeptAndFiltered() throws {
        let p = try plan()
        let filtered = (0..<p.slipCount).filter(p.isFiltered(slip:)).count
        #expect(p.slipCount == 38)
        #expect(filtered == p.slipCount - p.keptSlipCount)
    }

    @Test func slipStrideWorksForAnyCount() {
        for n in [1, 2, 17, 34, 38] {
            let p = BuildingPlan(totalPosts: n, sourceCount: 1, keptPosts: n / 2, hiddenPosts: n - n / 2, storyCount: 1, lawCount: 0, cards: [])
            let filtered = (0..<p.slipCount).filter(p.isFiltered(slip:)).count
            #expect(filtered == p.slipCount - p.keptSlipCount, "n=\(n)")
        }
    }

    @Test func emptyFeedDoesNotCrash() {
        let p = BuildingPlan(totalPosts: 0, sourceCount: 0, keptPosts: 0, hiddenPosts: 0, storyCount: 0, lawCount: 0, cards: [])
        #expect(p.slipCount == 0 && p.keptSlipCount == 0)
        #expect(p.stepTexts[1] == "Лишнего не нашлось")
        #expect(p.stepTexts[3] == "Изменений в законах для тебя пока нет")
    }

    @Test func clustersAreWithinRangeAndCoverAllCards() throws {
        let p = try plan()
        let clusters = (0..<p.keptSlipCount).map(p.cluster(forKept:))
        #expect(clusters.allSatisfy { (0..<p.cards.count).contains($0) })
        #expect(clusters == clusters.sorted())
    }

    @Test func stepTextsUsePlurals() throws {
        let p = try plan()
        #expect(p.stepTexts[0] == "Собрали 38 постов из 14 источников")
        #expect(p.stepTexts[2] == "Склеили повторы в 4 сюжета")
        #expect(p.stepTexts[3] == "Нашли 2 изменения в законах для тебя")
    }
}
