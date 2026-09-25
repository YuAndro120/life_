import SwiftUI

struct FiltersView: View {
    @Environment(\.theme) private var theme
    @Environment(AppModel.self) private var model
    @State private var showTopicPicker = false
    @State private var editingTime: TimeSlot?

    enum TimeSlot: Identifiable { case morning, evening; var id: Self { self } }

    private var isSage: Bool { theme.id == .sage }

    var body: some View {
        @Bindable var settings = model.settings
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                if isSage {
                    Text("Хранятся на телефоне").font(theme.fonts.body(13)).foregroundStyle(theme.muted).padding(.top, 14)
                    ScreenTitle(title: "Фильтры", subtitle: "что попадает в выпуск").padding(.top, 8)
                } else {
                    HStack { Text("Настройки выпуска"); Spacer(); Text("Хранятся на телефоне") }
                        .metaStyle().padding(.top, 14)
                    ScreenTitle(title: "Фильтры").padding(.top, 14)
                }

                section("01", "Тон") {
                    SwitchRow(title: "Спокойный режим", hint: "Нейтральные заголовки, без «срочно»", isOn: $settings.calmMode)
                }

                section("02", "Тип информации") {
                    ForEach(InfoType.allCases, id: \.self) { type in
                        SwitchRow(title: Self.infoTitle(type), hint: Self.infoHint(type), isOn: infoBinding(type))
                        if type != InfoType.allCases.last && !isSage { Rule() }
                    }
                }

                section("03", "Тяжёлые темы") {
                    PillSegments(
                        options: [(HeavyMode.hide, "Скрывать"), (.fold, "Сворачивать"), (.show, "Показывать")],
                        selection: heavyBinding
                    )
                    .padding(.top, isSage ? 0 : 16)
                    HStack {
                        Text("Не больше в выпуске").font(theme.fonts.body(17, .medium)).tracking(-0.17).foregroundStyle(theme.ink)
                        Spacer()
                        stepper
                    }
                    .frame(minHeight: 64)
                    .opacity(model.settings.preferences.heavyMode == .show ? 1 : 0.45)
                    .accessibilityElement(children: .contain)
                }

                section("04", "Стоп-темы") {
                    FlowLayout(spacing: 8) {
                        ForEach(model.settings.preferences.stopTopics.sorted { $0.title < $1.title }, id: \.self) { topic in
                            Button { toggleStopTopic(topic) } label: {
                                HStack(spacing: 10) {
                                    Text(topic.title)
                                    Text("×").foregroundStyle(theme.muted)
                                }
                                .font(theme.fonts.body(15, .medium)).foregroundStyle(theme.ink)
                                .padding(.leading, 14).padding(.trailing, 12).frame(minHeight: 44)
                                .background(Capsule().fill(theme.chipBg))
                                .overlay(Capsule().strokeBorder(theme.chipBorder, lineWidth: 1))
                            }
                            .buttonStyle(.plain)
                            .accessibilityLabel("Убрать стоп-тему \(topic.title)")
                        }
                        Button { showTopicPicker = true } label: {
                            Text("+ Добавить").font(theme.fonts.body(15, .medium)).foregroundStyle(theme.accent)
                                .padding(.horizontal, 14).frame(minHeight: 44)
                                .overlay(Capsule().strokeBorder(theme.muted, style: StrokeStyle(lineWidth: 1, dash: [3, 3])))
                        }
                        .buttonStyle(.plain)
                    }
                    .padding(.top, isSage ? 0 : 16)
                }

                section("05", "Реклама") {
                    SwitchRow(title: "Скрывать рекламные посты", hint: "По маркировке «Реклама» и erid", isOn: $settings.hideAds)
                }

                section("06", "Расписание") {
                    PillSegments(
                        options: [(SchedulePreference.both, "Утро и вечер"), (.am, "Только утро"), (.pm, "Только вечер")],
                        selection: scheduleBinding, fontSize: 13
                    )
                    .padding(.top, isSage ? 0 : 16)
                    if settings.schedulePreference != .pm { timeRow("Утренний выпуск", settings.morningMinutes, .morning) }
                    if settings.schedulePreference != .am { timeRow("Вечерний выпуск", settings.eveningMinutes, .evening) }
                    urgentRow
                }
            }
            .padding(.horizontal, 20)
            .padding(.bottom, 32)
        }
        .scrollIndicators(.hidden)
        .screenBackground(theme)
        .onChange(of: settings.preferences) { model.save() }
        .sheet(isPresented: $showTopicPicker) { TopicPickerSheet().environment(\.theme, theme).environment(model) }
        .sheet(item: $editingTime) { slot in
            TimePickerSheet(
                title: slot == .morning ? "Утренний выпуск" : "Вечерний выпуск",
                minutes: slot == .morning ? $settings.morningMinutes : $settings.eveningMinutes
            ) { model.scheduleChanged() }
            .environment(\.theme, theme)
        }
    }

    // MARK: секции

    @ViewBuilder private func section<Content: View>(_ index: String, _ title: String, @ViewBuilder _ content: () -> Content) -> some View {
        SectionTitle(index: index, title: title).padding(.top, isSage ? 30 : 36).padding(.bottom, isSage ? 10 : 0)
        if isSage {
            VStack(alignment: .leading, spacing: 0) { content() }
                .padding(.horizontal, 16).padding(.vertical, 4)
                .background(RoundedRectangle(cornerRadius: theme.cardRadius, style: .continuous).fill(theme.card))
        } else {
            VStack(alignment: .leading, spacing: 0) { content() }
        }
    }

    private func timeRow(_ title: String, _ minutes: Int, _ slot: TimeSlot) -> some View {
        VStack(spacing: 0) {
            Button { editingTime = slot } label: {
                HStack {
                    Text(title).font(theme.fonts.body(17, .medium)).tracking(-0.17).foregroundStyle(theme.ink)
                    Spacer()
                    Text(RuFormat.clock(minutes: minutes))
                        .font(theme.fonts.heading(22, .semibold)).tracking(-0.66).monospacedDigit().foregroundStyle(theme.ink)
                }
                .frame(minHeight: 60)
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            if !isSage { Rule() }
        }
    }

    private var urgentRow: some View {
        VStack(spacing: 0) {
            HStack(spacing: 12) {
                VStack(alignment: .leading, spacing: 3) {
                    Text("Срочные уведомления").font(theme.fonts.body(17, .medium)).tracking(-0.17).foregroundStyle(theme.ink)
                    Text("Появятся вместе с push-уведомлениями").font(theme.fonts.body(13)).foregroundStyle(theme.muted)
                }
                Spacer()
                Text("Ключевые слова →").metaStyle(color: theme.accent)
            }
            .frame(minHeight: 60)
            .opacity(0.4)
            .accessibilityElement(children: .combine)
            .accessibilityAddTraits(.isButton)
            .accessibilityHint("Пока недоступно")
        }
    }

    // MARK: привязки

    private var stepper: some View {
        let value = model.settings.maxHeavy
        return HStack(spacing: 0) {
            stepButton("−", "Меньше") { setMaxHeavy(value - 1) }
            Text("\(value)").font(theme.fonts.heading(22, .semibold)).monospacedDigit().foregroundStyle(theme.ink).frame(width: 44)
            stepButton("+", "Больше") { setMaxHeavy(value + 1) }
        }
    }

    private func stepButton(_ symbol: String, _ label: String, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Text(symbol).font(theme.fonts.body(20)).foregroundStyle(theme.ink)
                .frame(width: 44, height: 44)
                .overlay(Circle().strokeBorder(theme.chipBorder == .clear ? theme.line : theme.chipBorder, lineWidth: 1))
        }
        .buttonStyle(.plain)
        .accessibilityLabel(label)
    }

    private func setMaxHeavy(_ n: Int) {
        var p = model.settings.preferences
        p.maxHeavy = min(10, max(0, n))
        model.settings.preferences = p
    }

    private func infoBinding(_ type: InfoType) -> Binding<Bool> {
        Binding(
            get: { model.settings.preferences.infoTypes.contains(type) },
            set: { on in
                var p = model.settings.preferences
                if on { p.infoTypes.insert(type) } else { p.infoTypes.remove(type) }
                model.settings.preferences = p
            }
        )
    }

    private var heavyBinding: Binding<HeavyMode> {
        Binding(
            get: { model.settings.preferences.heavyMode },
            set: { mode in
                var p = model.settings.preferences
                p.heavyMode = mode
                model.settings.preferences = p
            }
        )
    }

    private var scheduleBinding: Binding<SchedulePreference> {
        Binding(
            get: { model.settings.schedulePreference },
            set: { model.settings.schedulePreference = $0; model.scheduleChanged() }
        )
    }

    private func toggleStopTopic(_ topic: Topic) {
        var p = model.settings.preferences
        p.stopTopics.remove(topic)
        model.settings.preferences = p
    }

    static func infoTitle(_ t: InfoType) -> String {
        switch t {
        case .fact: "Факты"
        case .official: "Официальные решения"
        case .opinion: "Мнения экспертов"
        case .forecast: "Прогнозы"
        case .rumor: "Неподтверждённое"
        }
    }

    static func infoHint(_ t: InfoType) -> String {
        switch t {
        case .fact: "Что произошло, без оценок"
        case .official: "Законы, постановления, ведомства"
        case .opinion: "Оценки и комментарии"
        case .forecast: "«Что будет, если…»"
        case .rumor: "«По данным источников…»"
        }
    }
}

// MARK: - листы

private struct TopicPickerSheet: View {
    @Environment(\.theme) private var theme
    @Environment(\.dismiss) private var dismiss
    @Environment(AppModel.self) private var model

    var body: some View {
        let stopped = model.settings.preferences.stopTopics
        VStack(alignment: .leading, spacing: 0) {
            HStack {
                Text("Добавить стоп-тему").font(theme.fonts.heading(22, .semibold)).foregroundStyle(theme.ink)
                Spacer()
                Button("Готово") { dismiss() }.font(theme.fonts.body(16, .medium)).foregroundStyle(theme.accent).frame(minHeight: 44)
            }
            .padding(.top, 20)
            ScrollView {
                VStack(spacing: 0) {
                    ForEach(Topic.allCases, id: \.self) { topic in
                        Button {
                            var p = model.settings.preferences
                            if stopped.contains(topic) { p.stopTopics.remove(topic) } else { p.stopTopics.insert(topic) }
                            model.settings.preferences = p
                        } label: {
                            HStack {
                                Text(topic.title).font(theme.fonts.body(17)).foregroundStyle(theme.ink)
                                Spacer()
                                if stopped.contains(topic) { Image(systemName: "checkmark").foregroundStyle(theme.accent) }
                            }
                            .frame(minHeight: 48).contentShape(Rectangle())
                        }
                        .buttonStyle(.plain)
                        Rule()
                    }
                }
            }
        }
        .padding(.horizontal, 20)
        .background(theme.bg.ignoresSafeArea())
        .presentationDetents([.large])
    }
}

struct TimePickerSheet: View {
    @Environment(\.theme) private var theme
    @Environment(\.dismiss) private var dismiss
    let title: String
    @Binding var minutes: Int
    var onDone: () -> Void

    var body: some View {
        VStack(spacing: 8) {
            HStack {
                Text(title).font(theme.fonts.heading(22, .semibold)).foregroundStyle(theme.ink)
                Spacer()
                Button("Готово") { onDone(); dismiss() }.font(theme.fonts.body(16, .medium)).foregroundStyle(theme.accent).frame(minHeight: 44)
            }
            .padding(.top, 20)
            DatePicker("Время", selection: dateBinding, displayedComponents: .hourAndMinute)
                .datePickerStyle(.wheel).labelsHidden()
                .environment(\.locale, Locale(identifier: "ru_RU"))
            Spacer()
        }
        .padding(.horizontal, 20)
        .background(theme.bg.ignoresSafeArea())
        .presentationDetents([.height(340)])
    }

    private var dateBinding: Binding<Date> {
        Binding(
            get: { Calendar.current.date(bySettingHour: minutes / 60, minute: minutes % 60, second: 0, of: .now) ?? .now },
            set: { d in
                let c = Calendar.current.dateComponents([.hour, .minute], from: d)
                minutes = (c.hour ?? 0) * 60 + (c.minute ?? 0)
            }
        )
    }
}

/// Простая раскладка чипов с переносом строк.
struct FlowLayout: Layout {
    var spacing: CGFloat = 8

    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        let maxWidth = proposal.width ?? .infinity
        var x: CGFloat = 0, y: CGFloat = 0, rowHeight: CGFloat = 0, width: CGFloat = 0
        for v in subviews {
            let s = v.sizeThatFits(.unspecified)
            if x + s.width > maxWidth, x > 0 { x = 0; y += rowHeight + spacing; rowHeight = 0 }
            x += s.width + spacing
            rowHeight = max(rowHeight, s.height)
            width = max(width, x - spacing)
        }
        return CGSize(width: width, height: y + rowHeight)
    }

    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        var x = bounds.minX, y = bounds.minY, rowHeight: CGFloat = 0
        for v in subviews {
            let s = v.sizeThatFits(.unspecified)
            if x + s.width > bounds.maxX, x > bounds.minX { x = bounds.minX; y += rowHeight + spacing; rowHeight = 0 }
            v.place(at: CGPoint(x: x, y: y), proposal: ProposedViewSize(s))
            x += s.width + spacing
            rowHeight = max(rowHeight, s.height)
        }
    }
}
