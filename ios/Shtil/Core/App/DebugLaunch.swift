import Foundation

/// Отладочные аргументы запуска для проверки экранов на симуляторе (только DEBUG):
/// `-shtilInMemory YES -shtilSource fixtures -shtilAPI http://192.168.0.5:8080 -shtilSeed sample -shtilTheme sage -shtilTab calendar -shtilLaw lw_01`.
enum DebugLaunch {
    #if DEBUG
    private static var defaults: UserDefaults { .standard }
    static var inMemory: Bool { defaults.bool(forKey: "shtilInMemory") }
    static var seedSample: Bool { defaults.string(forKey: "shtilSeed") == "sample" }
    static var theme: ThemeChoice? { defaults.string(forKey: "shtilTheme").flatMap(ThemeChoice.init(rawValue:)) }
    static var tab: String? { defaults.string(forKey: "shtilTab") }
    static var lawId: String? { defaults.string(forKey: "shtilLaw") }
    static var storyId: String? { defaults.string(forKey: "shtilStory") }
    static var aboutText: String? { defaults.string(forKey: "shtilAbout") }
    /// `-shtilBackup demo` кладёт в отладочное хранилище копию настроек, чтобы посмотреть экран восстановления.
    static var backupStore: any BackupStoring {
        guard defaults.string(forKey: "shtilBackup") == "demo" else { return InMemoryBackupStore() }
        let prefs = { () -> FilterPreferences in var p = FilterPreferences.default; p.interests = [.space]; return p }()
        let demo = ProfileBackup(
            savedAt: .now, profile: UserProfile(work: [.ip], occupations: [.trade], regionCode: "16"), preferences: prefs,
            theme: ThemeChoice.paper.rawValue, autoDusk: true, schedule: SchedulePreference.both.rawValue, morningMinutes: 480, eveningMinutes: 1140
        )
        return InMemoryBackupStore(demo.encoded())
    }
    static var apiOverride: String? { defaults.string(forKey: "shtilAPI") }
    static var useFixtures: Bool { defaults.string(forKey: "shtilSource") == "fixtures" }
    static var onboardingStep: Int? { defaults.string(forKey: "shtilOnbStep").flatMap { Int($0) } }
    static var autoDusk: Bool? { defaults.object(forKey: "shtilAutoDusk") as? Bool }
    static var fixedNow: Date? {
        defaults.string(forKey: "shtilNow").flatMap { ISO8601DateFormatter().date(from: $0) }
    }
    #else
    static var inMemory: Bool { false }
    static var seedSample: Bool { false }
    static var theme: ThemeChoice? { nil }
    static var tab: String? { nil }
    static var lawId: String? { nil }
    static var storyId: String? { nil }
    static var aboutText: String? { nil }
    static var backupStore: any BackupStoring { InMemoryBackupStore() }
    static var apiOverride: String? { nil }
    static var useFixtures: Bool { false }
    static var onboardingStep: Int? { nil }
    static var autoDusk: Bool? { nil }
    static var fixedNow: Date? { nil }
    #endif

    @MainActor
    static func apply(to model: AppModel) {
        #if DEBUG
        if seedSample {
            model.profile.snapshot = UserProfile(
                gender: .male, age: .a20to25, work: [.ip], housing: [.renter], drives: true
            )
            model.profile.onboardingCompleted = true
            model.counter.number = 268
        }
        if let theme { model.settings.themeChoice = theme }
        if let autoDusk { model.settings.autoDusk = autoDusk }
        #endif
    }
}
