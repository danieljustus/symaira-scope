import SwiftUI
import SymscopeKit

/// Public root view of the Symscope feature module — the single entry point
/// consumed by the standalone app AND the Symaira Hub (Module Integration
/// Contract, see symaira-hub/AGENTS.md). Owns its own AppState; internal
/// views stay module-private.
public struct SymscopeModuleView: View {
    @State private var appState = AppState()

    public init() {}

    public var body: some View {
        MainView()
            .environment(appState)
            .preferredColorScheme(.dark)
    }
}

/// Contract metadata for hub embedding.
public enum SymscopeModule {
    /// CLI JSON schema version this module expects. 0 until symscope ships
    /// corekit versionkit (`version --json`); bump together with the CLI.
    public static let expectedSchemaVersion = 1
}
