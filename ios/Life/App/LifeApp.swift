import SwiftData
import SwiftUI

@main
struct LifeApp: App {
    private let container: ModelContainer
    @State private var model: AppModel

    init() {
        do {
            let container = try PersistenceStack.makeContainer(inMemory: DebugLaunch.inMemory)
            self.container = container
            let model = AppModel(
                context: container.mainContext,
                notifier: DebugLaunch.inMemory ? NoopNotificationScheduler() : SystemNotificationScheduler(),
                now: DebugLaunch.fixedNow ?? .now
            )
            DebugLaunch.apply(to: model)
            _model = State(initialValue: model)
        } catch {
            fatalError("Не удалось открыть хранилище: \(error)")
        }
    }

    var body: some Scene {
        WindowGroup {
            RootView().environment(model)
        }
        .modelContainer(container)
    }
}
