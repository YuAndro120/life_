import Foundation

/// Когда напомнить о вступлении закона в силу: за 7 дней, а если осталось меньше — накануне.
enum ReminderPlanner {
    static let hour = 9
    static let leadDays = 7

    enum Kind: Equatable, Sendable {
        case weekBefore
        case dayBefore
    }

    struct Plan: Equatable, Sendable {
        var kind: Kind
        var fireDate: Date
        var day: CalendarDate
    }

    static func plan(effective: CalendarDate, now: Date, calendar: Calendar = .current) -> Plan? {
        for (kind, days) in [(Kind.weekBefore, leadDays), (Kind.dayBefore, 1)] {
            let day = effective.adding(days: -days, calendar: calendar)
            guard let base = day.date(calendar: calendar),
                  let fire = calendar.date(bySettingHour: hour, minute: 0, second: 0, of: base),
                  fire > now
            else { continue }
            return Plan(kind: kind, fireDate: fire, day: day)
        }
        return nil
    }

    /// Текст кнопки: «Напомнить накануне · 30.09».
    static func buttonTitle(_ plan: Plan, today: CalendarDate) -> String {
        switch plan.kind {
        case .weekBefore: "Напомнить за 7 дней · \(RuFormat.numericShort(plan.day, today: today))"
        case .dayBefore: "Напомнить накануне · \(RuFormat.numericShort(plan.day, today: today))"
        }
    }
}
