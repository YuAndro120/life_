import Foundation
import Security

/// Снимок профиля и настроек для восстановления после переустановки.
/// Хранится в связке ключей iOS (Keychain): её содержимое шифруется системой, переживает удаление приложения
/// и, если включена связка ключей iCloud, попадает на новый телефон со сквозным шифрованием. На наш сервер ничего не уходит.
struct ProfileBackup: Codable, Equatable, Sendable {
    static let currentVersion = 1

    var version = ProfileBackup.currentVersion
    var savedAt: Date
    var profile: UserProfile
    var preferences: FilterPreferences
    var theme: String
    var autoDusk: Bool
    var schedule: String
    var morningMinutes: Int
    var eveningMinutes: Int

    /// Скрытые сюжеты в копию не берём: их номера действуют недолго, а размер копии растёт.
    static func make(profile: UserProfile, preferences: FilterPreferences, settings: FilterSettings, at date: Date) -> ProfileBackup {
        var prefs = preferences
        prefs.hiddenStories = []
        return ProfileBackup(
            savedAt: date, profile: profile, preferences: prefs, theme: settings.theme, autoDusk: settings.autoDusk,
            schedule: settings.schedule, morningMinutes: settings.morningMinutes, eveningMinutes: settings.eveningMinutes
        )
    }

    func encoded() -> Data? {
        let encoder = JSONEncoder()
        encoder.dateEncodingStrategy = .iso8601
        return try? encoder.encode(self)
    }

    static func decode(_ data: Data) -> ProfileBackup? {
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .iso8601
        guard let backup = try? decoder.decode(ProfileBackup.self, from: data), backup.version <= currentVersion else { return nil }
        return backup
    }
}

protocol BackupStoring: Sendable {
    func read() -> Data?
    @discardableResult func write(_ data: Data) -> Bool
    func delete()
}

/// Ничего не хранит: для тестов и режима без копии.
struct NoopBackupStore: BackupStoring {
    func read() -> Data? { nil }
    func write(_ data: Data) -> Bool { false }
    func delete() {}
}

/// В памяти процесса: для тестов и отладочных запусков, чтобы не трогать настоящую связку ключей.
final class InMemoryBackupStore: BackupStoring, @unchecked Sendable {
    private let lock = NSLock()
    private var data: Data?
    init(_ data: Data? = nil) { self.data = data }
    func read() -> Data? { lock.withLock { data } }
    func write(_ data: Data) -> Bool { lock.withLock { self.data = data }; return true }
    func delete() { lock.withLock { data = nil } }
}

/// Keychain: один элемент «пароль» с JSON. Доступен после первой разблокировки телефона; синхронизируется через iCloud Keychain.
struct KeychainBackupStore: BackupStoring {
    private let service = "ru.andronov.shtil.backup"
    private let account = "profile"

    private var match: [String: Any] {
        [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
            kSecAttrSynchronizable as String: kSecAttrSynchronizableAny,
        ]
    }

    func read() -> Data? {
        var query = match
        query[kSecReturnData as String] = true
        query[kSecMatchLimit as String] = kSecMatchLimitOne
        var out: CFTypeRef?
        guard SecItemCopyMatching(query as CFDictionary, &out) == errSecSuccess else { return nil }
        return out as? Data
    }

    func write(_ data: Data) -> Bool {
        let update: [String: Any] = [kSecValueData as String: data]
        if SecItemUpdate(match as CFDictionary, update as CFDictionary) == errSecSuccess { return true }
        var add = match
        add[kSecAttrSynchronizable as String] = true
        add[kSecAttrAccessible as String] = kSecAttrAccessibleAfterFirstUnlock
        add[kSecValueData as String] = data
        return SecItemAdd(add as CFDictionary, nil) == errSecSuccess
    }

    func delete() {
        SecItemDelete(match as CFDictionary)
    }
}

// Старые копии не должны ломаться, когда в профиле или настройках появляются новые поля: недостающее берётся по умолчанию.
extension UserProfile {
    enum CodingKeys: String, CodingKey { case gender, age, work, housing, occupations, sells, drives, regionCode }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.init(
            gender: try c.decodeIfPresent(Gender.self, forKey: .gender),
            age: try c.decodeIfPresent(AgeBracket.self, forKey: .age),
            work: try c.decodeIfPresent(Set<Work>.self, forKey: .work) ?? [],
            housing: try c.decodeIfPresent(Set<Housing>.self, forKey: .housing) ?? [],
            occupations: try c.decodeIfPresent(Set<Occupation>.self, forKey: .occupations) ?? [],
            sells: try c.decodeIfPresent(Set<Sells>.self, forKey: .sells) ?? [],
            drives: try c.decodeIfPresent(Bool.self, forKey: .drives),
            regionCode: try c.decodeIfPresent(String.self, forKey: .regionCode)
        )
    }
}

extension FilterPreferences {
    enum CodingKeys: String, CodingKey {
        case calmMode, infoTypes, heavyMode, maxHeavy, stopTopics, hideAds, countries, interests, onlyInterests
        case mutedSources, hiddenStories, hideWar, hideOtherRegions, blockedWords, storyLimit
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        var p = FilterPreferences.default
        p.calmMode = try c.decodeIfPresent(Bool.self, forKey: .calmMode) ?? p.calmMode
        p.infoTypes = try c.decodeIfPresent(Set<InfoType>.self, forKey: .infoTypes) ?? p.infoTypes
        p.heavyMode = try c.decodeIfPresent(HeavyMode.self, forKey: .heavyMode) ?? p.heavyMode
        p.maxHeavy = try c.decodeIfPresent(Int.self, forKey: .maxHeavy) ?? p.maxHeavy
        p.stopTopics = try c.decodeIfPresent(Set<Topic>.self, forKey: .stopTopics) ?? p.stopTopics
        p.hideAds = try c.decodeIfPresent(Bool.self, forKey: .hideAds) ?? p.hideAds
        p.countries = try c.decodeIfPresent(Set<String>.self, forKey: .countries) ?? p.countries
        p.interests = try c.decodeIfPresent(Set<Topic>.self, forKey: .interests) ?? p.interests
        p.onlyInterests = try c.decodeIfPresent(Bool.self, forKey: .onlyInterests) ?? p.onlyInterests
        p.mutedSources = try c.decodeIfPresent(Set<String>.self, forKey: .mutedSources) ?? p.mutedSources
        p.hiddenStories = try c.decodeIfPresent(Set<String>.self, forKey: .hiddenStories) ?? p.hiddenStories
        p.hideWar = try c.decodeIfPresent(Bool.self, forKey: .hideWar) ?? p.hideWar
        p.hideOtherRegions = try c.decodeIfPresent(Bool.self, forKey: .hideOtherRegions) ?? p.hideOtherRegions
        p.blockedWords = try c.decodeIfPresent([String].self, forKey: .blockedWords) ?? p.blockedWords
        p.storyLimit = try c.decodeIfPresent(Int.self, forKey: .storyLimit) ?? p.storyLimit
        self = p
    }
}
