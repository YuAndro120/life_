import SwiftUI

struct CalendarView: View {
    @Environment(\.theme) private var theme
    @Environment(AppModel.self) private var model
    @State private var onlyMine = true

    private var isSage: Bool { theme.id == .sage }

    var body: some View {
        let laws = model.calendarLaws(onlyMine: onlyMine)
        let tags = AudienceMatcher.audienceTags(for: model.profile.snapshot)
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                header
                PillSegments(options: [(true, "Про меня"), (false, "Все изменения")], selection: $onlyMine)
                    .padding(.top, 22)
                    .padding(.bottom, 8)

                if laws.isEmpty {
                    Text(onlyMine ? "Пока ничего про тебя." : "Пока нет подписанных изменений.")
                        .font(theme.fonts.body(16)).foregroundStyle(theme.muted).padding(.vertical, 32)
                } else if isSage {
                    sageList(laws, tags)
                } else {
                    paperList(laws, tags)
                }

                Text(isSage
                     ? "Здесь только подписанные законы. Напоминаем за неделю до вступления в силу."
                     : "Только подписанные законы. Напоминаем за 7 дней до вступления в силу.")
                    .font(isSage ? theme.fonts.body(13) : ThemeFonts.mono(11))
                    .lineSpacing(4).foregroundStyle(theme.muted)
                    .padding(.top, 24)
            }
            .padding(.horizontal, 20)
            .padding(.bottom, 32)
        }
        .scrollIndicators(.hidden)
        .screenBackground(theme)
    }

    private var header: some View {
        VStack(alignment: .leading, spacing: 14) {
            if isSage {
                Text("Сегодня, \(RuFormat.dayMonth(model.today, today: model.today))")
                    .font(theme.fonts.body(13)).foregroundStyle(theme.muted).padding(.top, 14)
                ScreenTitle(title: "Что меняется", subtitle: "для тебя")
            } else {
                HStack {
                    Text("Календарь")
                    Spacer()
                    Text("Сегодня \(RuFormat.numeric(model.today))")
                }
                .metaStyle().padding(.top, 14)
                ScreenTitle(title: "Что меняется")
            }
        }
    }

    // MARK: Бумага / Сумерки

    private func paperList(_ laws: [Law], _ tags: Set<String>) -> some View {
        VStack(spacing: 0) {
            Rule(strong: true)
            ForEach(Array(laws.enumerated()), id: \.element.id) { i, law in
                let eff = law.dates.effective
                let showsDate = i == 0 || laws[i - 1].dates.effective != eff
                CalendarRow(law: law, tags: tags, today: model.today, showsDate: showsDate)
                Rule()
            }
        }
    }

    // MARK: Шалфей

    private func sageList(_ laws: [Law], _ tags: Set<String>) -> some View {
        let groups = Dictionary(grouping: laws) { $0.dates.effective ?? model.today }
            .sorted { $0.key < $1.key }
        return VStack(alignment: .leading, spacing: 22) {
            ForEach(groups, id: \.key) { date, items in
                VStack(alignment: .leading, spacing: 8) {
                    HStack(alignment: .firstTextBaseline) {
                        Text(RuFormat.dayMonth(date, today: model.today)).font(theme.fonts.heading(22, .medium)).foregroundStyle(theme.ink)
                        Spacer()
                        Text(RuFormat.countdownWords(days: date.days(from: model.today))).font(theme.fonts.body(13)).foregroundStyle(theme.muted)
                    }
                    ForEach(items) { law in
                        CalendarRow(law: law, tags: tags, today: model.today, showsDate: false)
                            .padding(.horizontal, 16)
                            .background(RoundedRectangle(cornerRadius: theme.cardRadius, style: .continuous).fill(theme.card))
                    }
                }
            }
        }
        .padding(.top, 8)
    }
}

private struct CalendarRow: View {
    @Environment(\.theme) private var theme
    @Environment(AppModel.self) private var model
    let law: Law
    let tags: Set<String>
    let today: CalendarDate
    let showsDate: Bool

    private var isSage: Bool { theme.id == .sage }
    private var days: Int { law.dates.effective.map { $0.days(from: today) } ?? 0 }

    var body: some View {
        HStack(alignment: .top, spacing: 12) {
            if !isSage { tile }
            VStack(alignment: .leading, spacing: 6) {
                let audience = AudienceLabels.label(for: law, profileTags: tags)
                if isSage {
                    Text(audience).font(theme.fonts.body(13)).foregroundStyle(theme.accentDeep)
                } else {
                    (Text("\(audience) · ") + Text(RuFormat.countdownShort(days: days)).foregroundColor(theme.accent))
                        .metaStyle()
                }
                NavigationLink(value: LawRoute(id: law.id)) {
                    Text(law.title)
                        .font(theme.fonts.body(isSage ? 16 : 17, .medium))
                        .foregroundStyle(theme.ink).multilineTextAlignment(.leading)
                        .fixedSize(horizontal: false, vertical: true)
                }
                .buttonStyle(.plain)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            bell
        }
        .padding(.vertical, 14)
    }

    private var tile: some View {
        VStack(alignment: .leading, spacing: 2) {
            if showsDate, let eff = law.dates.effective {
                let t = RuFormat.calendarTile(eff, today: today)
                Text(t.day).font(theme.fonts.heading(28, .semibold)).tracking(-1).monospacedDigit().foregroundStyle(theme.ink)
                Text(t.month).metaStyle()
            }
        }
        .frame(width: 60, alignment: .leading)
    }

    @ViewBuilder private var bell: some View {
        let on = model.hasReminder(law)
        let available = on || model.reminderPlan(for: law) != nil
        Button { model.toggleReminder(law) } label: {
            Image(systemName: on ? "bell.fill" : "bell")
                .font(.system(size: 18, weight: .regular))
                .foregroundStyle(on ? theme.accent : theme.muted)
                .frame(width: 44, height: 44)
                .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .disabled(!available)
        .opacity(available ? 1 : 0.3)
        .accessibilityLabel(on ? "Напоминание включено" : "Напомнить")
        .accessibilityAddTraits(on ? [.isButton, .isSelected] : .isButton)
    }
}
