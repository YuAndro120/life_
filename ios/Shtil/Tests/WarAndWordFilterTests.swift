import Foundation
import Testing
@testable import Shtil

@Suite struct WarAndWordFilterTests {
    private func story(_ title: String, topic: Topic = .incidents, type: InfoType = .fact, summary: String = "Краткое содержание") -> Story {
        TestData.story("w", topic: topic, type: type, title: title, summary: summary)
    }

    private var prefs: FilterPreferences {
        var p = FilterPreferences.default
        p.infoTypes = Set(InfoType.allCases)
        p.stopTopics = []
        return p
    }

    @Test func warIsHiddenByDefault() {
        #expect(FilterPreferences.default.hideWar)
        #expect(!FilterEngine.isAllowed(story("Вооружённые силы России нанесли удары по объектам инфраструктуры Украины"), prefs))
        #expect(!FilterEngine.isAllowed(story("Участник СВО получил награду", topic: .culture), prefs))
        #expect(!FilterEngine.isAllowed(story("Над Белгородом сбили беспилотники"), prefs))
    }

    @Test func warFilterCanBeSwitchedOff() {
        var p = prefs
        p.hideWar = false
        #expect(FilterEngine.isAllowed(story("Участник СВО получил награду", topic: .culture), p))
    }

    @Test func officialEndOfWarIsTheOnlyException() {
        var p = prefs
        p.stopTopics = [.politics]
        let end = story("Кремль официально объявил о завершении специальной военной операции", topic: .politics, type: .official)
        #expect(FilterEngine.isAllowed(end, p))
        let rumor = story("Источники сообщили о скором окончании СВО", topic: .politics, type: .rumor)
        #expect(!FilterEngine.isAllowed(rumor, p))
        let unrelated = story("Завершено строительство моста", type: .official)
        #expect(FilterEngine.isAllowed(unrelated, p))
    }

    @Test func ordinaryNewsAreNotTouchedByWarFilter() {
        #expect(FilterEngine.isAllowed(story("Ключевая ставка сохранена на уровне 16 процентов", topic: .finance), prefs))
        #expect(FilterEngine.isAllowed(story("Свой первый матч сыграл футбольный клуб", topic: .sport), prefs))
    }

    @Test func blockedWordsMatchByWordStartAndPhrase() {
        var p = prefs
        p.hideWar = false
        p.blockedWords = ["футбол"]
        #expect(!FilterEngine.isAllowed(story("Футболист перешёл в новый клуб", topic: .sport), p))
        #expect(!FilterEngine.isAllowed(story("Матч", topic: .sport, summary: "Прошёл футбольный матч"), p))
        #expect(FilterEngine.isAllowed(story("Хоккей: матч закончился вничью", topic: .sport), p))
        p.blockedWords = ["ключевая ставка"]
        #expect(!FilterEngine.isAllowed(story("Ключевая ставка снижена", topic: .finance), p))
        #expect(FilterEngine.isAllowed(story("Ставка по вкладам выросла", topic: .finance), p))
    }

    @Test func wordNormalization() {
        #expect(WordFilter.normalize("  Ёлка  ") == "елка")
        #expect(WordFilter.normalize("а") == "")
        #expect(WordFilter.normalize("   ") == "")
        #expect(WordFilter.normalize("Ключевая   Ставка") == "ключевая ставка")
        #expect(WordFilter.normalize(String(repeating: "я", count: 60)) == "")
    }

    @Test func settingsPersistNewFields() throws {
        let s = FilterSettings()
        var p = s.preferences
        #expect(p.hideWar)
        p.hideWar = false
        p.blockedWords = ["футбол", "погода"]
        s.preferences = p
        #expect(s.preferences.hideWar == false)
        #expect(s.preferences.blockedWords == ["футбол", "погода"])
    }

    @Test func otherRegionsAreHiddenByDefault() {
        func regional(_ code: String?) -> Story {
            let base = TestData.story("r", topic: .city, title: "Новость города", summary: "Что-то произошло")
            return Story(
                id: base.id, topic: base.topic, infoType: base.infoType, heaviness: base.heaviness, title: base.title, meaning: nil,
                summary: base.summary, postCount: base.postCount, sourceCount: base.sourceCount, sources: [],
                regionCode: code, country: nil, lang: "ru", updatedAt: base.updatedAt
            )
        }
        var p = prefs
        p.hideWar = false
        #expect(FilterPreferences.default.hideOtherRegions)
        p.homeRegion = "16"
        #expect(FilterEngine.isAllowed(regional(nil), p))
        #expect(FilterEngine.isAllowed(regional("16"), p))
        #expect(!FilterEngine.isAllowed(regional("78"), p))
        p.homeRegion = nil
        #expect(!FilterEngine.isAllowed(regional("16"), p))
        p.hideOtherRegions = false
        #expect(FilterEngine.isAllowed(regional("78"), p))
    }

    private func law(_ id: String, title: String, what: String = "Что меняется", who: String = "Кого касается") -> Law {
        Law(
            id: id, title: title, whatChanged: what, whoAffected: who, actions: [], audienceTags: ["all"], regionCode: nil,
            status: .signed, dates: LawDates(introduced: nil, passed: nil, signed: nil, effective: CalendarDate(string: "2026-10-01")),
            officialUrl: nil, billUrl: nil, actNumber: nil, verifiedAt: nil
        )
    }

    @Test func lawsAboutWarParticipantsFollowTheWarSwitch() {
        #expect(WarMarkers.matches(law("a", title: "Сохранение права на жильё по соцнайму семьям погибших участников СВО")))
        #expect(WarMarkers.matches(law("b", title: "Уточнён перечень лиц, относимых к ветеранам и инвалидам боевых действий")))
        #expect(WarMarkers.matches(law("c", title: "Льготы детям", who: "Дети погибших военнослужащих")))
        #expect(!WarMarkers.matches(law("d", title: "Меняется срок уведомлений об авансовых платежах")))
        let feed = TestData.feed([])
        let laws = [law("a", title: "Жильё семьям погибших участников СВО"), law("d", title: "Авансовые платежи для ИП")]
        func shown(hideWar: Bool) -> Set<String> {
            var p = prefs
            p.hideWar = hideWar
            let e = FilterEngine.edition(
                number: 1, feed: feed, laws: laws, profile: UserProfile(), preferences: p,
                window: EditionWindow(start: nil, end: TestData.date("2026-09-25T05:00:00Z")), today: TestData.today
            )
            return Set(e.laws.map(\.id))
        }
        #expect(shown(hideWar: true) == ["d"])
        #expect(shown(hideWar: false) == ["a", "d"])
    }
}
