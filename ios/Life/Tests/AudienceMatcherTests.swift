import Testing
@testable import Life

@Suite struct AudienceMatcherTests {
    private let sample = UserProfile(
        gender: .male, age: .a20to25, work: [.ip], housing: [.renter], drives: true, regionCode: nil
    )

    @Test func tagsFromOnboardingAnswers() {
        let tags = AudienceMatcher.audienceTags(for: sample)
        #expect(tags == [
            "gender:male", "age:20_25", "work:ip", "work:ip_usn",
            "housing:renter", "transport:driver", "military:registered",
        ])
    }

    @Test func emptyProfileHasNoTags() {
        #expect(AudienceMatcher.audienceTags(for: .empty).isEmpty)
    }

    @Test func nonDriverHasNoDriverTag() {
        var p = sample
        p.drives = false
        #expect(!AudienceMatcher.audienceTags(for: p).contains("transport:driver"))
    }

    @Test func militaryOnlyForMenUpTo35Bracket() {
        func tags(_ g: UserProfile.Gender, _ a: UserProfile.AgeBracket) -> Bool {
            AudienceMatcher.audienceTags(for: UserProfile(gender: g, age: a)).contains("military:registered")
        }
        #expect(tags(.male, .u20))
        #expect(tags(.male, .a26to35))
        #expect(!tags(.male, .a36to50))
        #expect(!tags(.male, .a50plus))
        #expect(!tags(.female, .a20to25))
    }

    @Test func regionTagAppearsWithRegion() {
        let tags = AudienceMatcher.audienceTags(for: UserProfile(regionCode: "77"))
        #expect(tags == ["region:77"])
    }

    @Test func intersectionMatches() {
        let laws = [
            TestData.law("usn", tags: ["work:ip_usn"]),
            TestData.law("emp", tags: ["work:employee"]),
            TestData.law("multi", tags: ["work:employee", "housing:renter"]),
        ]
        let ids = AudienceMatcher.relevantLaws(laws, profile: sample).map(\.id)
        #expect(Set(ids) == ["usn", "multi"])
    }

    @Test func allTagMatchesEveryoneIncludingEmptyProfile() {
        let laws = [TestData.law("everyone", tags: ["all"])]
        #expect(AudienceMatcher.relevantLaws(laws, profile: .empty).count == 1)
    }

    @Test func regionalLawRequiresSameRegion() {
        let law = TestData.law("msk", tags: ["work:student", "region:77"], region: "77")
        let student = UserProfile(work: [.student], regionCode: "77")
        let otherRegion = UserProfile(work: [.student], regionCode: "78")
        let noRegion = UserProfile(work: [.student])
        #expect(!AudienceMatcher.relevantLaws([law], profile: student).isEmpty)
        #expect(AudienceMatcher.relevantLaws([law], profile: otherRegion).isEmpty)
        #expect(AudienceMatcher.relevantLaws([law], profile: noRegion).isEmpty)
    }

    @Test func regionalLawWithAllTagStillNeedsRegion() {
        let law = TestData.law("msk-all", tags: ["all"], region: "77")
        #expect(AudienceMatcher.relevantLaws([law], profile: UserProfile(regionCode: "78")).isEmpty)
        #expect(!AudienceMatcher.relevantLaws([law], profile: UserProfile(regionCode: "77")).isEmpty)
    }

    @Test func federalLawIgnoresProfileRegion() {
        let law = TestData.law("fed", tags: ["work:employee"])
        #expect(!AudienceMatcher.relevantLaws([law], profile: UserProfile(work: [.employee], regionCode: "78")).isEmpty)
    }

    @Test func sortedByEffectiveDateNilLast() {
        let laws = [
            TestData.law("c", tags: ["all"], effective: nil),
            TestData.law("b", tags: ["all"], effective: "2027-03-01"),
            TestData.law("a", tags: ["all"], effective: "2026-10-01"),
        ]
        #expect(AudienceMatcher.relevantLaws(laws, profile: .empty).map(\.id) == ["a", "b", "c"])
    }

    @Test func asOfDropsLawsAlreadyInForce() {
        let laws = [
            TestData.law("past", tags: ["all"], effective: "2026-09-01", status: .inForce),
            TestData.law("today", tags: ["all"], effective: "2026-09-25"),
            TestData.law("future", tags: ["all"], effective: "2026-10-01"),
            TestData.law("undated", tags: ["all"], effective: nil),
        ]
        let ids = AudienceMatcher.relevantLaws(laws, profile: .empty, asOf: TestData.today).map(\.id)
        #expect(ids == ["today", "future", "undated"])
    }

    @Test func fixtureLawsForSampleProfile() throws {
        let laws = try Fixtures.laws()
        let ids = AudienceMatcher.relevantLaws(laws, profile: sample, asOf: TestData.today).map(\.id)
        // УСН (01.10.26), аренда (01.12.26), «для всех» (01.01.27), водитель (01.03.27).
        #expect(ids == ["lw_01", "lw_03", "lw_05", "lw_02"])
    }
}
