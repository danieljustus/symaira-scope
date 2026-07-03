import SwiftUI
import SymscopeKit

enum SidebarSelection: Hashable, CaseIterable {
    case dashboard
    case ports
    case mcpConfig
    case containers
    case suggester
    case daemon
    
    var title: String {
        switch self {
        case .dashboard: return "Dashboard"
        case .ports: return "Listening Ports"
        case .mcpConfig: return "MCP Configs"
        case .containers: return "Docker Containers"
        case .suggester: return "Port Suggester"
        case .daemon: return "Serve Daemon"
        }
    }
    
    var iconName: String {
        switch self {
        case .dashboard: return "gauge.with.needle"
        case .ports: return "network"
        case .mcpConfig: return "cpu"
        case .containers: return "shippingbox"
        case .suggester: return "wand.and.stars"
        case .daemon: return "server.rack"
        }
    }
}

struct MainView: View {
    @Environment(AppState.self) private var appState
    @State private var selection: SidebarSelection? = .dashboard
    
    var body: some View {
        NavigationSplitView {
            List(SidebarSelection.allCases, id: \.self, selection: $selection) { item in
                NavigationLink(value: item) {
                    Label(item.title, systemImage: item.iconName)
                        .font(.body.weight(.medium))
                        .padding(.vertical, 4)
                }
            }
            .listStyle(.sidebar)
            .navigationSplitViewColumnWidth(min: 200, ideal: 220, max: 280)
            .safeAreaInset(edge: .bottom) {
                SidebarBottomView()
            }
        } detail: {
            ZStack {
                AmbientGlowBackground()
                
                Group {
                    switch selection {
                    case .dashboard:
                        DashboardView()
                    case .ports:
                        PortsView()
                    case .mcpConfig:
                        MCPConfigView()
                    case .containers:
                        ContainersView()
                    case .suggester:
                        SuggesterView()
                    case .daemon:
                        DaemonView()
                    case nil:
                        ContentUnavailableView("Select an Item", systemImage: "compass")
                    }
                }
                .padding(24)
            }
            .navigationTitle(selection?.title ?? "Symscope")
        }
        .task {
            // Initial scan when app loads
            await appState.refreshScan()
        }
    }
}

// MARK: - Sidebar Bottom Panel

struct SidebarBottomView: View {
    @Environment(AppState.self) private var appState
    @State private var showVersion = false
    @State private var versionText = "symscope v0.1.2"
    
    var body: some View {
        VStack(spacing: 8) {
            Divider()
                .padding(.horizontal)
            
            HStack {
                VStack(alignment: .leading, spacing: 2) {
                    Text(versionText)
                        .font(.caption2)
                        .foregroundStyle(AppTheme.textMuted)
                    
                    if let lastScan = appState.lastScanDate {
                        Text("Updated: \(formatTime(lastScan))")
                            .font(.system(size: 9))
                            .foregroundStyle(AppTheme.textMuted.opacity(0.8))
                    }
                }
                
                Spacer()
                
                Button {
                    Task {
                        await appState.refreshScan()
                    }
                } label: {
                    if appState.isScanning {
                        ProgressView()
                            .controlSize(.small)
                    } else {
                        Image(systemName: "arrow.clockwise")
                            .imageScale(.medium)
                    }
                }
                .buttonStyle(.plain)
                .disabled(appState.isScanning)
                .help("Refresh scan")
            }
            .padding(.horizontal, 16)
            .padding(.bottom, 12)
        }
        .task {
            if let ver = try? await appState.client.getVersion() {
                versionText = ver
            }
        }
    }
    
    private func formatTime(_ date: Date) -> String {
        let formatter = DateFormatter()
        formatter.timeStyle = .medium
        return formatter.string(from: date)
    }
}
