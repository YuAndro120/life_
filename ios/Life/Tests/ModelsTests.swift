import Foundation
import Testing
@testable import Life

@Suite struct ModelsTests {
    @Test func fixturesDecode() throws {
        let feed = try Fixtures.feed()
        let laws = try Fixtures.laws()
        #expect(feed.stories.count == 9)
        #expect(feed.stats.postsTotal == 38)
        #expect(feed.stats.adsHidden == 7)
        #expect(laws.count == 6)
        #expect(laws[0].dates.effective == CalendarDate(year: 2026, month: 10, day: 1))
        #expect(laws[0].status == .signed)
    }

    @Test func unknownTopicSkipsOnlyThatStory() throws {
        let json = """
        {"generated_at":"2026-09-25T02:55:00Z","stats":{"posts_total":1,"ads_hidden":0},
         "stories":[
          {"id":"a","topic":"weather","info_type":"fact","heaviness":"neutral","title":"t","meaning":null,
           "summary":"s","post_count":1,"source_count":1,"sources":[],"region_code":null,"updated_at":"2026-09-25T01:10:00Z"},
          {"id":"b","topic":"economy","info_type":"fact","heaviness":"neutral","title":"t","meaning":null,
           "summary":"s","post_count":1,"source_count":1,"sources":[],"region_code":null,"updated_at":"2026-09-25T01:10:00Z"}]}
        """
        let feed = try APICoding.decoder().decode(Feed.self, from: Data(json.utf8))
        #expect(feed.stories.map(\.id) == ["b"])
    }

    @Test func fractionalSecondsDateDecodes() throws {
        let json = #"{"generated_at":"2026-09-25T02:55:00.123Z","stories":[],"stats":{"posts_total":0,"ads_hidden":0}}"#
        _ = try APICoding.decoder().decode(Feed.self, from: Data(json.utf8))
    }

    @Test func calendarDateParsingAndMath() {
        #expect(CalendarDate(string: "2026-10-01") == CalendarDate(year: 2026, month: 10, day: 1))
        #expect(CalendarDate(string: "2026-10-01T10:00:00Z") == CalendarDate(year: 2026, month: 10, day: 1))
        #expect(CalendarDate(string: "…") == nil)
        #expect(CalendarDate(string: "2026-13-01") == nil)
        let today = CalendarDate(year: 2026, month: 9, day: 25)
        #expect(CalendarDate(year: 2026, month: 10, day: 1).days(from: today) == 6)
        #expect(CalendarDate(year: 2027, month: 3, day: 1).days(from: today) == 157)
        #expect(today.days(from: CalendarDate(year: 2026, month: 10, day: 1)) == -6)
        #expect(CalendarDate(year: 2026, month: 10, day: 1).adding(days: -1) == CalendarDate(year: 2026, month: 9, day: 30))
        #expect(today < CalendarDate(year: 2026, month: 10, day: 1))
    }
}
