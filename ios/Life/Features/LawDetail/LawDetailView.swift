import SwiftUI

struct LawDetailView: View {
    @Environment(\.theme) private var theme
    @Environment(\.dismiss) private var dismiss
    @Environment(AppModel.self) private var model
    let law: Law
    @State private var done: Set<Int> = []

    private var today: CalendarDate { model.today }
    private var days: Int? { law.dates.effective.map { $0.days(from: today) } }
    private var isSage: Bool { theme.id == .sage }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                topBar
                breadcrumb
                title
                countdown
                statusSteps
                facts
                sources
                Text("Пересказ простым языком, не юридическая консультация.\(verified)")
                    .font(isSage ? theme.fonts.body(12) : ThemeFonts.mono(11))
                    .lineSpacing(4)
                    .foregroundStyle(theme.muted)
                    .padding(.top, 20)
                reminderButton
            }
            .padding(.horizontal, 20)
            .padding(.bottom, 34)
        }
        .scrollIndicators(.hidden)
        .screenBackground(theme)
        .toolbar(.hidden, for: .navigationBar)
    }

    private var verified: String {
        law.verifiedAt.map { " Сверено с официальным текстом \(RuFormat.numeric(CalendarDate($0)))." } ?? ""
    }

    // MARK: шапка

    private var topBar: some View {
        VStack(spacing: 0) {
            HStack {
                Button { dismiss() } label: {
                    HStack(spacing: 4) {
                        if isSage { Image(systemName: "chevron.left").font(.system(size: 17, weight: .semibold)) }
                        Text(isSage ? "Выпуск" : "← Выпуск \(model.edition?.number ?? 1)")
                    }
                    .font(theme.fonts.body(15, .medium))
                    .foregroundStyle(isSage ? theme.accentDeep : theme.ink)
                    .frame(minHeight: 48)
                }
                .buttonStyle(.plain)
                Spacer()
                if let url = law.officialUrl ?? law.billUrl {
                    ShareLink(item: url, subject: Text(law.title), message: Text(law.title)) {
                        if isSage {
                            Image(systemName: "square.and.arrow.up")
                                .foregroundStyle(theme.accentDeep)
                                .frame(width: 44, height: 44)
                                .background(Circle().fill(theme.card))
                        } else {
                            Text("Поделиться").font(theme.fonts.body(15, .medium)).foregroundStyle(theme.ink).frame(minHeight: 48)
                        }
                    }
                }
            }
            if !isSage { Rule(strong: true) }
        }
    }

    private var breadcrumb: some View {
        let tags = AudienceMatcher.audienceTags(for: model.profile.snapshot)
        let audience = AudienceLabels.label(for: law, profileTags: tags)
        return Group {
            if isSage {
                Text(audience)
                    .font(theme.fonts.body(13, .medium)).foregroundStyle(theme.accentDeep)
                    .padding(.horizontal, 12).frame(minHeight: 28)
                    .background(Capsule().fill(theme.plate))
            } else {
                Text("Закон / \(audience)").metaStyle()
            }
        }
        .padding(.top, 22)
    }

    private var title: some View {
        Text(law.title)
            .font(theme.fonts.heading(isSage ? 32 : 34, .semibold))
            .tracking(isSage ? -0.3 : -1.36)
            .lineSpacing(isSage ? -6 : -2)
            .foregroundStyle(theme.ink)
            .fixedSize(horizontal: false, vertical: true)
            .padding(.top, 12)
            .accessibilityAddTraits(.isHeader)
    }

    // MARK: отсчёт

    @ViewBuilder private var countdown: some View {
        if let eff = law.dates.effective, let d = days {
            HStack(alignment: .bottom, spacing: 16) {
                if isSage {
                    VStack(alignment: .leading, spacing: 4) {
                        Text(d > 0 ? RuFormat.daysWords(d) : "Сегодня")
                            .font(theme.fonts.heading(52, .semibold)).foregroundStyle(theme.accent)
                        Text(d > 0 ? "до вступления в силу" : "вступает в силу").font(theme.fonts.body(14)).foregroundStyle(theme.body)
                    }
                    Spacer()
                    VStack(alignment: .trailing, spacing: 2) {
                        Text(RuFormat.dayMonth(eff, today: CalendarDate(year: eff.year, month: 1, day: 1)))
                            .font(theme.fonts.heading(20, .semibold)).foregroundStyle(theme.ink)
                        Text(verbatim: String(eff.year)).font(theme.fonts.body(13)).foregroundStyle(theme.muted)
                    }
                } else {
                    Text(verbatim: d > 0 ? "Д–\(d)" : "Сегодня")
                        .font(theme.fonts.heading(96, .bold)).tracking(-96 * 0.07)
                        .monospacedDigit().foregroundStyle(theme.accent)
                        .lineLimit(1).minimumScaleFactor(0.5)
                    Spacer(minLength: 8)
                    VStack(alignment: .trailing, spacing: 4) {
                        Text("До вступления").metaStyle()
                        Text("в силу").metaStyle()
                        Text(RuFormat.numeric(eff)).metaStyle(color: theme.ink)
                    }
                }
            }
            .padding(.top, 28)
            .accessibilityElement(children: .combine)
        }
    }

    // MARK: шкала статуса

    private var statusSteps: some View {
        let order: [LawStatus] = [.introduced, .passed, .signed, .inForce]
        let current = order.firstIndex(of: law.status) ?? 0
        let dates: [CalendarDate?] = [law.dates.introduced, law.dates.passed, law.dates.signed, law.dates.effective]
        return HStack(alignment: .top, spacing: 4) {
            ForEach(order.indices, id: \.self) { i in
                VStack(alignment: .leading, spacing: 8) {
                    stepMark(done: i < current || (law.status == .inForce && i == 3), current: i == current && law.status != .inForce)
                    Text(order[i].title)
                        .font(isSage ? theme.fonts.body(13, .medium) : ThemeFonts.mono(10))
                        .tracking(isSage ? 0 : 0.4)
                        .textCase(isSage ? nil : .uppercase)
                        .foregroundStyle(i == current ? theme.accent : (i < current ? theme.ink : theme.muted))
                    Text(dates[i].map { isSage ? RuFormat.dayMonthShort($0) : RuFormat.numericShort($0, today: today) } ?? "—")
                        .font(isSage ? theme.fonts.body(12) : ThemeFonts.mono(10))
                        .tracking(isSage ? 0 : 0.4)
                        .foregroundStyle(theme.muted)
                }
                .frame(maxWidth: .infinity, alignment: .leading)
            }
        }
        .padding(.top, 26)
        .accessibilityElement(children: .combine)
        .accessibilityLabel("Статус закона: \(law.status.title)")
    }

    @ViewBuilder private func stepMark(done: Bool, current: Bool) -> some View {
        let color = current ? theme.accent : (done ? (isSage ? theme.accent : theme.ink) : theme.line)
        if isSage {
            HStack(spacing: 0) {
                Circle().fill(done || current ? color : .clear)
                    .overlay(Circle().strokeBorder(done || current ? color : theme.muted.opacity(0.5), lineWidth: 1.5))
                    .frame(width: 12, height: 12)
                Capsule().fill(color).frame(height: 3)
            }
            .frame(height: 16)
        } else if done || current {
            Capsule().fill(color).frame(height: 4)
        } else {
            Rectangle().fill(.clear).frame(height: 4)
                .overlay(
                    Path { p in p.move(to: CGPoint(x: 0, y: 2)); p.addLine(to: CGPoint(x: 400, y: 2)) }
                        .stroke(theme.muted.opacity(0.6), style: StrokeStyle(lineWidth: 4, lineCap: .round, dash: [0.1, 7]))
                )
                .clipped()
        }
    }

    // MARK: Что / Кого / Тебе / Сделать

    private var facts: some View {
        VStack(alignment: .leading, spacing: 0) {
            if !isSage { Rule(strong: true) }
            FactRow(label: isSage ? "Что изменилось" : "Что", text: law.whatChanged)
            FactRow(label: isSage ? "Кого касается" : "Кого", text: law.whoAffected)
            FactRow(label: isSage ? "Почему тебе" : "Тебе", text: AudienceLabels.reason(for: law, profile: model.profile.snapshot), accent: true)
            if !law.actions.isEmpty {
                if isSage {
                    VStack(alignment: .leading, spacing: 0) {
                        Text("Что сделать").font(theme.fonts.body(13)).foregroundStyle(theme.muted).padding(.bottom, 4)
                        checklist
                    }
                    .padding(16)
                    .background(RoundedRectangle(cornerRadius: theme.cardRadius, style: .continuous).fill(theme.card))
                    .padding(.top, 12)
                } else {
                    Rule()
                    HStack(alignment: .top, spacing: 0) {
                        Text("Сделать").metaStyle().frame(width: 92, alignment: .leading).padding(.top, 16)
                        checklist
                    }
                    .padding(.vertical, 2)
                }
            }
            if !isSage { Rule() }
        }
        .padding(.top, 36)
    }

    private var checklist: some View {
        VStack(alignment: .leading, spacing: 0) {
            ForEach(law.actions.indices, id: \.self) { i in
                Button {
                    if done.contains(i) { done.remove(i) } else { done.insert(i) }
                } label: {
                    HStack(spacing: 12) {
                        RoundedRectangle(cornerRadius: isSage ? 10 : 4, style: .continuous)
                            .strokeBorder(isSage ? theme.accent : theme.ink, lineWidth: 1.5)
                            .background(RoundedRectangle(cornerRadius: isSage ? 10 : 4).fill(done.contains(i) ? (isSage ? theme.accent : theme.ink) : .clear))
                            .overlay {
                                if done.contains(i) {
                                    Image(systemName: "checkmark").font(.system(size: 11, weight: .bold))
                                        .foregroundStyle(isSage ? theme.card : theme.bg)
                                }
                            }
                            .frame(width: 20, height: 20)
                        Text(law.actions[i]).font(theme.fonts.body(16)).foregroundStyle(theme.ink)
                            .strikethrough(done.contains(i), color: theme.muted)
                            .multilineTextAlignment(.leading)
                        Spacer(minLength: 0)
                    }
                    .frame(minHeight: 48)
                    .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
                .accessibilityAddTraits(done.contains(i) ? [.isButton, .isSelected] : .isButton)
            }
        }
    }

    // MARK: источники

    @ViewBuilder private var sources: some View {
        if law.officialUrl != nil || law.billUrl != nil {
            VStack(alignment: .leading, spacing: 0) {
                if isSage {
                    Spacer().frame(height: 12)
                } else {
                    Text("Источники").metaStyle(color: theme.ink).padding(.top, 36).padding(.bottom, 12)
                    Rule(strong: true)
                }
                if let url = law.officialUrl {
                    SourceRow(title: "Официальный текст закона", subtitle: [url.host(), law.actNumber].compactMap { $0 }.joined(separator: " · "), url: url)
                }
                if let url = law.billUrl {
                    SourceRow(title: "Ход законопроекта", subtitle: url.host() ?? "", url: url)
                }
            }
        }
    }

    // MARK: напоминание

    @ViewBuilder private var reminderButton: some View {
        let has = model.hasReminder(law)
        if let plan = model.reminderPlan(for: law) {
            PrimaryButton(
                title: has ? "Напоминание включено" : ReminderPlanner.buttonTitle(plan, today: today),
                trailing: has ? RuFormat.numericShort(plan.day, today: today) : nil,
                systemImage: isSage ? (has ? "bell.fill" : "bell") : nil,
                centered: isSage
            ) { model.toggleReminder(law) }
            .padding(.top, 28)
        } else if has {
            PrimaryButton(title: "Отключить напоминание", centered: isSage) { model.toggleReminder(law) }.padding(.top, 28)
        }
    }
}

private struct FactRow: View {
    @Environment(\.theme) private var theme
    let label: String
    let text: String
    var accent = false

    var body: some View {
        if theme.id == .sage {
            VStack(alignment: .leading, spacing: 6) {
                Text(label).font(theme.fonts.body(13)).foregroundStyle(accent ? theme.accentDeep : theme.muted)
                Text(text).font(theme.fonts.body(16)).lineSpacing(4).foregroundStyle(theme.ink).fixedSize(horizontal: false, vertical: true)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(16)
            .background(RoundedRectangle(cornerRadius: theme.cardRadius, style: .continuous).fill(accent ? theme.plate : theme.card))
            .padding(.top, 12)
        } else {
            VStack(spacing: 0) {
                HStack(alignment: .top, spacing: 0) {
                    Text(label).metaStyle(color: accent ? theme.accent : nil)
                        .frame(width: 92, alignment: .leading).padding(.top, 3)
                    Text(text).font(theme.fonts.body(16)).lineSpacing(4).foregroundStyle(theme.ink)
                        .fixedSize(horizontal: false, vertical: true)
                        .frame(maxWidth: .infinity, alignment: .leading)
                }
                .padding(.vertical, 16)
                Rule()
            }
        }
    }
}

private struct SourceRow: View {
    @Environment(\.theme) private var theme
    let title: String
    let subtitle: String
    let url: URL

    var body: some View {
        VStack(spacing: 0) {
            Link(destination: url) {
                HStack(spacing: 12) {
                    VStack(alignment: .leading, spacing: 4) {
                        Text(title).font(theme.fonts.body(16, .medium)).foregroundStyle(theme.ink)
                        Text(subtitle).font(theme.id == .sage ? theme.fonts.body(12) : ThemeFonts.mono(11)).foregroundStyle(theme.muted)
                    }
                    Spacer()
                    Text("↗").font(theme.fonts.body(20)).foregroundStyle(theme.ink)
                }
                .frame(minHeight: 68)
                .contentShape(Rectangle())
            }
            Rule()
        }
    }
}
