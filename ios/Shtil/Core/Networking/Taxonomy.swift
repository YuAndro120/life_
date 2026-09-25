import Foundation

/// Таксономии из раздела 7 plan.md, общие для сервера и iOS.
enum Topic: String, Codable, CaseIterable, Sendable {
    case economy, finance, law
    case techAi = "tech_ai"
    case city, health, education, transport, housing, science, culture, sport, showbiz, crypto
    case politics, crime, incidents, disasters

    var title: String {
        switch self {
        case .economy: "Экономика"
        case .finance: "Финансы"
        case .law: "Право"
        case .techAi: "ИИ и технологии"
        case .city: "Город"
        case .health: "Здоровье"
        case .education: "Образование"
        case .transport: "Транспорт"
        case .housing: "Жильё"
        case .science: "Наука"
        case .culture: "Культура"
        case .sport: "Спорт"
        case .showbiz: "Шоу-бизнес"
        case .crypto: "Криптовалюты"
        case .politics: "Политика"
        case .crime: "Криминал"
        case .incidents: "Происшествия"
        case .disasters: "Катастрофы"
        }
    }
}

enum InfoType: String, Codable, CaseIterable, Sendable {
    case fact, official, opinion, forecast, rumor

    var title: String {
        switch self {
        case .fact: "Факт"
        case .official: "Решение"
        case .opinion: "Мнение"
        case .forecast: "Прогноз"
        case .rumor: "Неподтверждённое"
        }
    }
}

enum Heaviness: String, Codable, CaseIterable, Sendable {
    case neutral, tense, heavy
}

enum LawStatus: String, Codable, CaseIterable, Sendable {
    case introduced, passed, signed
    case inForce = "in_force"
}
