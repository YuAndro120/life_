import SwiftUI

enum AppTab: String, CaseIterable, Identifiable {
    case today, calendar, filters, profile
    var id: String { rawValue }

    var title: String {
        switch self {
        case .today: "Сегодня"
        case .calendar: "Календарь"
        case .filters: "Фильтры"
        case .profile: "Профиль"
        }
    }
}

/// Бумага/Сумерки: текстовый с линией сверху у активной вкладки. Шалфей: плавающая капсула.
struct LifeTabBar: View {
    @Environment(\.theme) private var theme
    @Binding var selection: AppTab

    var body: some View {
        if theme.id == .sage { floating } else { flat }
    }

    private var flat: some View {
        VStack(spacing: 0) {
            Rule()
            HStack(spacing: 8) {
                ForEach(AppTab.allCases) { tab in
                    let active = tab == selection
                    Button { selection = tab } label: {
                        VStack(spacing: 0) {
                            Rectangle().fill(active ? theme.ink : .clear).frame(height: 2)
                            Text(tab.title)
                                .font(theme.fonts.body(13, active ? .semibold : .medium))
                                .foregroundStyle(active ? theme.ink : theme.muted)
                                .padding(.top, 10)
                                .frame(maxWidth: .infinity, minHeight: 48, alignment: .topLeading)
                        }
                        .contentShape(Rectangle())
                    }
                    .buttonStyle(.plain)
                    .accessibilityAddTraits(active ? [.isButton, .isSelected] : .isButton)
                }
            }
            .padding(.horizontal, 20)
        }
        .background(theme.bg)
    }

    private var floating: some View {
        HStack(spacing: 4) {
            ForEach(AppTab.allCases) { tab in
                let active = tab == selection
                Button { selection = tab } label: {
                    Text(tab.title)
                        .font(theme.fonts.body(13, active ? .semibold : .medium))
                        .foregroundStyle(active ? theme.accentDeep : theme.muted)
                        .frame(maxWidth: .infinity, minHeight: 48)
                        .background(RoundedRectangle(cornerRadius: 22, style: .continuous).fill(active ? theme.plate : .clear))
                        .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
                .accessibilityAddTraits(active ? [.isButton, .isSelected] : .isButton)
            }
        }
        .padding(6)
        .background(
            RoundedRectangle(cornerRadius: 28, style: .continuous).fill(theme.card)
                .shadow(color: theme.ink.opacity(0.08), radius: 10, y: 6)
        )
        .padding(.horizontal, 12)
        .padding(.bottom, 8)
    }
}
