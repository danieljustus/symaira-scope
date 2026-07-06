import SwiftUI
import SymscopeFeature

@main
struct SymscopeApp: App {
    var body: some Scene {
        Window("Symscope", id: "main") {
            SymscopeModuleView()
                .frame(minWidth: 800, minHeight: 600)
        }
        .windowStyle(.hiddenTitleBar)
    }
}
