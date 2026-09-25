import Foundation

/// Расписание выпусков: время «Выпуск готов» и подпись «Следующий — сегодня в 19:00».
struct EditionSchedule: Equatable, Sendable {
    var preference: SchedulePreference
    var morningMinutes: Int
    var eveningMinutes: Int

    var slots: [Int] {
        switch preference {
        case .both: [morningMinutes, eveningMinutes].sorted()
        case .am: [morningMinutes]
        case .pm: [eveningMinutes]
        }
    }

    struct Next: Equatable, Sendable {
        var date: Date
        var isToday: Bool
        var minutes: Int
    }

    /// Ближайший выпуск строго после `now` (сегодня или завтра).
    func next(after now: Date, calendar: Calendar = .current) -> Next? {
        let start = calendar.startOfDay(for: now)
        for dayOffset in 0...1 {
            guard let day = calendar.date(byAdding: .day, value: dayOffset, to: start) else { continue }
            for minutes in slots {
                guard let candidate = calendar.date(byAdding: .minute, value: minutes, to: day) else { continue }
                if candidate > now { return Next(date: candidate, isToday: dayOffset == 0, minutes: minutes) }
            }
        }
        return nil
    }

    /// «Сегодня в 19:00» / «Завтра в 08:00».
    func nextCaption(after now: Date, calendar: Calendar = .current) -> String? {
        guard let next = next(after: now, calendar: calendar) else { return nil }
        return "\(next.isToday ? "сегодня" : "завтра") в \(RuFormat.clock(minutes: next.minutes))"
    }
}
