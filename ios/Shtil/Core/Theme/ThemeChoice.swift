import Foundation

enum ThemeChoice: String, Codable, CaseIterable, Sendable {
    case paper
    case sage
    case dusk

    var title: String {
        switch self {
        case .paper: "Бумага"
        case .sage: "Шалфей"
        case .dusk: "Сумерки"
        }
    }

    var subtitle: String {
        switch self {
        case .paper: "Собранно, как хорошая газета"
        case .sage: "Мягко и спокойно, как утро"
        case .dusk: "Тёмная, бережёт глаза вечером"
        }
    }
}

enum ThemeResolver {
    /// Сумерки включаются с 19:00 и держатся до 06:00 (верхняя граница не задана в плане, выбрана здесь).
    static let duskStartHour = 19
    static let duskEndHour = 6

    static func resolve(
        selected: ThemeChoice,
        autoDusk: Bool,
        now: Date,
        calendar: Calendar = .current
    ) -> ThemeChoice {
        guard autoDusk else { return selected }
        let hour = calendar.component(.hour, from: now)
        return (hour >= duskStartHour || hour < duskEndHour) ? .dusk : selected
    }
}
