import SwiftUI

/// Ответы «что про тебя важно знать». Используется в онбординге и во вкладке «Профиль».
struct ProfileForm: View {
    @Environment(\.theme) private var theme
    @Environment(AppModel.self) private var model
    @State private var showRegions = false

    private var isSage: Bool { theme.id == .sage }

    var body: some View {
        VStack(alignment: .leading, spacing: 28) {
            group("01", "О себе") {
                ForEach(UserProfile.Gender.allCases, id: \.self) { g in
                    ChipButton(title: g.title, isOn: profile.gender == g) { set { $0.gender = $0.gender == g ? nil : g } }
                }
            }
            group("02", "Возраст") {
                ForEach(UserProfile.AgeBracket.allCases, id: \.self) { a in
                    ChipButton(title: a.title, isOn: profile.age == a) { set { $0.age = $0.age == a ? nil : a } }
                }
            }
            group("03", "Работа") {
                ForEach(UserProfile.Work.allCases, id: \.self) { w in
                    ChipButton(title: w.title, isOn: profile.work.contains(w)) {
                        set { if !$0.work.insert(w).inserted { $0.work.remove(w) } }
                    }
                }
            }
            group("04", "Сфера работы") {
                ForEach(UserProfile.Occupation.allCases, id: \.self) { o in
                    ChipButton(title: o.title, isOn: profile.occupations.contains(o)) {
                        set { if !$0.occupations.insert(o).inserted { $0.occupations.remove(o) } }
                    }
                }
            }
            if profile.work.contains(.ip) || profile.work.contains(.selfemployed) {
                group("05", "Что продаёшь") {
                    ForEach(UserProfile.Sells.allCases, id: \.self) { x in
                        ChipButton(title: x.title, isOn: profile.sells.contains(x)) {
                            set { if !$0.sells.insert(x).inserted { $0.sells.remove(x) } }
                        }
                    }
                }
            }
            group("06", "Жильё") {
                ForEach(UserProfile.Housing.allCases, id: \.self) { h in
                    ChipButton(title: h.title, isOn: profile.housing.contains(h)) {
                        set { if !$0.housing.insert(h).inserted { $0.housing.remove(h) } }
                    }
                }
            }
            group("07", "Транспорт") {
                ChipButton(title: "Вожу авто", isOn: profile.drives == true) { set { $0.drives = $0.drives == true ? nil : true } }
                ChipButton(title: "Не вожу", isOn: profile.drives == false) { set { $0.drives = $0.drives == false ? nil : false } }
            }
            VStack(alignment: .leading, spacing: 12) {
                heading("08", "Регион")
                Button { showRegions = true } label: {
                    HStack {
                        Text("Для региональных законов").font(theme.fonts.body(15)).foregroundStyle(theme.ink)
                        Spacer()
                        Text(Region.title(for: profile.regionCode).map { "\($0) →" } ?? "Выбрать →").metaStyle(color: theme.accent)
                    }
                    .frame(minHeight: 44).contentShape(Rectangle())
                }
                .buttonStyle(.plain)
            }
        }
        .sheet(isPresented: $showRegions) { RegionPickerSheet().environment(\.theme, theme).environment(model) }
    }

    private var profile: UserProfile { model.profile.snapshot }

    private func set(_ change: (inout UserProfile) -> Void) {
        var p = model.profile.snapshot
        change(&p)
        model.profile.snapshot = p
        model.save()
    }

    private func heading(_ index: String, _ title: String) -> some View {
        Group {
            if isSage {
                Text(title).font(theme.fonts.body(15, .semibold)).foregroundStyle(theme.accentDeep)
            } else {
                Text("\(index) — \(title)").metaStyle(color: theme.ink)
            }
        }
        .accessibilityAddTraits(.isHeader)
    }

    private func group<Content: View>(_ index: String, _ title: String, @ViewBuilder _ chips: () -> Content) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            heading(index, title)
            FlowLayout(spacing: 8) { chips() }
        }
        .accessibilityElement(children: .contain)
        .accessibilityLabel(title)
    }
}
