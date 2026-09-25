import Foundation
@testable import Shtil

enum TestData {
    static func date(_ iso: String) -> Date {
        ISO8601DateFormatter().date(from: iso)!
    }

    static func story(
        _ id: String,
        topic: Topic = .economy,
        type: InfoType = .fact,
        heaviness: Heaviness = .neutral,
        sources: Int = 2,
        updated: String = "2026-09-25T01:00:00Z",
        title: String = "Заголовок",
        summary: String = "Краткое содержание",
        meaning: String? = nil
    ) -> Story {
        Story(
            id: id, topic: topic, infoType: type, heaviness: heaviness, title: title,
            meaning: meaning, summary: summary, postCount: sources * 2, sourceCount: sources,
            sources: [], regionCode: nil, updatedAt: date(updated)
        )
    }

    static func law(
        _ id: String,
        tags: [String],
        region: String? = nil,
        effective: String? = "2026-10-01",
        status: LawStatus = .signed
    ) -> Law {
        Law(
            id: id, title: "Закон \(id)", whatChanged: "Что", whoAffected: "Кого", actions: [],
            audienceTags: tags, regionCode: region, status: status,
            dates: LawDates(introduced: nil, passed: nil, signed: nil, effective: effective.flatMap(CalendarDate.init(string:))),
            officialUrl: nil, billUrl: nil, actNumber: nil, verifiedAt: nil
        )
    }

    static func feed(_ stories: [Story], posts: Int = 38, ads: Int = 7) -> Feed {
        Feed(generatedAt: date("2026-09-25T02:55:00Z"), stories: stories, stats: FeedStats(postsTotal: posts, adsHidden: ads))
    }

    static let window = EditionWindow(start: date("2026-09-24T19:00:00Z"), end: date("2026-09-25T05:00:00Z"))
    static let today = CalendarDate(year: 2026, month: 9, day: 25)
}
