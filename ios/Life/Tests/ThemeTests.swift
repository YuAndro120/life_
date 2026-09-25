import SwiftUI
import Testing
import UIKit
@testable import Life

@Suite struct ThemeTests {
    private func date(hour: Int, minute: Int = 0) -> Date {
        var c = DateComponents(year: 2026, month: 9, day: 25, hour: hour, minute: minute)
        c.timeZone = TimeZone(identifier: "Europe/Moscow")
        var cal = Calendar(identifier: .gregorian)
        cal.timeZone = TimeZone(identifier: "Europe/Moscow")!
        return cal.date(from: c)!
    }

    private var cal: Calendar {
        var c = Calendar(identifier: .gregorian)
        c.timeZone = TimeZone(identifier: "Europe/Moscow")!
        return c
    }

    @Test func duskStartsAt1900() {
        #expect(ThemeResolver.resolve(selected: .paper, autoDusk: true, now: date(hour: 18, minute: 59), calendar: cal) == .paper)
        #expect(ThemeResolver.resolve(selected: .paper, autoDusk: true, now: date(hour: 19), calendar: cal) == .dusk)
        #expect(ThemeResolver.resolve(selected: .sage, autoDusk: true, now: date(hour: 23), calendar: cal) == .dusk)
    }

    @Test func duskEndsInTheMorning() {
        #expect(ThemeResolver.resolve(selected: .paper, autoDusk: true, now: date(hour: 5), calendar: cal) == .dusk)
        #expect(ThemeResolver.resolve(selected: .paper, autoDusk: true, now: date(hour: 6), calendar: cal) == .paper)
    }

    @Test func autoDuskOffKeepsSelection() {
        #expect(ThemeResolver.resolve(selected: .sage, autoDusk: false, now: date(hour: 22), calendar: cal) == .sage)
    }

    @Test func themesMatchPlanTokens() {
        #expect(Theme.of(.paper).id == .paper)
        #expect(Theme.sage.cardRadius == 22)
        #expect(Theme.sage.plateRadius == 30)
        #expect(Theme.paper.cardRadius == 18)
        #expect(Theme.dusk.isDark)
        #expect(Theme.paper.colorScheme == .light)
    }

    @Test func colorHexParsesComponents() {
        let resolved = UIColor(Color(hex: 0x2432D0))
        var r: CGFloat = 0, g: CGFloat = 0, b: CGFloat = 0, a: CGFloat = 0
        resolved.getRed(&r, green: &g, blue: &b, alpha: &a)
        #expect(abs(r - 0x24 / 255.0) < 0.01)
        #expect(abs(g - 0x32 / 255.0) < 0.01)
        #expect(abs(b - 0xD0 / 255.0) < 0.01)
    }

    @Test func bundledFontsAreRegistered() {
        for name in FontRegistry.expectedNames {
            #expect(UIFont(name: name, size: 12) != nil, "Шрифт не найден: \(name)")
        }
    }
}
