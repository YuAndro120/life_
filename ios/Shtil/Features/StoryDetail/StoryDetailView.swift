import SwiftUI

struct StoryRoute: Hashable {
    let id: String
}

/// Экран сюжета: пересказ, «что это значит» и ссылки на оригиналы. Полный текст остаётся у источников.
struct StoryDetailView: View {
    @Environment(\.theme) private var theme
    @Environment(\.dismiss) private var dismiss
    @Environment(AppModel.self) private var model
    let story: Story

    private var isSage: Bool { theme.id == .sage }

    private var meta: String {
        var text = isSage
            ? "\(story.topic.title) · \(story.infoType.title.lowercased())"
            : "\(story.topic.title) / \(story.infoType.title)"
        if story.countryCode != "RU", let country = NewsCountry(rawValue: story.countryCode) {
            text += " · \(country.title)"
        }
        return text
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                topBar
                Text(meta).metaStyle().padding(.top, 20)
                Text(story.title)
                    .font(theme.fonts.heading(isSage ? 28 : 26, .semibold))
                    .tracking(isSage ? 0 : -0.6)
                    .foregroundStyle(theme.ink)
                    .fixedSize(horizontal: false, vertical: true)
                    .padding(.top, 10)
                Text(story.summary)
                    .font(theme.fonts.body(17)).lineSpacing(5)
                    .foregroundStyle(theme.body)
                    .fixedSize(horizontal: false, vertical: true)
                    .padding(.top, 16)
                if let meaning = story.meaning, !meaning.isEmpty {
                    (Text(isSage ? "Значит, " : "Значит: ").fontWeight(.semibold).foregroundColor(theme.ink)
                        + Text(isSage ? meaning.lowercasedFirst : meaning))
                        .font(theme.fonts.body(16)).lineSpacing(4)
                        .foregroundStyle(theme.body)
                        .fixedSize(horizontal: false, vertical: true)
                        .padding(.top, 16)
                }
                sources
                Text("Пересказ сделан автоматически по публикациям источников. Проверяйте важное по оригиналам.")
                    .font(isSage ? theme.fonts.body(12) : ThemeFonts.mono(11))
                    .lineSpacing(4)
                    .foregroundStyle(theme.muted)
                    .padding(.top, 20)
                feedback
            }
            .padding(.horizontal, 20)
            .padding(.bottom, 34)
        }
        .scrollIndicators(.hidden)
        .screenBackground(theme)
        .toolbar(.hidden, for: .navigationBar)
    }

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
                if let first = story.sources.first {
                    ShareLink(item: first.url, subject: Text(story.title), message: Text(story.title)) {
                        Text(isSage ? "" : "Поделиться")
                            .font(theme.fonts.body(15, .medium)).foregroundStyle(theme.ink).frame(minHeight: 48)
                            .overlay { if isSage { Image(systemName: "square.and.arrow.up").foregroundStyle(theme.accentDeep) } }
                            .frame(minWidth: isSage ? 44 : 0)
                    }
                }
            }
            if !isSage { Rule(strong: true) }
        }
    }

    @ViewBuilder private var sources: some View {
        if !story.sources.isEmpty {
            VStack(alignment: .leading, spacing: 0) {
                Text(isSage ? "Читать полностью" : "ЧИТАТЬ ПОЛНОСТЬЮ").metaStyle().padding(.top, 24).padding(.bottom, 6)
                ForEach(Array(story.sources.enumerated()), id: \.element.url) { index, source in
                    Rule()
                    Link(destination: source.url) {
                        HStack {
                            VStack(alignment: .leading, spacing: 2) {
                                Text(source.title).font(theme.fonts.body(16, .medium)).foregroundStyle(theme.ink)
                                if let note = note(for: source, at: index) {
                                    Text(note).font(theme.fonts.body(12)).foregroundStyle(theme.muted)
                                }
                            }
                            Spacer(minLength: 8)
                            Text("↗").foregroundStyle(theme.muted)
                        }
                        .frame(minHeight: 48)
                        .contentShape(Rectangle())
                    }
                }
                Rule()
            }
        }
    }

    /// Подпись под источником: где искать оригинал, а где пересказ. Сервер отдаёт первоисточники первыми.
    private func note(for source: SourceLink, at index: Int) -> String? {
        if source.isReprint { return "пересказывает более раннее сообщение" }
        guard story.sources.count > 1 else { return nil }
        return index == 0 ? "первоисточник" : "сообщил независимо"
    }

    private var feedback: some View {
        VStack(alignment: .leading, spacing: 0) {
            Text(isSage ? "Лента" : "ЛЕНТА").metaStyle().padding(.top, 26).padding(.bottom, 6)
            Rule()
            row("Не интересно, скрыть сюжет", action: .hideStory(id: story.id))
            Rule()
            row("Меньше про «\(story.topic.title)»", action: .muteTopic(story.topic))
            if !model.settings.preferences.interests.contains(story.topic) {
                Rule()
                row("Больше про «\(story.topic.title)»", action: .boostTopic(story.topic))
            }
            Rule()
        }
    }

    private func row(_ title: String, action: FeedbackAction) -> some View {
        Button {
            model.apply(action)
            dismiss()
        } label: {
            Text(title).font(theme.fonts.body(15)).foregroundStyle(theme.body)
                .frame(maxWidth: .infinity, minHeight: 48, alignment: .leading)
                .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
    }
}
