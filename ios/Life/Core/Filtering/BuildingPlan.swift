import Foundation

/// Числа для анимации «Собираем выпуск»: настоящие, из ответа сервера и результата FilterEngine.
struct BuildingPlan: Equatable, Sendable {
    struct Card: Equatable, Sendable {
        var tag: String
        var merge: String
        var title: String
        var posts: Int
    }

    var totalPosts: Int
    var sourceCount: Int
    var keptPosts: Int
    var hiddenPosts: Int
    var storyCount: Int
    var lawCount: Int
    var cards: [Card]

    /// Не больше стольких полосок-постов рисуем, даже если постов сотни.
    static let maxSlips = 38

    static func make(edition: Edition, feed: Feed) -> BuildingPlan {
        let all = edition.stories + edition.foldedHeavy
        let total = max(feed.stats.postsTotal, 0)
        let kept = min(all.reduce(0) { $0 + $1.postCount }, total)
        let sources = Set(feed.stories.flatMap { $0.sources.map(\.title) }).count
        let cards = edition.stories.prefix(3).map {
            Card(
                tag: "\($0.topic.title) / \($0.infoType.title)",
                merge: "\($0.postCount) → 1",
                title: $0.title,
                posts: $0.postCount
            )
        }
        return BuildingPlan(
            totalPosts: total,
            sourceCount: sources,
            keptPosts: kept,
            hiddenPosts: total - kept,
            storyCount: edition.stories.count,
            lawCount: edition.laws.count,
            cards: Array(cards)
        )
    }

    // MARK: полоски-посты

    var slipCount: Int { min(totalPosts, Self.maxSlips) }

    /// Сколько полосок остаётся после фильтров.
    var keptSlipCount: Int {
        guard totalPosts > 0 else { return 0 }
        return min(slipCount, Int((Double(slipCount) * Double(keptPosts) / Double(totalPosts)).rounded()))
    }

    /// Шаг, взаимно простой с числом полосок: «отфильтрованные» размазаны по сетке, а не идут подряд.
    private var stride: Int {
        let n = max(slipCount, 1)
        return [17, 13, 11, 7, 5, 3, 1].first { gcd($0, n) == 1 } ?? 1
    }

    func isFiltered(slip i: Int) -> Bool {
        guard slipCount > 0 else { return false }
        return (i * stride) % slipCount < slipCount - keptSlipCount
    }

    /// В какую стопку уезжает `j`-я оставшаяся полоска (пропорционально числу постов в сюжетах).
    func cluster(forKept j: Int) -> Int {
        guard !cards.isEmpty else { return 0 }
        let weights = cards.map { max($0.posts, 1) }
        let sum = Double(weights.reduce(0, +))
        var acc = 0.0
        for (i, w) in weights.enumerated() {
            acc += Double(w) / sum * Double(max(keptSlipCount, 1))
            if Double(j) < acc.rounded(.down) || i == weights.count - 1 { return i }
        }
        return cards.count - 1
    }

    private func gcd(_ a: Int, _ b: Int) -> Int { b == 0 ? a : gcd(b, a % b) }

    // MARK: подписи шагов

    var stepTexts: [String] {
        [
            "Собрали \(totalPosts) \(RuFormat.plural(totalPosts, one: "пост", few: "поста", many: "постов")) из \(sourceCount) \(RuFormat.plural(sourceCount, one: "источника", few: "источников", many: "источников"))",
            hiddenPosts > 0 ? "Скрыли \(hiddenPosts): реклама, стоп-темы, слухи" : "Лишнего не нашлось",
            "Склеили повторы в \(storyCount) \(RuFormat.plural(storyCount, one: "сюжет", few: "сюжета", many: "сюжетов"))",
            lawCount > 0
                ? "Нашли \(lawCount) \(RuFormat.plural(lawCount, one: "изменение", few: "изменения", many: "изменений")) в законах для тебя"
                : "Изменений в законах для тебя пока нет",
        ]
    }
}
