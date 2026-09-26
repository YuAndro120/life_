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

    @Test func occupationsAndSells() {
        let r = AboutParser.parse("Я программист, живу в Москве")
        #expect(r.occupations == [.it])
        let b = AboutParser.parse("Занимаюсь торговлей на Wildberries, продаю одежду и обувь")
        #expect(b.sells.contains(.online))
        #expect(b.sells.contains(.marked))
        #expect(b.work == [.ip])
        #expect(b.occupations.contains(.trade))
        let s = AboutParser.parse("ИП, делаю маникюр, мастер салона красоты")
        #expect(s.occupations.contains(.beauty))
        #expect(s.sells.contains(.services))
    }

    @Test func aStudentWhoRentsIsNotASeller() {
        let r = AboutParser.parse("Студент, снимаю квартиру, заказываю доставку")
        #expect(r.sells.isEmpty)
        #expect(r.work == [.student])
        #expect(r.occupations.isEmpty)
    }

    @Test func occupationAndSellsBecomeAudienceTags() {
        let p = UserProfile(work: [.ip], occupations: [.trade], sells: [.marked, .online])
        let tags = AudienceMatcher.audienceTags(for: p)
        #expect(tags.contains("industry:trade"))
        #expect(tags.contains("sells:marked"))
        #expect(tags.contains("sells:online"))
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

@Suite struct AudienceGatingTests {
    private func matches(_ tags: [String], profile: UserProfile) -> Bool {
        let law = TestData.law("l", tags: tags)
        return AudienceMatcher.matches(law, tags: AudienceMatcher.audienceTags(for: profile), regionCode: profile.regionCode)
    }

    @Test func industryMustMatchWhenProfileAnswered() {
        let it = UserProfile(work: [.employee], occupations: [.it])
        #expect(!matches(["work:employee", "industry:health"], profile: it))
        #expect(matches(["work:employee", "industry:it"], profile: it))
        #expect(matches(["work:employee"], profile: it))
    }

    @Test func unansweredProfileIsNotGated() {
        let plain = UserProfile(work: [.employee])
        #expect(matches(["work:employee", "industry:health"], profile: plain))
    }

    @Test func sellsMustMatchWhenProfileAnswered() {
        let seller = UserProfile(work: [.ip], sells: [.services])
        #expect(!matches(["work:ip", "sells:alcohol"], profile: seller))
        #expect(matches(["work:ip", "sells:services"], profile: seller))
        #expect(matches(["work:ip"], profile: seller))
    }
}
