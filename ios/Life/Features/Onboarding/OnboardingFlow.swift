import SwiftUI

struct OnboardingFlow: View {
    @Environment(\.theme) private var theme
    @Environment(AppModel.self) private var model
    @State private var step: Step = DebugLaunch.onboardingStep.flatMap(Step.init(rawValue:)) ?? .welcome

    enum Step: Int { case welcome, profile, calm, theme, building }

    var body: some View {
        ZStack {
            switch step {
            case .welcome: OnbWelcome { go(.profile) }
            case .profile: OnbProfile(back: { go(.welcome) }, next: { go(.calm) }, skip: skip)
            case .calm: OnbCalm(back: { go(.profile) }, next: { go(.theme) })
            case .theme: OnbTheme(back: { go(.calm) }, next: { go(.building) })
            case .building: OnbBuilding()
            }
        }
        .screenBackground(theme)
        .transition(.opacity)
    }

    private func go(_ s: Step) {
        withAnimation(.easeOut(duration: 0.25)) { step = s }
    }

    private func skip() {
        Task { await model.completeOnboarding() }
    }
}

// MARK: - общие элементы

private struct OnbTop: View {
    @Environment(\.theme) private var theme
    let current: Int
    let back: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            HStack {
                Button(action: back) {
                    Text("← Назад").font(theme.fonts.body(15, .medium)).foregroundStyle(theme.ink).frame(minHeight: 44)
                }
                .buttonStyle(.plain)
                Spacer()
                Text("\(RuFormat.two(current)) / 03").metaStyle()
            }
            HStack(spacing: 6) {
                ForEach(1...3, id: \.self) { i in
                    Capsule().fill(i <= current ? (theme.id == .sage ? theme.accent : theme.ink) : theme.line).frame(height: 4)
                }
            }
            .accessibilityHidden(true)
        }
    }
}

private struct OnbScaffold<Content: View, Footer: View>: View {
    @Environment(\.theme) private var theme
    let current: Int
    let back: () -> Void
    @ViewBuilder var content: Content
    @ViewBuilder var footer: Footer

    var body: some View {
        VStack(spacing: 0) {
            ScrollView {
                VStack(alignment: .leading, spacing: 0) {
                    OnbTop(current: current, back: back)
                    content
                }
                .padding(.horizontal, 20).padding(.top, 8).padding(.bottom, 24)
            }
            .scrollIndicators(.hidden)
            VStack(spacing: 4) { footer }
                .padding(.horizontal, 20).padding(.top, 10).padding(.bottom, 12)
        }
    }
}

// MARK: - 0. приветствие

private struct OnbWelcome: View {
    @Environment(\.theme) private var theme
    let start: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack { Text("Выпуск № 1"); Spacer(); Text("Без регистрации") }.metaStyle().padding(.top, 14)
            (Text("Суть") + Text(".").foregroundStyle(theme.accent))
                .font(theme.fonts.heading(96, .bold)).tracking(-96 * 0.04).foregroundStyle(theme.ink)
                .lineLimit(1).minimumScaleFactor(0.6).padding(.top, 36)
            VStack(alignment: .leading, spacing: 0) {
                Text("Новости без шума.").foregroundStyle(theme.ink)
                Text("Законы — только твои.").foregroundStyle(theme.muted)
            }
            .font(theme.id == .sage ? theme.fonts.heading(34, .medium) : theme.fonts.heading(34, .bold))
            .tracking(theme.id == .sage ? -0.5 : -1.5)
            .padding(.top, 20)

            VStack(alignment: .leading, spacing: 0) {
                point("01", "Два выпуска в день, каждый можно дочитать до конца")
                point("02", "Изменения в законах, которые касаются именно тебя")
                point("03", "Настройки живут на телефоне, сервер не знает, кто ты")
            }
            .padding(.top, 32)
            Spacer(minLength: 16)
            PrimaryButton(title: "Начать", trailing: "→", action: start)
            Text("3 шага · около минуты").metaStyle().frame(maxWidth: .infinity).padding(.top, 12)
        }
        .padding(.horizontal, 20).padding(.bottom, 16)
    }

    private func point(_ n: String, _ text: String) -> some View {
        VStack(spacing: 0) {
            Rule(strong: n == "01")
            HStack(alignment: .top, spacing: 16) {
                Text(n).metaStyle(color: theme.accent).padding(.top, 4)
                Text(text).font(theme.fonts.body(16)).lineSpacing(3).foregroundStyle(theme.ink)
                Spacer(minLength: 0)
            }
            .padding(.vertical, 14)
        }
        .accessibilityElement(children: .combine)
    }
}

// MARK: - 1. профиль

private struct OnbProfile: View {
    @Environment(\.theme) private var theme
    let back: () -> Void
    let next: () -> Void
    let skip: () -> Void

    var body: some View {
        OnbScaffold(current: 1, back: back) {
            ScreenTitle(title: "Что про тебя важно знать", size: 40, mark: "?").padding(.top, 22)
            Text("Покажем только те законы и изменения, которые касаются тебя.")
                .font(theme.fonts.body(16)).lineSpacing(3).foregroundStyle(theme.muted).padding(.top, 12)
            ProfileForm().padding(.top, 28)
            HStack(alignment: .top, spacing: 10) {
                Circle().fill(theme.accent).frame(width: 7, height: 7).padding(.top, 6)
                Text("Профиль хранится только на этом телефоне. Сервер не знает, кто ты и что читаешь.")
                    .font(theme.fonts.body(13)).lineSpacing(3).foregroundStyle(theme.muted)
            }
            .padding(.top, 28)
        } footer: {
            PrimaryButton(title: "Дальше", trailing: "→", action: next)
            Button(action: skip) {
                Text("Пропустить").font(theme.fonts.body(15, .medium)).foregroundStyle(theme.muted).frame(maxWidth: .infinity, minHeight: 44)
            }
            .buttonStyle(.plain)
        }
    }
}

// MARK: - 2. что не показывать

private struct OnbCalm: View {
    @Environment(\.theme) private var theme
    @Environment(AppModel.self) private var model
    let back: () -> Void
    let next: () -> Void

    private static let hideable: [Topic] = [.politics, .crime, .incidents, .disasters, .showbiz, .sport, .crypto]

    var body: some View {
        OnbScaffold(current: 2, back: back) {
            ScreenTitle(title: "Что тебе не показывать", size: 40, mark: "?").padding(.top, 22)
            Text("Всё это можно поменять потом в фильтрах.")
                .font(theme.fonts.body(16)).foregroundStyle(theme.muted).padding(.top, 12)

            SectionTitle(index: "01", title: "Скрыть темы").padding(.top, 28).padding(.bottom, 14)
            FlowLayout(spacing: 8) {
                ForEach(Self.hideable, id: \.self) { topic in
                    ChipButton(title: topic.title, isOn: model.settings.preferences.stopTopics.contains(topic), showsRemove: true) {
                        var p = model.settings.preferences
                        if !p.stopTopics.insert(topic).inserted { p.stopTopics.remove(topic) }
                        model.settings.preferences = p
                        model.save()
                    }
                }
            }

            SectionTitle(index: "02", title: "Тяжёлые новости").padding(.top, 32)
            VStack(spacing: 0) {
                mode(.hide, "Скрывать", "Не показывать совсем")
                mode(.fold, "Сворачивать", "Одна нейтральная сводка в конце выпуска")
                mode(.show, "Показывать", "Как обычные сюжеты")
            }

            SectionTitle(index: "03", title: "Тон").padding(.top, 32)
            SwitchRow(title: "Без слухов и прогнозов", hint: "Только факты и официальные решения", isOn: strictBinding)
        } footer: {
            PrimaryButton(title: "Дальше", trailing: "→", action: next)
        }
    }

    private func mode(_ m: HeavyMode, _ title: String, _ hint: String) -> some View {
        let selected = model.settings.preferences.heavyMode == m
        return VStack(spacing: 0) {
            Button {
                var p = model.settings.preferences
                p.heavyMode = m
                model.settings.preferences = p
                model.save()
            } label: {
                HStack(spacing: 14) {
                    Circle().strokeBorder(selected ? theme.accent : theme.muted, lineWidth: selected ? 7 : 1.5).frame(width: 22, height: 22)
                    VStack(alignment: .leading, spacing: 3) {
                        Text(title).font(theme.fonts.body(17, .medium)).foregroundStyle(theme.ink)
                        Text(hint).font(theme.fonts.body(13)).foregroundStyle(theme.muted)
                    }
                    Spacer()
                }
                .frame(minHeight: 64).contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .accessibilityAddTraits(selected ? [.isButton, .isSelected] : .isButton)
            Rule()
        }
    }

    /// «Без слухов и прогнозов» = только факты и официальные решения.
    private var strictBinding: Binding<Bool> {
        Binding(
            get: {
                let t = model.settings.preferences.infoTypes
                return !t.contains(.rumor) && !t.contains(.forecast)
            },
            set: { strict in
                var p = model.settings.preferences
                if strict { p.infoTypes.subtract([.rumor, .forecast]) } else { p.infoTypes.formUnion([.rumor, .forecast]) }
                model.settings.preferences = p
                model.save()
            }
        )
    }
}

// MARK: - 3. тема и расписание

private struct OnbTheme: View {
    @Environment(\.theme) private var theme
    @Environment(AppModel.self) private var model
    let back: () -> Void
    let next: () -> Void

    var body: some View {
        OnbScaffold(current: 3, back: back) {
            ScreenTitle(title: "Как будет выглядеть выпуск", size: 40, mark: "?").padding(.top, 22)
            Text("Выбери тему, экран сразу покажет её. Сменить можно в любой момент.")
                .font(theme.fonts.body(16)).lineSpacing(3).foregroundStyle(theme.muted).padding(.top, 12)
            ThemePicker().padding(.top, 24)
            Text("Когда присылать выпуск").font(theme.fonts.body(15, .medium)).foregroundStyle(theme.ink).padding(.top, 20).padding(.bottom, 10)
            PillSegments(
                options: [(SchedulePreference.both, "Утро и вечер"), (.am, "Только утро"), (.pm, "Только вечер")],
                selection: Binding(
                    get: { model.settings.schedulePreference },
                    set: { model.settings.schedulePreference = $0; model.save() }
                ),
                fontSize: 13
            )
        } footer: {
            PrimaryButton(title: "Собрать первый выпуск", trailing: "→", action: next)
        }
    }
}

// MARK: - 4. собираем выпуск

private struct OnbBuilding: View {
    @Environment(\.theme) private var theme
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @Environment(AppModel.self) private var model
    @State private var t = 0

    private static let clusterTop: [CGFloat] = [0, 97, 194]

    var body: some View {
        if let edition = model.edition, let feed = model.feed {
            content(BuildingPlan.make(edition: edition, feed: feed))
        } else {
            VStack { ProgressView() }.frame(maxWidth: .infinity, maxHeight: .infinity)
        }
    }

    private func content(_ plan: BuildingPlan) -> some View {
        let anim = Anim(plan: plan, t: t)
        return VStack(alignment: .leading, spacing: 0) {
            HStack { Text("Выпуск № 1"); Spacer(); Text(anim.done ? "Готов" : "Собираем") }.metaStyle().padding(.top, 14)
            Text(RuFormat.two(anim.count))
                .font(theme.fonts.heading(104, .bold)).tracking(-104 * 0.045).monospacedDigit()
                .foregroundStyle(theme.ink).padding(.top, 20)
            Text(anim.countLabel).font(theme.fonts.body(16)).foregroundStyle(theme.muted).padding(.top, 6)
            HStack(spacing: 8) {
                Circle().fill(theme.accent).frame(width: 7, height: 7)
                    .opacity(anim.done || (t / 400) % 2 == 0 ? 1 : 0.25).animation(.easeInOut(duration: 0.35), value: t / 400)
                Text(anim.status).metaStyle(color: theme.accent)
            }
            .padding(.top, 22)
            .accessibilityElement(children: .combine)

            stage(plan, anim).padding(.top, 20)

            VStack(alignment: .leading, spacing: 10) {
                ForEach(Array(plan.stepTexts.enumerated()), id: \.offset) { i, text in
                    let ok = t > Anim.stepAt[i]
                    let blue = i == 3
                    HStack(spacing: 12) {
                        Circle().fill(ok ? (blue ? theme.accent : theme.ink) : .clear)
                            .overlay(Circle().strokeBorder(ok ? .clear : theme.chipBorder == .clear ? theme.line : theme.chipBorder, lineWidth: 1.5))
                            .overlay { if ok { Image(systemName: "checkmark").font(.system(size: 10, weight: .heavy)).foregroundStyle(theme.bg) } }
                            .frame(width: 22, height: 22)
                        Text(text).font(theme.fonts.body(15, ok && blue ? .medium : .regular))
                            .foregroundStyle(ok ? (blue ? theme.accent : theme.ink) : theme.muted.opacity(0.6))
                    }
                    .animation(.easeOut(duration: 0.3), value: ok)
                }
            }
            .padding(.top, 22)
            Spacer(minLength: 12)
            if anim.done {
                PrimaryButton(title: "Открыть выпуск", trailing: "→") {
                    Task { await model.completeOnboarding() }
                }
                .transition(.opacity)
            } else {
                Color.clear.frame(height: theme.buttonHeight)
            }
        }
        .padding(.horizontal, 20).padding(.bottom, 16)
        .animation(.easeOut(duration: 0.3), value: anim.done)
        .task {
            if reduceMotion { t = 5000; return }
            let clock = ContinuousClock()
            let start = clock.now
            while !Task.isCancelled, t <= 4800 {
                try? await Task.sleep(for: .milliseconds(50))
                t = Int((clock.now - start) / .milliseconds(1))
            }
            t = max(t, 4900)
        }
    }

    // MARK: сцена

    private func stage(_ plan: BuildingPlan, _ anim: Anim) -> some View {
        let moved = t > 2900
        return ZStack(alignment: .topLeading) {
            ForEach(0..<plan.slipCount, id: \.self) { i in
                let s = anim.slip(i)
                Capsule()
                    .fill(s.color)
                    .frame(width: s.width, height: 10)
                    .offset(x: s.x, y: s.y)
                    .opacity(s.opacity)
                    .animation(.timingCurve(0.2, 0.8, 0.2, 1, duration: 0.7).delay(s.delay), value: moved)
                    .animation(.easeOut(duration: 0.4), value: s.opacity)
                    .animation(.easeOut(duration: 0.4), value: s.color)
            }
            ForEach(Array(plan.cards.enumerated()), id: \.offset) { i, card in
                VStack(alignment: .leading) {
                    HStack { Text(card.tag); Spacer(); Text(card.merge) }.metaStyle(size: 10)
                    Text(card.title).font(theme.fonts.heading(15, .semibold)).tracking(-0.3).lineLimit(2).minimumScaleFactor(0.85).foregroundStyle(theme.ink)
                }
                .padding(.horizontal, 16).padding(.vertical, 12)
                .frame(width: 350, height: 76, alignment: .topLeading)
                .background(RoundedRectangle(cornerRadius: theme.cardRadius, style: .continuous).fill(theme.card))
                .overlay(RoundedRectangle(cornerRadius: theme.cardRadius, style: .continuous).strokeBorder(theme.line, lineWidth: 1))
                .offset(y: Self.clusterTop[i] + (t > 3800 ? 0 : 6))
                .opacity(t > 3800 ? 1 : 0)
                .animation(.easeOut(duration: 0.5).delay(Double(i) * 0.12), value: t > 3800)
            }
        }
        .frame(width: 350, height: 270, alignment: .topLeading)
        .accessibilityHidden(true)
    }

    // MARK: состояние анимации по времени `t` (мс)

    struct Anim {
        let plan: BuildingPlan
        let t: Int
        static let stepAt = [1200, 2600, 3900, 4400]

        struct Slip { var x: CGFloat; var y: CGFloat; var width: CGFloat; var color: Color; var opacity: Double; var delay: Double }

        var done: Bool { t > 4500 }

        private var visible: Int { (0..<plan.slipCount).filter { t > 100 + $0 * 28 }.count }
        private var filteredDone: Int {
            var k = 0, n = 0
            for i in 0..<plan.slipCount where plan.isFiltered(slip: i) { if t > 1600 + k * 50 { n += 1 }; k += 1 }
            return n
        }

        /// Огромное число на экране: сначала растут посты, затем тают до сюжетов.
        var count: Int {
            let total = plan.totalPosts
            if t < 1200 { return plan.slipCount == 0 ? 0 : Int(Double(total) * Double(visible) / Double(plan.slipCount)) }
            if t < 1600 { return total }
            if t < 2900 {
                let kept = plan.keptPosts
                return total - Int(Double(total - kept) * Double(filteredDone) / Double(max(plan.slipCount - plan.keptSlipCount, 1)))
            }
            if t < 3700 {
                let f = Double(t - 2900) / 800
                return max(plan.storyCount, Int((Double(plan.keptPosts) - Double(plan.keptPosts - plan.storyCount) * f).rounded()))
            }
            return plan.storyCount
        }

        var countLabel: String {
            if t < 1600 { return "постов из \(plan.sourceCount) \(RuFormat.plural(plan.sourceCount, one: "источника", few: "источников", many: "источников"))" }
            if t < 2900 { return "осталось после фильтров" }
            return "\(RuFormat.plural(plan.storyCount, one: "сюжет", few: "сюжета", many: "сюжетов")) в первом выпуске"
        }

        var status: String {
            switch t {
            case ..<1200: "Читаем источники"
            case ..<2600: "Убираем рекламу и стоп-темы"
            case ..<3900: "Склеиваем повторы"
            case ..<4400: "Ищем законы про тебя"
            default: "Готово"
            }
        }

        func slip(_ i: Int) -> Slip {
            let col = i % 4, row = i / 4
            let w = CGFloat(44 + ((i * 37) % 36))
            var s = Slip(x: CGFloat(col) * 89, y: CGFloat(row) * 27, width: w, color: Color(hex: 0x8C8A83), opacity: t > 100 + i * 28 ? 0.55 : 0, delay: 0)
            if plan.isFiltered(slip: i) {
                let k = (0..<i).filter { plan.isFiltered(slip: $0) }.count
                if t > 1600 + k * 50 { s.opacity = 0.1; s.color = Color(hex: 0xA9A69C) }
                if t > 2900 { s.opacity = 0 }
            } else {
                let j = (0..<i).filter { !plan.isFiltered(slip: $0) }.count
                if t > 1600 { s.color = Color(hex: 0x22211E); s.opacity = 0.8 }
                if t > 2900, !plan.cards.isEmpty {
                    let c = plan.cluster(forKept: j)
                    let within = (0..<j).filter { plan.cluster(forKept: $0) == c }.count
                    s.x = 16
                    s.y = OnbBuilding.clusterTop[min(c, 2)] + 22 + CGFloat(within) * 2
                    s.width = 240
                    s.delay = Double(j) * 0.022
                }
                if t > 3800 { s.opacity = 0 }
            }
            return s
        }
    }
}
