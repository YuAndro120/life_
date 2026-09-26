import Foundation
import Testing
@testable import Shtil

@Suite struct AboutParserTests {
    @Test func parsesTheExampleFromTheOnboarding() {
        let r = AboutParser.parse("Живу в Казани, ИП на патенте, езжу на машине, увлекаюсь космосом и футболом")
        #expect(r.regionCode == "16")
        #expect(r.work == [.ip])
        #expect(r.drives == true)
        #expect(r.interests == [.space, .sport])
    }

    @Test func regionsAreFoundInAnyCaseForm() {
        #expect(AboutParser.parse("Я из Москвы").regionCode == "77")
        #expect(AboutParser.parse("живу в Санкт-Петербурге").regionCode == "78")
        #expect(AboutParser.parse("живу в питере").regionCode == "78")
        #expect(AboutParser.parse("Екатеринбург").regionCode == "66")
        #expect(AboutParser.parse("живу в Нижнем Новгороде").regionCode == "52")
        #expect(AboutParser.parse("живу в Новом Уренгое").regionCode == "89")
        #expect(AboutParser.parse("Красноярский край").regionCode == "24")
    }

    @Test func liveCueWinsOverWorkPlace() {
        #expect(AboutParser.parse("Работаю в Москве, но живу в Туле").regionCode == "71")
    }

    @Test func doesNotInventRegionsFromOrdinaryWords() {
        #expect(AboutParser.parse("Я читал книгу про Казанову").regionCode == nil)
        #expect(AboutParser.parse("Ответ тверже камня").regionCode == nil)
        #expect(AboutParser.parse("Люблю читать").regionCode == nil)
        #expect(AboutParser.parse("").isEmpty)
    }

    @Test func workAndHousing() {
        let r = AboutParser.parse("Студент, снимаю квартиру, самозанятый")
        #expect(r.work == [.student, .selfemployed])
        #expect(r.housing == [.renter])
        let m = AboutParser.parse("Работаю по найму, есть ипотека")
        #expect(m.work == [.employee])
        #expect(m.housing.contains(.mortgage))
    }

    @Test func drivingNegation() {
        #expect(AboutParser.parse("Не вожу, езжу на метро").drives == false)
        #expect(AboutParser.parse("Вожу каждый день").drives == true)
    }

    @Test func negativePhrasesBecomeMutesAndWordsNotInterests() {
        let r = AboutParser.parse("Люблю космос. Не люблю футбол и политику, надоели сплетни")
        #expect(r.interests == [.space])
        #expect(r.mutedTopics == [.sport, .politics])
        #expect(r.blockedWords.contains("сплетн"))
        #expect(!r.interests.contains(.sport))
    }

    @Test func cueWordsAndHousingDoNotBecomeInterestsOrBlockedWords() {
        let r = AboutParser.parse("Не люблю футбол и политику, надоели сплетни")
        #expect(r.blockedWords == ["сплетн"])
        let h = AboutParser.parse("Снимаю квартиру")
        #expect(h.interests.isEmpty)
        #expect(h.housing == [.renter])
    }

    @Test func dismissedChipsAreRemoved() {
        let r = AboutParser.parse("Живу в Казани, ИП, люблю космос")
        let trimmed = r.without(["region:16", "interest:space"])
        #expect(trimmed.regionCode == nil)
        #expect(trimmed.interests.isEmpty)
        #expect(trimmed.work == [.ip])
    }

    @Test func applyAddsWithoutErasingWhatWasChosen() {
        var profile = UserProfile(work: [.employee])
        var prefs = FilterPreferences.default
        AboutParser.parse("Живу в Казани, ИП, люблю космос, не люблю футбол").apply(to: &profile, preferences: &prefs)
        #expect(profile.regionCode == "16")
        #expect(profile.work == [.employee, .ip])
        #expect(prefs.interests.contains(.space))
        #expect(prefs.stopTopics.contains(.sport))
        #expect(prefs.stopTopics.contains(.politics))
    }

    @Test func allRegionsAreUniqueAndSearchable() {
        let codes = Region.all.map(\.code)
        #expect(Set(codes).count == codes.count)
        #expect(Region.all.count == 85)
        #expect(Region.search("каз").map(\.code) == ["16"])
        #expect(Region.search("").count == Region.all.count)
    }

    @Test func profileSummaryLine() {
        var p = UserProfile(work: [.ip], drives: true, regionCode: "16")
        var prefs = FilterPreferences.default
        prefs.interests = [.space]
        #expect(ProfileSummary.line(profile: p, preferences: prefs) == "Татарстан · ИП · водитель · Космос")
        p = UserProfile()
        prefs.interests = []
        #expect(ProfileSummary.line(profile: p, preferences: prefs) == nil)
    }
}
