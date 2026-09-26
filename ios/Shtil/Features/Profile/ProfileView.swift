import SwiftUI

struct ProfileView: View {
    @Environment(\.theme) private var theme
    @Environment(AppModel.self) private var model

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                if theme.id == .sage {
                    Text("Хранится только на этом телефоне").font(theme.fonts.body(13)).foregroundStyle(theme.muted).padding(.top, 14)
                    ScreenTitle(title: "Профиль", subtitle: "что про тебя важно").padding(.top, 8)
                } else {
                    HStack { Text("Профиль"); Spacer(); Text("Хранится на телефоне") }.metaStyle().padding(.top, 14)
                    ScreenTitle(title: "Профиль").padding(.top, 14)
                }
                Text("Покажем только те законы и изменения, которые касаются тебя.")
                    .font(theme.fonts.body(16)).lineSpacing(3).foregroundStyle(theme.muted).padding(.top, 14)
                ProfileForm().padding(.top, 28)

                SectionTitle(index: "07", title: "Оформление").padding(.top, 40).padding(.bottom, 16)
                ThemePicker()
                SectionTitle(index: "08", title: "Резервная копия").padding(.top, 40).padding(.bottom, 8)
                Text("Профиль и фильтры хранятся в связке ключей iOS: она зашифрована и переживает переустановку. С iCloud-связкой ключей настройки переедут и на новый телефон. На сервер ничего не уходит.")
                    .font(theme.fonts.body(14)).lineSpacing(3).foregroundStyle(theme.muted)
                SwitchRow(
                    title: "Хранить копию настроек",
                    hint: "Выключите, чтобы стереть копию из связки ключей",
                    isOn: Binding(get: { model.settings.backupEnabled }, set: { model.setBackup(enabled: $0) })
                )
                .padding(.top, 6)
                Text("Профиль хранится только на этом телефоне. Сервер не знает, кто ты и что читаешь.")
                    .font(theme.id == .sage ? theme.fonts.body(13) : ThemeFonts.mono(11)).lineSpacing(4)
                    .foregroundStyle(theme.muted).padding(.top, 28)
            }
            .padding(.horizontal, 20)
            .padding(.bottom, 32)
        }
        .scrollIndicators(.hidden)
        .screenBackground(theme)
    }
}
