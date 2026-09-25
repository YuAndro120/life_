import Foundation

/// Действия «Не интересно» и «Больше такого». Хранятся только на устройстве, на сервер ничего не отправляется.
enum FeedbackAction: Equatable, Sendable {
    case hideStory(id: String)
    case muteSource(String)
    case muteTopic(Topic)
    case boostTopic(Topic)

    /// Что показать пользователю после действия.
    var message: String {
        switch self {
        case .hideStory: "Сюжет скрыт"
        case .muteSource(let title): "Источник скрыт: \(title)"
        case .muteTopic(let t): "Меньше про: \(t.title)"
        case .boostTopic(let t): "Больше про: \(t.title)"
        }
    }

    /// Применяет действие к настройкам. Возвращает false, если менять было нечего.
    @discardableResult
    func apply(to p: inout FilterPreferences) -> Bool {
        switch self {
        case .hideStory(let id):
            return p.hiddenStories.insert(id).inserted
        case .muteSource(let title):
            return p.mutedSources.insert(title).inserted
        case .muteTopic(let topic):
            guard !p.stopTopics.contains(topic) else { return false }
            p.toggleStopTopic(topic)
            return true
        case .boostTopic(let topic):
            guard !p.interests.contains(topic) else { return false }
            p.toggleInterest(topic)
            return true
        }
    }

    /// Отменяет действие.
    func revert(on p: inout FilterPreferences) {
        switch self {
        case .hideStory(let id): p.hiddenStories.remove(id)
        case .muteSource(let title): p.mutedSources.remove(title)
        case .muteTopic(let topic): p.stopTopics.remove(topic)
        case .boostTopic(let topic): p.interests.remove(topic)
        }
    }
}
