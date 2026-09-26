import Foundation

/// Короткая подпись «Лента для: …» из профиля и интересов. Показывается над лентой, чтобы было видно, чем она подстроена.
enum ProfileSummary {
    static func line(profile: UserProfile, preferences: FilterPreferences) -> String? {
        var parts: [String] = []
        if let region = Region.title(for: profile.regionCode) { parts.append(region) }
        for w in UserProfile.Work.allCases where profile.work.contains(w) { parts.append(w.title) }
        if profile.drives == true { parts.append("водитель") }
        parts.append(contentsOf: Topic.allCases.filter { preferences.interests.contains($0) }.prefix(3).map(\.title))
        return parts.isEmpty ? nil : parts.prefix(5).joined(separator: " · ")
    }
}
