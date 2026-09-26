import Foundation
import Testing
@testable import Shtil

@Suite struct PoliticalMarkersTests {
    private func story(_ title: String, topic: Topic = .culture, summary: String = "Краткое содержание") -> Story {
        TestData.story("s", topic: topic, title: title, summary: summary)
    }

    @Test func detectsPoliticalFiguresInAnyCase() {
        #expect(PoliticalMarkers.matches(story("Владимир Путин выступил на форуме культур")))
        #expect(PoliticalMarkers.matches(story("Встреча с Путиным прошла в Кремле")))
        #expect(PoliticalMarkers.matches(story("Песков рассказал о планах")))
        #expect(PoliticalMarkers.matches(story("Заголовок", summary: "Дональд Трамп заявил о пошлинах")))
        #expect(PoliticalMarkers.matches(story("ЗЕЛЕНСКОГО пригласили на саммит")))
    }

    @Test func detectsListsAndAuthorities() {
        #expect(PoliticalMarkers.matches(story("Минюст включил проект в реестр иноагентов")))
        #expect(PoliticalMarkers.matches(story("Депутаты приняли заявление о выборах")))
        #expect(PoliticalMarkers.matches(story("Введены новые санкции против компаний")))
        #expect(PoliticalMarkers.matches(story("МИД России вызвал посла")))
    }

    @Test func doesNotMatchUnrelatedWords() {
        #expect(!PoliticalMarkers.matches(story("Стоимость мидий выросла на рынке")))
        #expect(!PoliticalMarkers.matches(story("В парке открыли новый трамплин для прыжков")))
        #expect(!PoliticalMarkers.matches(story("Ключевая ставка сохранена на уровне 16 процентов")))
        #expect(!PoliticalMarkers.matches(story("NASA объявило состав экипажа Crew-14")))
    }

    @Test func politicsStopTopicAlsoHidesMisclassifiedStories() {
        var p = FilterPreferences.default
        p.stopTopics = [.politics]
        let misclassified = story("Владимир Путин выступил на форуме объединённых культур", topic: .culture)
        #expect(!FilterEngine.isAllowed(misclassified, p))
        p.stopTopics = []
        #expect(FilterEngine.isAllowed(misclassified, p))
    }

    @Test func doesNotApplyWhenPoliticsIsAllowedOrTopicIsInterest() {
        var p = FilterPreferences.default
        p.stopTopics = [.crime]
        #expect(FilterEngine.isAllowed(story("Путин подписал закон"), p))
        p.stopTopics = [.politics]
        p.interests = [.culture]
        #expect(FilterEngine.isAllowed(story("Путин посетил выставку", topic: .culture), p))
    }
}
