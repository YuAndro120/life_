import Foundation

/// Отладочные аргументы запуска для проверки экранов на симуляторе (только DEBUG):
/// `-lifeInMemory YES -lifeSource fixtures -lifeAPI http://192.168.0.5:8080 -lifeSeed sample -lifeTheme sage -lifeTab calendar -lifeLaw lw_01`.
enum DebugLaunch {
    #if DEBUG
    private static var defaults: UserDefaults { .standard }
    static var inMemory: Bool { defaults.bool(forKey: "lifeInMemory") }
    static var seedSample: Bool { defaults.string(forKey: "lifeSeed") == "sample" }
    static var theme: ThemeChoice? { defaults.string(forKey: "lifeTheme").flatMap(ThemeChoice.init(rawValue:)) }
    static var tab: String? { defaults.string(forKey: "lifeTab") }
    static var lawId: String? { defaults.string(forKey: "lifeLaw") }
    static var apiOverride: String? { defaults.string(forKey: "lifeAPI") }
    static var useFixtures: Bool { defaults.string(forKey: "lifeSource") == "fixtures" }
    static var onboardingStep: Int? { defaults.string(forKey: "lifeOnbStep").flatMap { Int($0) } }
    static var autoDusk: Bool? { defaults.object(forKey: "lifeAutoDusk") as? Bool }
    static var fixedNow: Date? {
        defaults.string(forKey: "lifeNow").flatMap { ISO8601DateFormatter().date(from: $0) }
    }
    #else
    static var inMemory: Bool { false }
    static var seedSample: Bool { false }
    static var theme: ThemeChoice? { nil }
    static var tab: String? { nil }
    static var lawId: String? { nil }
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
