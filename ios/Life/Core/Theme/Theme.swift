import SwiftUI

/// Токены темы из раздела 12 plan.md. Правило цвета: `accent` только для того, что касается пользователя лично.
struct Theme: Equatable, Sendable {
    let id: ThemeChoice

    let bg: Color
    let ink: Color
    let body: Color
    let muted: Color
    let line: Color
    let accent: Color
    let accentDeep: Color
    let plate: Color
    let card: Color
    let segmentBg: Color
    let chipBorder: Color
    let chipBg: Color
    let buttonBg: Color
    let buttonText: Color

    let cardRadius: CGFloat
    let plateRadius: CGFloat
    let buttonRadius: CGFloat = 18
    let buttonHeight: CGFloat = 58
    let minTapSize: CGFloat = 44

    let fonts: ThemeFonts

    var isDark: Bool { id == .dusk }
    var colorScheme: ColorScheme { isDark ? .dark : .light }
}

extension Theme {
    static let paper = Theme(
        id: .paper,
        bg: Color(hex: 0xF2F1EC), ink: Color(hex: 0x22211E), body: Color(hex: 0x3D3C38),
        muted: Color(hex: 0x66655F), line: Color(hex: 0xD8D6CE),
        accent: Color(hex: 0x2432D0), accentDeep: Color(hex: 0x1D2A9E),
        plate: Color(hex: 0xE4E7F6), card: Color(hex: 0xFBFBFE), segmentBg: Color(hex: 0xE6E4DD),
        chipBorder: Color(hex: 0xC9C7BF), chipBg: Color(hex: 0xFAF9F6),
        buttonBg: Color(hex: 0x22211E), buttonText: Color(hex: 0xF2F1EC),
        cardRadius: 18, plateRadius: 26, fonts: .onest
    )

    static let sage = Theme(
        id: .sage,
        bg: Color(hex: 0xEDF0EA), ink: Color(hex: 0x1E2621), body: Color(hex: 0x36403A),
        muted: Color(hex: 0x5B665E), line: Color(hex: 0xD2DAD1),
        accent: Color(hex: 0x2F6B4F), accentDeep: Color(hex: 0x245740),
        plate: Color(hex: 0xDAE7DD), card: Color(hex: 0xFAFCF9), segmentBg: Color(hex: 0xDFE6DE),
        chipBorder: .clear, chipBg: Color(hex: 0xFAFCF9),
        buttonBg: Color(hex: 0x2F6B4F), buttonText: Color(hex: 0xFAFCF9),
        cardRadius: 22, plateRadius: 30, fonts: .sage
    )

    static let dusk = Theme(
        id: .dusk,
        bg: Color(hex: 0x1C1D22), ink: Color(hex: 0xECEAE4), body: Color(hex: 0xCFCDC7),
        muted: Color(hex: 0xA3A19B), line: Color(hex: 0x34363E),
        accent: Color(hex: 0xA9B3FF), accentDeep: Color(hex: 0xC6CCFF),
        plate: Color(hex: 0x262937), card: Color(hex: 0x30344A), segmentBg: Color(hex: 0x2A2C34),
        chipBorder: Color(hex: 0x34363E), chipBg: Color(hex: 0x262937),
        buttonBg: Color(hex: 0xA9B3FF), buttonText: Color(hex: 0x1C1D22),
        cardRadius: 18, plateRadius: 26, fonts: .onest
    )

    static func of(_ choice: ThemeChoice) -> Theme {
        switch choice {
        case .paper: .paper
        case .sage: .sage
        case .dusk: .dusk
        }
    }
}

extension EnvironmentValues {
    @Entry var theme: Theme = .paper
}
