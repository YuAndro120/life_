import Foundation

/// Страны изданий, из которых пользователь выбирает «откуда читать».
enum NewsCountry: String, CaseIterable, Sendable {
    case ru = "RU"
    case us = "US"
    case gb = "GB"
    case eu = "EU"

    var title: String {
        switch self {
        case .ru: "Россия"
        case .us: "США"
        case .gb: "Великобритания"
        case .eu: "Европа"
        }
    }
}
