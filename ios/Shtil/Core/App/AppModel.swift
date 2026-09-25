import Foundation
import Observation
import SwiftData

@MainActor
@Observable
final class AppModel {
    let context: ModelContext
    let profile: Profile
    let settings: FilterSettings
    let counter: EditionCounter
    private let source: any ContentSource
    /// Запасной источник для первого запуска без сети и без кэша (в DEBUG — встроенные тестовые данные).
    private let fallback: (any ContentSource)?
    private let notifier: any NotificationScheduling

    private(set) var feed: Feed?
    private(set) var laws: [Law] = []
    private(set) var reminderLawIds: Set<String> = []
    private(set) var loadError: String?
    /// Данные взяты из кэша, потому что сервер недоступен.
    private(set) var isOffline = false
    /// Показаны встроенные тестовые данные, потому что сервер недоступен, а кэша ещё нет.
    private(set) var isDemoData = false
    /// Когда данные последний раз получены с сервера (или из кэша при офлайне).
    private(set) var lastSyncedAt: Date?
    private let cache: ContentCache
    /// Последнее действие «Не интересно» для кнопки «Отменить».
    private(set) var undo: FeedbackAction?
    /// Текущее время; обновляется раз в минуту, чтобы тема и «следующий выпуск» не устаревали.
    var now: Date

    init(
        context: ModelContext,
        source: any ContentSource = FixtureContentSource(),
        fallback: (any ContentSource)? = nil,
        notifier: any NotificationScheduling = NoopNotificationScheduler(),
        now: Date = .now
    ) {
        self.context = context
        self.source = source
        self.fallback = fallback
        self.notifier = notifier
        self.now = now
        self.profile = Self.fetchOrCreate(Profile.self, in: context) { Profile() }
        self.settings = Self.fetchOrCreate(FilterSettings.self, in: context) { FilterSettings() }
        self.counter = Self.fetchOrCreate(EditionCounter.self, in: context) { EditionCounter() }
        self.reminderLawIds = Set(((try? context.fetch(FetchDescriptor<Reminder>())) ?? []).map(\.lawId))
        self.cache = ContentCache(context: context)
        // Показываем сохранённый выпуск сразу, до ответа сети.
        if let snapshot = cache.load() {
            feed = snapshot.feed
            laws = snapshot.laws
            lastSyncedAt = snapshot.fetchedAt
        }
    }

    private static func fetchOrCreate<T: PersistentModel>(
        _ type: T.Type, in context: ModelContext, make: () -> T
    ) -> T {
        if let existing = try? context.fetch(FetchDescriptor<T>()).first { return existing }
        let created = make()
        context.insert(created)
        try? context.save()
        return created
    }

    // MARK: производные значения

    var themeChoice: ThemeChoice {
        ThemeResolver.resolve(selected: settings.themeChoice, autoDusk: settings.autoDusk, now: now)
    }

    var theme: Theme { .of(themeChoice) }
    var today: CalendarDate { CalendarDate(now) }

    var schedule: EditionSchedule {
        EditionSchedule(
            preference: settings.schedulePreference,
            morningMinutes: settings.morningMinutes,
            eveningMinutes: settings.eveningMinutes
        )
    }

    var onboardingCompleted: Bool { profile.onboardingCompleted }

    var edition: Edition? {
        guard let feed else { return nil }
        return FilterEngine.edition(
            number: max(counter.number, 1),
            feed: feed,
            laws: laws,
            profile: profile.snapshot,
            preferences: settings.preferences,
            window: EditionWindow(start: nil, end: max(feed.generatedAt, now)),
            today: today
        )
    }

    /// Законы для календаря: только подписанные и ещё не вступившие в силу, по дате вступления.
    func calendarLaws(onlyMine: Bool) -> [Law] {
        let signed = laws.filter { $0.status == .signed && $0.dates.effective != nil }
        if onlyMine {
            return AudienceMatcher.relevantLaws(signed, profile: profile.snapshot, asOf: today)
        }
        return signed.filter { ($0.dates.effective ?? today) >= today }.sorted(by: AudienceMatcher.lawOrder)
    }

    func law(id: String) -> Law? { laws.first { $0.id == id } }

    // MARK: загрузка

    func refresh() async {
        do {
            async let f = source.feed()
            async let l = source.laws()
            let (newFeed, newLaws) = try await (f, l)
            feed = newFeed
            laws = newLaws
            loadError = nil
            isOffline = false
            isDemoData = false
            lastSyncedAt = Date.now
            cache.save(feed: newFeed, laws: newLaws, fetchedAt: lastSyncedAt ?? .now)
        } catch {
            if feed == nil {
                if let fallback, let demoFeed = try? await fallback.feed(), let demoLaws = try? await fallback.laws() {
                    // В кэш не пишем: иначе демо-данные потом выдавались бы за настоящие.
                    feed = demoFeed
                    laws = demoLaws
                    isDemoData = true
                    loadError = nil
                } else {
                    loadError = "Не удалось загрузить выпуск"
                }
            } else {
                isOffline = true
            }
        }
    }

    func tick(now: Date = .now) { self.now = now }

    // MARK: «Не интересно»

    func apply(_ action: FeedbackAction) {
        var p = settings.preferences
        guard action.apply(to: &p) else { return }
        settings.preferences = p
        save()
        undo = action
    }

    func undoLast() {
        guard let action = undo else { return }
        var p = settings.preferences
        action.revert(on: &p)
        settings.preferences = p
        save()
        undo = nil
    }

    func clearUndo() { undo = nil }

    // MARK: онбординг

    func completeOnboarding() async {
        profile.onboardingCompleted = true
        if counter.number < 1 { counter.number = 1 }
        counter.lastEditionAt = now
        save()
        _ = await notifier.requestAuthorization()
        await notifier.scheduleEditions(schedule)
    }

    // MARK: настройки

    func save() { try? context.save() }

    /// Вызывать после изменения расписания.
    func scheduleChanged() {
        save()
        Task { await notifier.scheduleEditions(schedule) }
    }

    // MARK: напоминания

    func hasReminder(_ law: Law) -> Bool { reminderLawIds.contains(law.id) }

    func reminderPlan(for law: Law) -> ReminderPlanner.Plan? {
        law.dates.effective.flatMap { ReminderPlanner.plan(effective: $0, now: now) }
    }

    func toggleReminder(_ law: Law) {
        if hasReminder(law) {
            reminderLawIds.remove(law.id)
            if let existing = try? context.fetch(FetchDescriptor<Reminder>(predicate: #Predicate { $0.lawId == law.id })) {
                existing.forEach(context.delete)
            }
            save()
            Task { await notifier.cancelLawReminder(lawId: law.id) }
        } else if let plan = reminderPlan(for: law) {
            reminderLawIds.insert(law.id)
            context.insert(Reminder(lawId: law.id, fireDate: plan.fireDate))
            save()
            let title = law.title
            Task {
                _ = await notifier.requestAuthorization()
                await notifier.scheduleLawReminder(lawId: law.id, title: title, fireDate: plan.fireDate)
            }
        }
    }
}
