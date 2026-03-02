import SwiftUI

struct SearchBar: View {
    @EnvironmentObject var mapViewModel: MapViewModel
    @State private var searchText = ""

    var body: some View {
        HStack(spacing: 12) {
            Image(systemName: "magnifyingglass")
                .foregroundStyle(Color.textSecondary)

            TextField("Search shelters...", text: $searchText)
                .foregroundStyle(Color.textPrimary)
                .onChange(of: searchText) { _, newQuery in
                    mapViewModel.search(query: newQuery)
                }
                .onSubmit {
                    mapViewModel.search(query: searchText)
                }

            Button(action: {
                mapViewModel.findNearest()
            }) {
                Image(systemName: "line.3.horizontal")
                    .foregroundStyle(Color.textSecondary)
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .background(.ultraThinMaterial)
        .clipShape(RoundedRectangle(cornerRadius: 12))
    }
}
