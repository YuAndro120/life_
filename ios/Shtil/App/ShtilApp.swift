import SwiftData
import SwiftUI

@main
struct ShtilApp: App {
    private let container: ModelContainer
    @State private var model: AppModel

    init() {
        do {
            let container = try PersistenceStack.makeContainer(inMemory: DebugLaunch.inMemory)
            self.container = container
            let model = AppModel(
                context: container.mainContext,
                source: Self.makeSource(),
                fallback: Self.makeFallback(),
                notifier: DebugLaunch.inMemory ? NoopNotificationScheduler() : SystemNotificationScheduler(),
                now: DebugLaunch.fixedNow ?? .now
            )
            DebugLaunch.apply(to: model)
            _model = State(initialValue: model)
        } catch {
            fatalError("Не удалось открыть хранилище: \(error)")
        }
    }

    private static func makeSource() -> any ContentSource {
        if DebugLaunch.useFixtures { return FixtureContentSource() }
        guard let url = AppConfig.apiBaseURL else { return FixtureContentSource() }
        return APIClient(baseURL: url)
    }

    /// Только для разработки: без сервера и кэша показываем встроенные данные, а не пустой экран.
    private static func makeFallback() -> (any ContentSource)? {
        #if DEBUG
        return FixtureContentSource()
        #else
        return nil
        #endif
    }

    var body: some Scene {
        WindowGroup {
            RootView().environment(model)
        }
        .modelContainer(container)
    }
}
