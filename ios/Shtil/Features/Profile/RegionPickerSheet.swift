import SwiftUI

/// Выбор региона: полный список субъектов РФ с поиском по названию и городу.
struct RegionPickerSheet: View {
    @Environment(\.theme) private var theme
    @Environment(\.dismiss) private var dismiss
    @Environment(AppModel.self) private var model
    @State private var query = ""

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack {
                Text("Регион").font(theme.fonts.heading(22, .semibold)).foregroundStyle(theme.ink)
                Spacer()
                Button("Готово") { dismiss() }.font(theme.fonts.body(16, .medium)).foregroundStyle(theme.accent).frame(minHeight: 44)
            }
            .padding(.top, 20)
            TextField("Название или город", text: $query)
                .font(theme.fonts.body(16)).foregroundStyle(theme.ink)
                .textInputAutocapitalization(.never).autocorrectionDisabled()
                .padding(.horizontal, 14).frame(minHeight: 46)
                .background(RoundedRectangle(cornerRadius: 12, style: .continuous).fill(theme.chipBg))
                .overlay(RoundedRectangle(cornerRadius: 12, style: .continuous).strokeBorder(theme.chipBorder, lineWidth: 1))
                .padding(.vertical, 10)
            ScrollView {
                LazyVStack(spacing: 0) {
                    if query.isEmpty { row(title: "Не указывать", code: nil) }
                    ForEach(Region.search(query)) { r in row(title: r.title, code: r.code) }
                }
            }
        }
        .padding(.horizontal, 20)
        .background(theme.bg.ignoresSafeArea())
    }

    private func row(title: String, code: String?) -> some View {
        VStack(spacing: 0) {
            Button {
                var p = model.profile.snapshot
                p.regionCode = code
                model.profile.snapshot = p
                model.save()
                dismiss()
            } label: {
                HStack {
                    Text(title).font(theme.fonts.body(17)).foregroundStyle(theme.ink)
                    Spacer()
                    if model.profile.regionCode == code { Image(systemName: "checkmark").foregroundStyle(theme.accent) }
                }
                .frame(minHeight: 48).contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            Rule()
        }
    }
}
