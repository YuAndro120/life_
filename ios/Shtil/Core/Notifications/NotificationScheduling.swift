import Foundation
import UserNotifications

/// Локальные уведомления: «Выпуск готов» по расписанию и напоминания о законах.
/// Push (APNs) в MVP недоступен: бесплатная подпись его не поддерживает.
protocol NotificationScheduling: Sendable {
    func requestAuthorization() async -> Bool
    func scheduleEditions(_ schedule: EditionSchedule) async
    func scheduleLawReminder(lawId: String, title: String, fireDate: Date) async
    func cancelLawReminder(lawId: String) async
}

struct NoopNotificationScheduler: NotificationScheduling {
    func requestAuthorization() async -> Bool { false }
    func scheduleEditions(_ schedule: EditionSchedule) async {}
    func scheduleLawReminder(lawId: String, title: String, fireDate: Date) async {}
    func cancelLawReminder(lawId: String) async {}
}

struct SystemNotificationScheduler: NotificationScheduling {
    private static let editionPrefix = "edition-"
    private static let lawPrefix = "law-"

    private var center: UNUserNotificationCenter { .current() }

    func requestAuthorization() async -> Bool {
        (try? await center.requestAuthorization(options: [.alert, .sound])) ?? false
    }

    func scheduleEditions(_ schedule: EditionSchedule) async {
        let ids = ["am", "pm"].map { Self.editionPrefix + $0 }
        center.removePendingNotificationRequests(withIdentifiers: ids)

        let slots: [(String, Int)] = {
            switch schedule.preference {
            case .both: [("am", schedule.morningMinutes), ("pm", schedule.eveningMinutes)]
            case .am: [("am", schedule.morningMinutes)]
            case .pm: [("pm", schedule.eveningMinutes)]
            }
        }()

        for (name, minutes) in slots {
            let content = UNMutableNotificationContent()
            content.title = "Выпуск готов"
            content.body = "Можно дочитать до конца."
            content.sound = .default
            var comps = DateComponents()
            comps.hour = minutes / 60 % 24
            comps.minute = minutes % 60
            let trigger = UNCalendarNotificationTrigger(dateMatching: comps, repeats: true)
            try? await center.add(UNNotificationRequest(identifier: Self.editionPrefix + name, content: content, trigger: trigger))
        }
    }

    func scheduleLawReminder(lawId: String, title: String, fireDate: Date) async {
        let content = UNMutableNotificationContent()
        content.title = "Скоро вступает в силу"
        content.body = title
        content.sound = .default
        let comps = Calendar.current.dateComponents([.year, .month, .day, .hour, .minute], from: fireDate)
        let trigger = UNCalendarNotificationTrigger(dateMatching: comps, repeats: false)
        try? await center.add(UNNotificationRequest(identifier: Self.lawPrefix + lawId, content: content, trigger: trigger))
    }

    func cancelLawReminder(lawId: String) async {
        center.removePendingNotificationRequests(withIdentifiers: [Self.lawPrefix + lawId])
    }
}
