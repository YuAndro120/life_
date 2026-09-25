import Foundation

/// Русское форматирование дат и отсчётов. Без `DateFormatter`, чтобы результат не зависел от локали устройства.
enum RuFormat {
    private static let monthsGenitive = [
        "января", "февраля", "марта", "апреля", "мая", "июня",
        "июля", "августа", "сентября", "октября", "ноября", "декабря",
    ]
    private static let monthsShort = ["Янв", "Фев", "Мар", "Апр", "Май", "Июн", "Июл", "Авг", "Сен", "Окт", "Ноя", "Дек"]
    private static let weekdaysShort = ["Вс", "Пн", "Вт", "Ср", "Чт", "Пт", "Сб"]
    private static let weekdaysFull = [
        "Воскресенье", "Понедельник", "Вторник", "Среда", "Четверг", "Пятница", "Суббота",
    ]

    static func plural(_ n: Int, one: String, few: String, many: String) -> String {
        let n10 = abs(n) % 10, n100 = abs(n) % 100
        if n10 == 1 && n100 != 11 { return one }
        if (2...4).contains(n10) && !(12...14).contains(n100) { return few }
        return many
    }

    static func two(_ n: Int) -> String { String(format: "%02d", n) }

    /// «1 октября», а для другого года «1 марта 2027».
    static func dayMonth(_ d: CalendarDate, today: CalendarDate) -> String {
        let base = "\(d.day) \(monthsGenitive[d.month - 1])"
        return d.year == today.year ? base : "\(base) \(d.year)"
    }

    static func dayMonthYear(_ d: CalendarDate) -> String {
        "\(d.day) \(monthsGenitive[d.month - 1]) \(d.year)"
    }

    /// «01.10.2026».
    static func numeric(_ d: CalendarDate) -> String {
        "\(two(d.day)).\(two(d.month)).\(d.year)"
    }

    /// «01.10», а для другого года «01.03.27».
    static func numericShort(_ d: CalendarDate, today: CalendarDate) -> String {
        let base = "\(two(d.day)).\(two(d.month))"
        return d.year == today.year ? base : "\(base).\(two(d.year % 100))"
    }

    /// Для плитки календаря: («01», «Окт») или («01», «Янв 27»).
    static func calendarTile(_ d: CalendarDate, today: CalendarDate) -> (day: String, month: String) {
        let m = monthsShort[d.month - 1]
        return (two(d.day), d.year == today.year ? m : "\(m) \(two(d.year % 100))")
    }

    /// Только месяц для короткой подписи в плане/статусе: «1 окт».
    static func dayMonthShort(_ d: CalendarDate) -> String {
        "\(d.day) \(monthsShort[d.month - 1].lowercased())"
    }

    static func weekdayIndex(_ d: CalendarDate) -> Int {
        var cal = Calendar(identifier: .gregorian)
        cal.timeZone = TimeZone(identifier: "UTC") ?? .gmt
        let date = cal.date(from: DateComponents(year: d.year, month: d.month, day: d.day)) ?? Date(timeIntervalSince1970: 0)
        return cal.component(.weekday, from: date) - 1
    }

    static func weekdayShort(_ d: CalendarDate) -> String { weekdaysShort[weekdayIndex(d)] }
    static func weekdayFull(_ d: CalendarDate) -> String { weekdaysFull[weekdayIndex(d)] }

    /// «Д–6»; в день вступления — «Сегодня».
    static func countdownShort(days: Int) -> String {
        days <= 0 ? "Сегодня" : "Д–\(days)"
    }

    /// «6 дней» — для крупной цифры в карточке закона.
    static func daysWords(_ days: Int) -> String {
        "\(days) \(plural(days, one: "день", few: "дня", many: "дней"))"
    }

    /// «через 6 дней», «через 3 месяца», «завтра», «сегодня».
    static func countdownWords(days: Int) -> String {
        switch days {
        case ..<1: return "сегодня"
        case 1: return "завтра"
        case 2..<60: return "через \(daysWords(days))"
        default:
            let months = days / 30
            return "через \(months) \(plural(months, one: "месяц", few: "месяца", many: "месяцев"))"
        }
    }

    /// «Пт 25.09 — 08:00».
    static func editionStamp(date: Date, calendar: Calendar = .current) -> String {
        let d = CalendarDate(date, calendar: calendar)
        let c = calendar.dateComponents([.hour, .minute], from: date)
        return "\(weekdayShort(d)) \(two(d.day)).\(two(d.month)) — \(two(c.hour ?? 0)):\(two(c.minute ?? 0))"
    }

    /// «Пятница, 25 сентября».
    static func longDay(_ d: CalendarDate) -> String {
        "\(weekdayFull(d)), \(d.day) \(monthsGenitive[d.month - 1])"
    }

    static func clock(minutes: Int) -> String {
        "\(two(minutes / 60 % 24)):\(two(minutes % 60))"
    }

    static func greeting(hour: Int) -> String {
        switch hour {
        case 5..<12: "Доброе утро."
        case 12..<18: "Добрый день."
        case 18..<23: "Добрый вечер."
        default: "Доброй ночи."
        }
    }

    static func readingMinutes(_ n: Int) -> String { two(n) }
}
