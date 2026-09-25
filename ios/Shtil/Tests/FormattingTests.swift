import Foundation
import Testing
@testable import Shtil

@Suite struct FormattingTests {
    private let today = CalendarDate(year: 2026, month: 9, day: 25)

    @Test func pluralRules() {
        func d(_ n: Int) -> String { RuFormat.daysWords(n) }
        #expect(d(1) == "1 день")
        #expect(d(2) == "2 дня")
        #expect(d(5) == "5 дней")
        #expect(d(11) == "11 дней")
        #expect(d(12) == "12 дней")
        #expect(d(21) == "21 день")
        #expect(d(22) == "22 дня")
        #expect(d(111) == "111 дней")
    }

    @Test func countdowns() {
        #expect(RuFormat.countdownShort(days: 6) == "Д–6")
        #expect(RuFormat.countdownShort(days: 0) == "Сегодня")
        #expect(RuFormat.countdownWords(days: 0) == "сегодня")
        #expect(RuFormat.countdownWords(days: 1) == "завтра")
        #expect(RuFormat.countdownWords(days: 6) == "через 6 дней")
        #expect(RuFormat.countdownWords(days: 20) == "через 20 дней")
        #expect(RuFormat.countdownWords(days: 98) == "через 3 месяца")
        #expect(RuFormat.countdownWords(days: 157) == "через 5 месяцев")
    }

    @Test func dates() {
        let oct1 = CalendarDate(year: 2026, month: 10, day: 1)
        let mar27 = CalendarDate(year: 2027, month: 3, day: 1)
        #expect(RuFormat.dayMonth(oct1, today: today) == "1 октября")
        #expect(RuFormat.dayMonth(mar27, today: today) == "1 марта 2027")
        #expect(RuFormat.numeric(oct1) == "01.10.2026")
        #expect(RuFormat.numericShort(oct1, today: today) == "01.10")
        #expect(RuFormat.numericShort(mar27, today: today) == "01.03.27")
        #expect(RuFormat.calendarTile(oct1, today: today) == ("01", "Окт"))
        #expect(RuFormat.calendarTile(CalendarDate(year: 2027, month: 1, day: 1), today: today) == ("01", "Янв 27"))
        #expect(RuFormat.dayMonthShort(oct1) == "1 окт")
    }

    @Test func weekdays() {
        #expect(RuFormat.weekdayShort(today) == "Пт")
        #expect(RuFormat.longDay(today) == "Пятница, 25 сентября")
    }

    @Test func greetingByHour() {
        #expect(RuFormat.greeting(hour: 8) == "Доброе утро.")
        #expect(RuFormat.greeting(hour: 14) == "Добрый день.")
        #expect(RuFormat.greeting(hour: 20) == "Добрый вечер.")
        #expect(RuFormat.greeting(hour: 2) == "Доброй ночи.")
    }

    @Test func clock() {
        #expect(RuFormat.clock(minutes: 480) == "08:00")
        #expect(RuFormat.clock(minutes: 1140) == "19:00")
    }

    @Test func audienceLabelsPreferMatchedTag() {
        let law = TestData.law("x", tags: ["work:employee", "housing:renter"])
        let tags = AudienceMatcher.audienceTags(for: UserProfile(housing: [.renter]))
        #expect(AudienceLabels.label(for: law, profileTags: tags) == "Снимаю жильё")
        #expect(AudienceLabels.label(for: law, profileTags: []) == "Работающие по найму")
    }

    @Test func reasonUsesOnboardingAnswers() {
        let usn = TestData.law("usn", tags: ["work:ip_usn"])
        #expect(AudienceLabels.reason(for: usn, profile: UserProfile(work: [.ip])) == "В профиле указано: «ИП».")
        let all = TestData.law("all", tags: ["all"])
        #expect(AudienceLabels.reason(for: all, profile: .empty) == "Закон касается всех.")
        let mil = TestData.law("mil", tags: ["military:registered"])
        #expect(AudienceLabels.reason(for: mil, profile: UserProfile(gender: .male, age: .a20to25)) == "В профиле указано: «Мужчина 20–25».")
    }
}

@Suite struct ScheduleTests {
    private var cal: Calendar {
        var c = Calendar(identifier: .gregorian)
        c.timeZone = TimeZone(identifier: "Europe/Moscow")!
        return c
    }

    private func at(_ day: Int, _ hour: Int, _ minute: Int = 0) -> Date {
        cal.date(from: DateComponents(year: 2026, month: 9, day: day, hour: hour, minute: minute))!
    }

    private let both = EditionSchedule(preference: .both, morningMinutes: 480, eveningMinutes: 1140)

    @Test func nextEditionSameDayAndTomorrow() {
        #expect(both.nextCaption(after: at(25, 8, 30), calendar: cal) == "сегодня в 19:00")
        #expect(both.nextCaption(after: at(25, 3), calendar: cal) == "сегодня в 08:00")
        #expect(both.nextCaption(after: at(25, 19, 0), calendar: cal) == "завтра в 08:00")
        #expect(both.nextCaption(after: at(25, 23), calendar: cal) == "завтра в 08:00")
    }

    @Test func singleSlotSchedules() {
        let am = EditionSchedule(preference: .am, morningMinutes: 480, eveningMinutes: 1140)
        let pm = EditionSchedule(preference: .pm, morningMinutes: 480, eveningMinutes: 1140)
        #expect(am.nextCaption(after: at(25, 12), calendar: cal) == "завтра в 08:00")
        #expect(pm.nextCaption(after: at(25, 12), calendar: cal) == "сегодня в 19:00")
    }

    @Test func slotsAreSorted() {
        let odd = EditionSchedule(preference: .both, morningMinutes: 1200, eveningMinutes: 300)
        #expect(odd.slots == [300, 1200])
    }

    @Test func reminderWeekBefore() {
        let plan = ReminderPlanner.plan(effective: CalendarDate(year: 2026, month: 12, day: 1), now: at(25, 12), calendar: cal)
        #expect(plan?.kind == .weekBefore)
        #expect(plan?.day == CalendarDate(year: 2026, month: 11, day: 24))
        #expect(plan?.fireDate == cal.date(from: DateComponents(year: 2026, month: 11, day: 24, hour: 9)))
    }

    @Test func reminderFallsBackToDayBeforeWhenLessThanAWeek() {
        let plan = ReminderPlanner.plan(effective: CalendarDate(year: 2026, month: 10, day: 1), now: at(25, 12), calendar: cal)
        #expect(plan?.kind == .dayBefore)
        #expect(plan?.day == CalendarDate(year: 2026, month: 9, day: 30))
        let title = plan.map { ReminderPlanner.buttonTitle($0, today: CalendarDate(year: 2026, month: 9, day: 25)) }
        #expect(title == "Напомнить накануне · 30.09")
    }

    @Test func noReminderWhenTooLate() {
        // Накануне 09:00 уже прошло.
        #expect(ReminderPlanner.plan(effective: CalendarDate(year: 2026, month: 9, day: 26), now: at(25, 12), calendar: cal) == nil)
        #expect(ReminderPlanner.plan(effective: CalendarDate(year: 2026, month: 9, day: 20), now: at(25, 12), calendar: cal) == nil)
    }
}
