import SwiftUI

struct Rule: View {
    @Environment(\.theme) private var theme
    var strong = false
    var body: some View {
        Rectangle().fill(strong ? theme.ink : theme.line).frame(height: 1)
    }
}

/// Крупный заголовок экрана. Бумага/Сумерки: жирный Onest с акцентной точкой;
/// Шалфей: Spectral, вторая строка курсивом.
struct ScreenTitle: View {
    @Environment(\.theme) private var theme
    let title: String
    var subtitle: String?
    var size: CGFloat = 48
    var mark = "."

    var body: some View {
        if theme.id == .sage {
            VStack(alignment: .leading, spacing: 0) {
                Text(title).font(theme.fonts.heading(min(size, 40), .medium)).tracking(-0.8)
                if let subtitle {
                    Text(subtitle).font(theme.fonts.heading(min(size, 40), .regular, italic: true))
                        .foregroundStyle(theme.muted).lineLimit(1).minimumScaleFactor(0.7)
                }
            }
            .foregroundStyle(theme.ink)
            .lineSpacing(-8)
            .frame(maxWidth: .infinity, alignment: .leading)
            .accessibilityElement(children: .combine)
        } else {
            (Text(title) + Text(mark).foregroundStyle(theme.accent))
                .font(theme.fonts.heading(size, .bold))
                .tracking(-size * 0.045)
                .foregroundStyle(theme.ink)
                .lineSpacing(-size * 0.05)
                .frame(maxWidth: .infinity, alignment: .leading)
                .accessibilityAddTraits(.isHeader)
        }
    }
}

/// «01 — Тон» с жирной линией (Бумага/Сумерки) или зелёный заголовок (Шалфей).
struct SectionTitle: View {
    @Environment(\.theme) private var theme
    let index: String?
    let title: String

    var body: some View {
        if theme.id == .sage {
            Text(title)
                .font(theme.fonts.body(15, .semibold))
                .foregroundStyle(theme.accentDeep)
                .padding(.horizontal, 4)
                .frame(maxWidth: .infinity, alignment: .leading)
                .accessibilityAddTraits(.isHeader)
        } else {
            VStack(alignment: .leading, spacing: 10) {
                Text(index.map { "\($0) — \(title)" } ?? title)
                    .metaStyle(color: theme.ink)
                    .accessibilityAddTraits(.isHeader)
                Rule(strong: true)
            }
        }
    }
}

struct ThemedSwitch: View {
    @Environment(\.theme) private var theme
    @Binding var isOn: Bool
    let label: String
    var disabled = false

    var body: some View {
        Button {
            isOn.toggle()
        } label: {
            track
                .frame(width: theme.id == .sage ? 56 : 52, height: 44, alignment: .trailing)
                .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .disabled(disabled)
        .opacity(disabled ? 0.4 : 1)
        .accessibilityLabel(label)
        .accessibilityValue(isOn ? "включено" : "выключено")
        .accessibilityAddTraits(.isToggle)
    }

    @ViewBuilder private var track: some View {
        if theme.id == .sage {
            Capsule().fill(isOn ? theme.accent : theme.line)
                .frame(width: 50, height: 30)
                .overlay(alignment: isOn ? .trailing : .leading) {
                    Circle().fill(theme.card).frame(width: 26, height: 26).padding(2)
                }
                .animation(.easeOut(duration: 0.2), value: isOn)
        } else {
            Capsule().fill(isOn ? theme.ink : .clear)
                .overlay(Capsule().strokeBorder(theme.ink, lineWidth: 1.5))
                .frame(width: 42, height: 24)
                .overlay(alignment: isOn ? .trailing : .leading) {
                    Circle().fill(isOn ? theme.bg : theme.ink).frame(width: 17, height: 17).padding(.horizontal, 3.5)
                }
                .animation(.easeOut(duration: 0.18), value: isOn)
        }
    }
}

/// Строка «название + подсказка + переключатель».
struct SwitchRow: View {
    @Environment(\.theme) private var theme
    let title: String
    var hint: String?
    @Binding var isOn: Bool
    var disabled = false

    var body: some View {
        HStack(spacing: 12) {
            VStack(alignment: .leading, spacing: 3) {
                Text(title).font(theme.fonts.body(17, .medium)).tracking(-0.17).foregroundStyle(theme.ink)
                if let hint {
                    Text(hint).font(theme.fonts.body(13)).foregroundStyle(theme.muted)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            ThemedSwitch(isOn: $isOn, label: title, disabled: disabled)
        }
        .frame(minHeight: 64)
    }
}

/// Капсула-чип: выбор в онбординге, стоп-темы.
struct ChipButton: View {
    @Environment(\.theme) private var theme
    let title: String
    let isOn: Bool
    var showsRemove = false
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            HStack(spacing: 8) {
                Text(title)
                if showsRemove && isOn {
                    Text("×").foregroundStyle(isOn ? theme.bg.opacity(0.7) : theme.muted)
                }
            }
            .font(theme.fonts.body(15, .medium))
            .padding(.horizontal, 16)
            .frame(minHeight: 44)
            .foregroundStyle(isOn ? onText : theme.ink)
            .background(Capsule().fill(isOn ? onFill : theme.chipBg))
            .overlay(Capsule().strokeBorder(isOn ? onFill : theme.chipBorder, lineWidth: 1))
        }
        .buttonStyle(.plain)
        .accessibilityAddTraits(isOn ? [.isButton, .isSelected] : .isButton)
    }

    private var onFill: Color { theme.id == .sage ? theme.accent : theme.ink }
    private var onText: Color { theme.id == .sage ? theme.card : theme.bg }
}

/// Сегментированный переключатель-капсула (тяжёлые темы, расписание, «Про меня / Все»).
struct PillSegments<Value: Hashable>: View {
    @Environment(\.theme) private var theme
    let options: [(value: Value, title: String)]
    @Binding var selection: Value
    var fontSize: CGFloat = 14

    var body: some View {
        HStack(spacing: 4) {
            ForEach(options.indices, id: \.self) { i in
                let option = options[i]
                let selected = option.value == selection
                Button {
                    selection = option.value
                } label: {
                    Text(option.title)
                        .font(theme.fonts.body(fontSize, selected && theme.id == .sage ? .semibold : .medium))
                        .foregroundStyle(selected ? selectedText : theme.ink)
                        .frame(maxWidth: .infinity, minHeight: 42)
                        .background(Capsule().fill(selected ? selectedFill : .clear))
                }
                .buttonStyle(.plain)
                .accessibilityAddTraits(selected ? [.isButton, .isSelected] : .isButton)
            }
        }
        .padding(4)
        .background(Capsule().fill(theme.segmentBg))
    }

    private var selectedFill: Color { theme.id == .sage ? theme.accent : theme.ink }
    private var selectedText: Color { theme.id == .sage ? theme.card : theme.bg }
}

/// Главная кнопка: высота 58, радиус 18 (Шалфей — капсула).
struct PrimaryButton: View {
    @Environment(\.theme) private var theme
    let title: String
    var trailing: String?
    var systemImage: String?
    var centered = false
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            HStack {
                if let systemImage { Image(systemName: systemImage) }
                Text(title)
                if !centered { Spacer(minLength: 8) }
                if let trailing { Text(trailing) }
            }
            .font(theme.fonts.body(16, .medium))
            .foregroundStyle(theme.buttonText)
            .padding(.horizontal, 20)
            .frame(maxWidth: .infinity, minHeight: theme.buttonHeight)
            .frame(maxWidth: .infinity)
            .background(
                RoundedRectangle(cornerRadius: theme.id == .sage ? theme.buttonHeight / 2 : theme.buttonRadius, style: .continuous)
                    .fill(theme.buttonBg)
            )
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
    }
}

extension View {
    /// Фон экрана на весь размер.
    func screenBackground(_ theme: Theme) -> some View {
        background(theme.bg.ignoresSafeArea())
    }
}
