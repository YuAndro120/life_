import SwiftUI

/// Выбор темы с живой перекраской + «Сумерки по вечерам».
struct ThemePicker: View {
    @Environment(\.theme) private var theme
    @Environment(AppModel.self) private var model

    var body: some View {
        @Bindable var settings = model.settings
        VStack(alignment: .leading, spacing: 18) {
            HStack(alignment: .top, spacing: 8) {
                ForEach(ThemeChoice.allCases, id: \.self) { choice in
                    let selected = settings.themeChoice == choice
                    Button {
                        withAnimation(.easeOut(duration: 0.25)) { settings.themeChoice = choice }
                        model.save()
                    } label: {
                        VStack(alignment: .leading, spacing: 8) {
                            ThemePreview(theme: .of(choice))
                            Text(choice.title).font(theme.fonts.body(15, .semibold)).foregroundStyle(theme.ink).padding(.horizontal, 5)
                            Text(choice.subtitle).font(theme.fonts.body(12)).lineSpacing(2).foregroundStyle(theme.muted)
                                .multilineTextAlignment(.leading).padding(.horizontal, 5)
                                .fixedSize(horizontal: false, vertical: true)
                        }
                        .frame(maxWidth: .infinity, alignment: .topLeading)
                        .padding(.horizontal, 5).padding(.top, 5).padding(.bottom, 10)
                        .overlay(RoundedRectangle(cornerRadius: 22, style: .continuous).strokeBorder(selected ? theme.accent : .clear, lineWidth: 2))
                        .contentShape(Rectangle())
                    }
                    .buttonStyle(.plain)
                    .accessibilityElement(children: .combine)
                    .accessibilityAddTraits(selected ? [.isButton, .isSelected] : .isButton)
                }
            }
            SwitchRow(title: "Сумерки по вечерам", hint: "Тёмная тема сама включится после 19:00", isOn: Binding(
                get: { model.settings.autoDusk },
                set: { model.settings.autoDusk = $0; model.save() }
            ))
        }
    }
}

struct ThemePreview: View {
    let theme: Theme

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            bar(0.46, 9, theme.ink, 1)
            bar(0.72, 5, theme.muted, 0.6)
            HStack(spacing: 5) {
                Circle().fill(theme.accent).frame(width: 6, height: 6)
                Capsule().fill(theme.ink.opacity(0.55)).frame(height: 4)
            }
            .padding(.horizontal, 7).frame(height: 30)
            .background(RoundedRectangle(cornerRadius: 7, style: .continuous).fill(theme.card))
            .padding(5)
            .background(RoundedRectangle(cornerRadius: 10, style: .continuous).fill(theme.plate))
            .padding(.top, 4)
            bar(0.88, 5, theme.ink, 0.5).padding(.top, 4)
            bar(0.60, 5, theme.muted, 0.5)
            Spacer(minLength: 0)
        }
        .padding(.horizontal, 10).padding(.vertical, 12)
        .frame(height: 148)
        .background(RoundedRectangle(cornerRadius: 16, style: .continuous).fill(theme.bg))
        .overlay(RoundedRectangle(cornerRadius: 16, style: .continuous).strokeBorder(theme.isDark ? Color(hex: 0x3A3C46) : theme.line, lineWidth: 1))
        .accessibilityHidden(true)
    }

    private func bar(_ fraction: CGFloat, _ height: CGFloat, _ color: Color, _ opacity: Double) -> some View {
        GeometryReader { g in
            Capsule().fill(color.opacity(opacity)).frame(width: g.size.width * fraction, height: height)
        }
        .frame(height: height)
    }
}
