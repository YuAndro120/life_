import SwiftUI

struct LawRoute: Hashable {
    let id: String
}

extension String {
    var lowercasedFirst: String {
        guard let first else { return self }
        return first.lowercased() + dropFirst()
    }
}

extension LawStatus {
    var title: String {
        switch self {
        case .introduced: "Внесён"
        case .passed: "Принят"
        case .signed: "Подписан"
        case .inForce: "В силе"
        }
    }
}

struct TodayView: View {
    @Environment(\.theme) private var theme
    @Environment(AppModel.self) private var model
    var openFilters: () -> Void
    var openCalendar: () -> Void

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                if let edition = model.edition {
                    header(edition)
                    content(edition)
                } else if model.loadError != nil {
                    Text(model.loadError ?? "").font(theme.fonts.body(16)).foregroundStyle(theme.muted).padding(20)
                } else {
                    ProgressView().frame(maxWidth: .infinity).padding(.top, 120)
                }
            }
        }
        .scrollIndicators(.hidden)
        .screenBackground(theme)
        .refreshable { await model.refresh() }
    }

    // MARK: шапка

    @ViewBuilder private func header(_ e: Edition) -> some View {
        if theme.id == .sage { sageHeader(e) } else { paperHeader(e) }
    }

    private var calmLabel: String { model.settings.calmMode ? "Спокойно" : "Обычный" }

    private func paperHeader(_ e: Edition) -> some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack {
                Text("Выпуск № \(e.number)")
                Spacer()
                Text(RuFormat.editionStamp(date: model.now))
            }
            .metaStyle()
            .padding(.top, 14)

            HStack(alignment: .bottom) {
                (Text("Суть") + Text(".").foregroundStyle(theme.accent))
                    .font(theme.fonts.heading(76, .bold))
                    .tracking(-76 * 0.04)
                    .foregroundStyle(theme.ink)
                    .lineLimit(1)
                    .minimumScaleFactor(0.6)
                Spacer()
                Button(action: openFilters) {
                    HStack(spacing: 7) {
                        Circle().fill(theme.accent).frame(width: 8, height: 8)
                        Text(calmLabel).metaStyle(color: theme.ink)
                    }
                    .frame(minHeight: 44)
                }
                .buttonStyle(.plain)
                .accessibilityLabel("Фильтры: \(calmLabel)")
            }
            .padding(.top, 8)

            Rectangle().fill(theme.ink).frame(height: 1).padding(.top, 10)
            stats(e)
            Rule()
        }
        .padding(.horizontal, 20)
    }

    private func stats(_ e: Edition) -> some View {
        HStack(alignment: .top, spacing: 0) {
            stat(RuFormat.two(e.stats.aboutYou), "Про тебя", accent: true, first: true)
            stat(RuFormat.two(e.stats.stories), "Сюжета", accent: false, first: false)
            stat(RuFormat.two(e.stats.readingMinutes), "Мин чтения", accent: false, first: false)
        }
        .padding(.vertical, 14)
    }

    private func stat(_ value: String, _ label: String, accent: Bool, first: Bool) -> some View {
        HStack(spacing: 0) {
            if !first { Rectangle().fill(theme.line).frame(width: 1).padding(.trailing, 14) }
            VStack(alignment: .leading, spacing: 6) {
                Text(value)
                    .font(theme.fonts.heading(44, .semibold))
                    .tracking(-44 * 0.045)
                    .monospacedDigit()
                    .foregroundStyle(accent ? theme.accent : theme.ink)
                Text(label).metaStyle()
            }
            Spacer(minLength: 0)
        }
        .frame(maxWidth: .infinity)
        .accessibilityElement(children: .combine)
    }

    private func sageHeader(_ e: Edition) -> some View {
        let hour = Calendar.current.component(.hour, from: model.now)
        return VStack(alignment: .leading, spacing: 0) {
            HStack {
                Text("Суть").font(theme.fonts.heading(22, .semibold)).foregroundStyle(theme.ink)
                Spacer()
                Button(action: openFilters) {
                    HStack(spacing: 8) {
                        Circle().fill(theme.accent).frame(width: 8, height: 8)
                        Text(model.settings.calmMode ? "Спокойный режим" : "Обычный режим")
                            .font(theme.fonts.body(13, .medium)).foregroundStyle(theme.accentDeep)
                    }
                    .padding(.horizontal, 14)
                    .frame(minHeight: 44)
                    .background(Capsule().fill(theme.plate))
                }
                .buttonStyle(.plain)
            }
            .padding(.top, 10)

            Text("\(RuFormat.longDay(model.today)) · выпуск \(e.number)")
                .font(theme.fonts.body(13)).foregroundStyle(theme.muted).padding(.top, 22)

            VStack(alignment: .leading, spacing: 0) {
                Text(RuFormat.greeting(hour: hour)).font(theme.fonts.heading(40, .medium)).tracking(-0.8)
                Text("Вот что важно сегодня").font(theme.fonts.heading(40, .regular, italic: true))
                    .foregroundStyle(theme.muted).lineLimit(1).minimumScaleFactor(0.7)
            }
            .lineSpacing(-8)
            .foregroundStyle(theme.ink)
            .padding(.top, 8)

            HStack(alignment: .top, spacing: 12) {
                sageStat("\(e.stats.aboutYou)", "про тебя", accent: true)
                sageStat("\(e.stats.stories)", "сюжета", accent: false)
                sageStat("\(e.stats.readingMinutes)", "мин чтения", accent: false)
            }
            .padding(.top, 24)
        }
        .padding(.horizontal, 20)
    }

    private func sageStat(_ value: String, _ label: String, accent: Bool) -> some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(value).font(theme.fonts.heading(36, .semibold)).foregroundStyle(accent ? theme.accent : theme.ink)
            Text(label).font(theme.fonts.body(13)).foregroundStyle(theme.muted)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(14)
        .background(RoundedRectangle(cornerRadius: 18, style: .continuous).fill(theme.card))
        .accessibilityElement(children: .combine)
    }

    // MARK: содержимое

    @ViewBuilder private func content(_ e: Edition) -> some View {
        if e.laws.isEmpty && e.stories.isEmpty && e.foldedHeavy.isEmpty {
            Text("Сегодня тихо.").font(theme.fonts.heading(26, .medium)).foregroundStyle(theme.ink)
                .padding(20).padding(.top, 20)
        }
        if !e.laws.isEmpty { lawsSection(e) }
        if !e.stories.isEmpty || !e.foldedHeavy.isEmpty { storiesSection(e) }
        endOfEdition
    }

    private func lawsSection(_ e: Edition) -> some View {
        let tags = AudienceMatcher.audienceTags(for: model.profile.snapshot)
        return VStack(alignment: .leading, spacing: 8) {
            HStack {
                Group {
                    if theme.id == .sage {
                        Text("Касается тебя").font(theme.fonts.body(15, .semibold))
                    } else {
                        Text("01 — Касается тебя").metaStyle(color: theme.accentDeep)
                    }
                }
                .foregroundStyle(theme.accentDeep)
                .accessibilityAddTraits(.isHeader)
                Spacer()
                Button(action: openCalendar) {
                    Text("Календарь →")
                        .font(theme.id == .sage ? theme.fonts.body(14, .medium) : ThemeFonts.mono(11))
                        .tracking(theme.id == .sage ? 0 : 0.44)
                        .textCase(theme.id == .sage ? nil : .uppercase)
                        .foregroundStyle(theme.accentDeep)
                        .frame(minHeight: 44)
                }
                .buttonStyle(.plain)
            }
            .padding(.leading, 12).padding(.trailing, 8)

            ForEach(e.laws) { law in
                NavigationLink(value: LawRoute(id: law.id)) {
                    LawCard(law: law, profileTags: tags, today: model.today)
                }
                .buttonStyle(.plain)
            }
        }
        .padding(.top, 6).padding(.horizontal, 8).padding(.bottom, 8)
        .background(RoundedRectangle(cornerRadius: theme.plateRadius, style: .continuous).fill(theme.plate))
        .padding(.horizontal, 12)
        .padding(.top, 28)
    }

    private func storiesSection(_ e: Edition) -> some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack {
                Group {
                    if theme.id == .sage {
                        Text("Главное за сутки").font(theme.fonts.heading(26, .medium))
                    } else {
                        Text("\(e.laws.isEmpty ? "01" : "02") — Главное за сутки").metaStyle(color: theme.ink)
                    }
                }
                .foregroundStyle(theme.ink)
                .accessibilityAddTraits(.isHeader)
                Spacer()
                Text("\(e.stats.postsTotal) постов → \(e.stats.stories)")
                    .font(theme.id == .sage ? theme.fonts.body(13) : ThemeFonts.mono(11))
                    .tracking(theme.id == .sage ? 0 : 0.44)
                    .textCase(theme.id == .sage ? nil : .uppercase)
                    .foregroundStyle(theme.muted)
            }
            .frame(minHeight: 44)

            ForEach(Array(e.stories.enumerated()), id: \.element.id) { index, story in
                if theme.id != .sage { Rule(strong: index == 0) } else if index > 0 { Rule() }
                StoryRow(story: story)
            }

            if !e.foldedHeavy.isEmpty {
                HeavyBlock(stories: e.foldedHeavy)
            }
        }
        .padding(.horizontal, 20)
        .padding(.top, 44)
    }

    private var endOfEdition: some View {
        VStack(alignment: .leading, spacing: 14) {
            if theme.id == .sage {
                Circle().fill(theme.accent).frame(width: 10, height: 10)
                Text("На сегодня всё.").font(theme.fonts.heading(36, .regular, italic: true)).foregroundStyle(theme.ink)
                if let next = model.schedule.nextCaption(after: model.now) {
                    Text("Следующий выпуск \(next)").font(theme.fonts.body(14)).foregroundStyle(theme.muted)
                }
            } else {
                (Text("Конец\nвыпуска") + Text(".").foregroundStyle(theme.accent))
                    .font(theme.fonts.heading(56, .bold)).tracking(-56 * 0.045).lineSpacing(-6)
                    .foregroundStyle(theme.ink)
                if let next = model.schedule.nextCaption(after: model.now) {
                    Text("Следующий — \(next)").metaStyle()
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.horizontal, 20)
        .padding(.top, 56).padding(.bottom, 40)
        .accessibilityElement(children: .combine)
    }
}

// MARK: - карточка закона

struct LawCard: View {
    @Environment(\.theme) private var theme
    let law: Law
    let profileTags: Set<String>
    let today: CalendarDate

    private var days: Int? { law.dates.effective.map { $0.days(from: today) } }

    private var doLine: String {
        law.actions.first.map { "Сделать: \($0.lowercasedFirst)" } ?? "Сделать: пока ничего"
    }

    private var metaLeft: String {
        guard let eff = law.dates.effective else { return law.status.title }
        if theme.id == .sage { return "С \(RuFormat.dayMonth(eff, today: today))" }
        let near = (days ?? 0) <= 30 && law.status == .signed
        let date = "в силе с \(RuFormat.numericShort(eff, today: today))"
        return near ? date.prefix(1).uppercased() + date.dropFirst() : "\(law.status.title) · \(date)"
    }

    private var metaRight: String? {
        guard let d = days else { return nil }
        return theme.id == .sage ? RuFormat.countdownWords(days: d) : RuFormat.countdownShort(days: d)
    }

    var body: some View {
        let audience = AudienceLabels.label(for: law, profileTags: profileTags)
        HStack(alignment: .top, spacing: 12) {
            if theme.id != .sage {
                Circle().fill(theme.accent).frame(width: 10, height: 10).padding(.top, 5)
            }
            VStack(alignment: .leading, spacing: 10) {
                HStack {
                    Text(metaLeft).metaStyle(color: theme.accent)
                    Spacer()
                    if let metaRight { Text(metaRight).metaStyle(color: theme.id == .sage ? theme.muted : theme.ink) }
                }
                Text(law.title)
                    .font(theme.fonts.heading(theme.id == .sage ? 22 : 23, .semibold))
                    .tracking(theme.id == .sage ? -0.2 : -0.69)
                    .foregroundStyle(theme.ink)
                    .multilineTextAlignment(.leading)
                    .fixedSize(horizontal: false, vertical: true)
                Text("\(audience) · \(theme.id == .sage ? sageDo : doLine)")
                    .font(theme.fonts.body(14)).foregroundStyle(theme.muted)
                    .multilineTextAlignment(.leading)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
        .padding(.leading, 16).padding(.trailing, 18).padding(.vertical, 18)
        .background(
            RoundedRectangle(cornerRadius: theme.cardRadius, style: .continuous).fill(theme.card)
                .shadow(color: theme.accent.opacity(0.06), radius: 1, y: 1)
        )
        .contentShape(Rectangle())
        .accessibilityElement(children: .combine)
    }

    private var sageDo: String {
        law.actions.first.map { $0.lowercasedFirst } ?? "пока ничего делать не нужно"
    }
}

// MARK: - сюжет

struct StoryRow: View {
    @Environment(\.theme) private var theme
    let story: Story

    private var meta: String {
        theme.id == .sage
            ? "\(story.topic.title) · \(story.infoType.title.lowercased())"
            : "\(story.topic.title) / \(story.infoType.title)"
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                HStack(spacing: 8) {
                    if theme.id == .sage { Circle().fill(theme.muted).frame(width: 6, height: 6) }
                    Text(meta).metaStyle()
                }
                Spacer()
                Text("\(story.postCount) → 1").metaStyle()
            }
            Text(story.title)
                .font(theme.fonts.heading(theme.id == .sage ? 22 : 21, .semibold))
                .tracking(theme.id == .sage ? 0 : -0.5)
                .foregroundStyle(theme.ink)
                .fixedSize(horizontal: false, vertical: true)
            if let meaning = story.meaning, !meaning.isEmpty {
                (Text(theme.id == .sage ? "Значит, " : "Значит: ").fontWeight(.medium).foregroundColor(theme.ink)
                    + Text(theme.id == .sage ? meaning.lowercasedFirst : meaning))
                    .font(theme.fonts.body(15)).lineSpacing(3)
                    .foregroundStyle(theme.body)
                    .fixedSize(horizontal: false, vertical: true)
            }
            if !story.sources.isEmpty {
                HStack(spacing: 12) {
                    ForEach(story.sources, id: \.url) { source in
                        Link(destination: source.url) {
                            Text("\(source.title) ↗")
                                .font(theme.id == .sage ? theme.fonts.body(13) : ThemeFonts.mono(11))
                                .foregroundStyle(theme.muted)
                                .frame(minHeight: 32)
                        }
                    }
                }
            }
        }
        .padding(.top, 18).padding(.bottom, theme.id == .sage ? 22 : 14)
        .frame(maxWidth: .infinity, alignment: .leading)
    }
}

// MARK: - свёрнутые тяжёлые темы

struct HeavyBlock: View {
    @Environment(\.theme) private var theme
    let stories: [Story]
    @State private var expanded = false
    @State private var showAll = false

    private var digest: String {
        stories.map { $0.title.hasSuffix(".") ? $0.title : $0.title + "." }.joined(separator: " ")
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            Rule(strong: theme.id != .sage)
            Button {
                withAnimation(.easeOut(duration: 0.2)) { expanded.toggle() }
            } label: {
                HStack(spacing: 12) {
                    VStack(alignment: .leading, spacing: 4) {
                        Text("Тяжёлые темы").font(theme.fonts.body(17, .medium)).tracking(-0.17).foregroundStyle(theme.ink)
                        Text("\(RuFormat.two(stories.count)) \(RuFormat.plural(stories.count, one: "сюжет", few: "сюжета", many: "сюжетов")) · \(expanded ? "раскрыто" : "свёрнуто")")
                            .metaStyle()
                    }
                    Spacer()
                    Text(expanded ? "−" : "+").font(theme.fonts.body(28)).foregroundStyle(theme.ink).frame(width: 28)
                }
                .frame(minHeight: 64)
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .accessibilityValue(expanded ? "раскрыто" : "свёрнуто")

            if expanded {
                VStack(alignment: .leading, spacing: 14) {
                    Text(digest).font(theme.fonts.body(15)).lineSpacing(4).foregroundStyle(theme.body)
                    if !showAll {
                        Button { showAll = true } label: {
                            Text("Открыть все \(stories.count)")
                                .font(theme.fonts.body(14, .medium)).foregroundStyle(theme.ink)
                                .padding(.horizontal, 16).frame(minHeight: 44)
                                .overlay(Capsule().strokeBorder(theme.ink, lineWidth: 1))
                        }
                        .buttonStyle(.plain)
                    } else {
                        ForEach(stories) { story in
                            Rule()
                            StoryRow(story: story)
                        }
                    }
                }
                .padding(.bottom, 20)
            }
            Rule()
        }
    }
}
