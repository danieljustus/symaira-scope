import SwiftUI
import SymscopeKit

@main
struct SymscopeApp: App {
    @State private var appState = AppState()

    var body: some Scene {
        Window("Symscope", id: "main") {
            MainView()
                .environment(appState)
                .preferredColorScheme(.dark)
                .frame(minWidth: 800, minHeight: 600)
        }
        .windowStyle(.hiddenTitleBar)
    }
}
