import SwiftUI
import UIKit

struct RootView: View {
    @Environment(AppModel.self) private var model

    var body: some View {
        Group {
            if model.onboardingCompleted {
                MainTabs()
            } else {
                OnboardingFlow()
            }
        }
        .environment(\.theme, model.theme)
        .preferredColorScheme(model.theme.colorScheme)
        .tint(model.theme.accent)
        .background(model.theme.bg.ignoresSafeArea())
        .task { await model.refresh() }
        .task {
            while !Task.isCancelled {
                try? await Task.sleep(for: .seconds(60))
                model.tick()
            }
        }
    }
}

struct MainTabs: View {
    @Environment(\.theme) private var theme
    @Environment(AppModel.self) private var model
    @State private var tab: AppTab = DebugLaunch.tab.flatMap(AppTab.init(rawValue:)) ?? .today
    @State private var path: [LawRoute] = DebugLaunch.lawId.map { [LawRoute(id: $0)] } ?? []

    var body: some View {
        NavigationStack(path: $path) {
            Group {
                switch tab {
                case .today: TodayView(openFilters: { tab = .filters }, openCalendar: { tab = .calendar })
                case .calendar: CalendarView()
                case .filters: FiltersView()
                case .profile: ProfileView()
                }
            }
            .safeAreaInset(edge: .bottom, spacing: 0) { LifeTabBar(selection: $tab) }
            .toolbar(.hidden, for: .navigationBar)
            .navigationDestination(for: LawRoute.self) { route in
                if let law = model.law(id: route.id) {
                    LawDetailView(law: law)
                } else {
                    Text("Закон не найден").foregroundStyle(theme.muted)
                }
            }
        }
    }
}

/// Возвращает жест «назад» свайпом при скрытой навигационной панели.
extension UINavigationController: @retroactive UIGestureRecognizerDelegate {
    override open func viewDidLoad() {
        super.viewDidLoad()
        interactivePopGestureRecognizer?.delegate = self
    }

    public func gestureRecognizerShouldBegin(_ gestureRecognizer: UIGestureRecognizer) -> Bool {
        viewControllers.count > 1
    }
}
