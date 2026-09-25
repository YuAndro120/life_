import SwiftUI

/// Гарнитуры темы. Onest и Golos Text вариативные: вес выбирается по именованному начертанию
/// (PostScript-имя), а не модификатором `.weight`, чтобы результат не зависел от версии iOS.
struct ThemeFonts: Equatable, Sendable {
    struct Faces: Equatable, Sendable {
        let regular: String
        let medium: String
        let semibold: String
        let bold: String

        func name(for weight: Font.Weight) -> String {
            switch weight {
            case .bold, .heavy, .black: bold
            case .semibold: semibold
            case .medium: medium
            default: regular
            }
        }
    }

    let body: Faces
    /// Заголовки: в Бумаге и Сумерках Onest, в Шалфее Spectral.
    let heading: Faces
    let headingItalic: String?

    static let onestFaces = Faces(
        regular: "Onest-Regular", medium: "Onest-Medium", semibold: "Onest-SemiBold", bold: "Onest-Bold"
    )
    static let golosFaces = Faces(
        regular: "GolosText-Regular", medium: "GolosText-Regular_Medium",
        semibold: "GolosText-Regular_SemiBold", bold: "GolosText-Regular_Bold"
    )
    static let spectralFaces = Faces(
        regular: "Spectral-Medium", medium: "Spectral-Medium", semibold: "Spectral-SemiBold", bold: "Spectral-SemiBold"
    )

    static let onest = ThemeFonts(body: onestFaces, heading: onestFaces, headingItalic: nil)
    static let sage = ThemeFonts(body: golosFaces, heading: spectralFaces, headingItalic: "Spectral-Italic")

    func body(_ size: CGFloat, _ weight: Font.Weight = .regular) -> Font {
        .custom(body.name(for: weight), size: size)
    }

    func heading(_ size: CGFloat, _ weight: Font.Weight = .semibold, italic: Bool = false) -> Font {
        if italic, let headingItalic { return .custom(headingItalic, size: size) }
        return .custom(heading.name(for: weight), size: size)
    }

    /// Метаданные: IBM Plex Mono. Заглавные и трекинг задаёт модификатор `.metaStyle()`.
    static func mono(_ size: CGFloat = 11, medium: Bool = false) -> Font {
        .custom(medium ? "IBMPlexMono-Medium" : "IBMPlexMono-Regular", size: size)
    }
}

/// PostScript-имена, которые должны быть зарегистрированы через UIAppFonts. Проверяется тестом.
enum FontRegistry {
    static let expectedNames: [String] = {
        let all = [ThemeFonts.onestFaces, ThemeFonts.golosFaces, ThemeFonts.spectralFaces]
            .flatMap { [$0.regular, $0.medium, $0.semibold, $0.bold] }
        return Array(Set(all + ["Spectral-Italic", "IBMPlexMono-Regular", "IBMPlexMono-Medium"])).sorted()
    }()
}

private struct MetaStyle: ViewModifier {
    @Environment(\.theme) private var theme
    let size: CGFloat
    let color: Color?

    func body(content: Content) -> some View {
        content
            .font(theme.id == .sage ? theme.fonts.body(13) : ThemeFonts.mono(size))
            .tracking(theme.id == .sage ? 0 : size * 0.04)
            .textCase(theme.id == .sage ? nil : .uppercase)
            .foregroundStyle(color ?? theme.muted)
    }
}

extension View {
    /// Метка-метаданные. Бумага/Сумерки: моно 11 заглавными; Шалфей: обычный регистр 13pt.
    func metaStyle(size: CGFloat = 11, color: Color? = nil) -> some View {
        modifier(MetaStyle(size: size, color: color))
    }
}
