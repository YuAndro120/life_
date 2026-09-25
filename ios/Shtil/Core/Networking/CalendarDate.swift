import Foundation

/// Дата без времени и часового пояса («2026-10-01»). Даты законов приходят именно так,
/// поэтому отсчёт «Д–N» не зависит от пояса устройства.
struct CalendarDate: Hashable, Comparable, Codable, Sendable {
    let year: Int
    let month: Int
    let day: Int

    init(year: Int, month: Int, day: Int) {
        self.year = year
        self.month = month
        self.day = day
    }

    /// Принимает «yyyy-MM-dd» и полный ISO-8601 (берётся дата-префикс).
    init?(string: String) {
        let parts = string.prefix(10).split(separator: "-")
        guard parts.count == 3,
              let y = Int(parts[0]), let m = Int(parts[1]), let d = Int(parts[2]),
              (1...12).contains(m), (1...31).contains(d)
        else { return nil }
        self.init(year: y, month: m, day: d)
    }

    init(_ date: Date, calendar: Calendar = .current) {
        let c = calendar.dateComponents([.year, .month, .day], from: date)
        self.init(year: c.year ?? 1970, month: c.month ?? 1, day: c.day ?? 1)
    }

    init(from decoder: Decoder) throws {
        let raw = try decoder.singleValueContainer().decode(String.self)
        guard let value = CalendarDate(string: raw) else {
            throw DecodingError.dataCorrupted(
                .init(codingPath: decoder.codingPath, debugDescription: "Неверная дата: \(raw)")
            )
        }
        self = value
    }

    func encode(to encoder: Encoder) throws {
        var c = encoder.singleValueContainer()
        try c.encode(isoString)
    }

    var isoString: String { String(format: "%04d-%02d-%02d", year, month, day) }

    static func < (l: CalendarDate, r: CalendarDate) -> Bool {
        (l.year, l.month, l.day) < (r.year, r.month, r.day)
    }

    func date(calendar: Calendar = .current) -> Date? {
        calendar.date(from: DateComponents(year: year, month: month, day: day))
    }

    /// Целых дней от `from` до этой даты (отрицательное — уже прошла).
    func days(from other: CalendarDate, calendar: Calendar = .current) -> Int {
        guard let a = other.date(calendar: calendar), let b = date(calendar: calendar) else { return 0 }
        return calendar.dateComponents([.day], from: a, to: b).day ?? 0
    }

    func adding(days: Int, calendar: Calendar = .current) -> CalendarDate {
        guard let d = date(calendar: calendar),
              let shifted = calendar.date(byAdding: .day, value: days, to: d)
        else { return self }
        return CalendarDate(shifted, calendar: calendar)
    }
}
