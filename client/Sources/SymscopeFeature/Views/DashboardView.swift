import SwiftUI
import SymscopeKit

struct DashboardView: View {
    @Environment(AppState.self) private var appState

    private var columns: [GridItem] {
        [GridItem(.flexible(), spacing: 16), GridItem(.flexible(), spacing: 16)]
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 24) {
                // Header Panel
                HStack(alignment: .center) {
                    VStack(alignment: .leading, spacing: 4) {
                        Text("Machine State Inventory")
                            .font(.title2.bold())
                            .foregroundStyle(AppTheme.textPrimary)
                        Text("Overview of system listening ports, configured MCP instances, and active containers.")
                            .font(.subheadline)
                            .foregroundStyle(AppTheme.textSecondary)
                    }
                    Spacer()
                    
                    Button {
                        Task {
                            await appState.refreshScan()
                        }
                    } label: {
                        HStack {
                            if appState.isScanning {
                                ProgressView()
                                    .controlSize(.small)
                                    .padding(.trailing, 4)
                                Text("Scanning…")
                            } else {
                                Image(systemName: "arrow.clockwise")
                                Text("Scan Now")
                            }
                        }
                    }
                    .buttonStyle(ProminentGlassButtonStyle())
                    .disabled(appState.isScanning)
                }
                .padding(.bottom, 8)

                if let error = appState.lastScanError {
                    HStack {
                        Image(systemName: "exclamationmark.triangle.fill")
                            .foregroundStyle(.red)
                        Text(error)
                            .foregroundStyle(AppTheme.textPrimary)
                    }
                    .glassCard()
                }

                // Grid of status counters
                LazyVGrid(columns: columns, spacing: 16) {
                    // Active Ports
                    StatusCard(
                        title: "Active Ports",
                        value: "\(appState.snapshot?.ports.count ?? 0)",
                        subtitle: "Listening TCP/UDP sockets",
                        iconName: "network",
                        accentColor: AppTheme.icePrimary
                    )

                    // MCP Configurations
                    StatusCard(
                        title: "MCP Servers",
                        value: "\(appState.snapshot?.mcpServers.count ?? 0)",
                        subtitle: "Discovered client configurations",
                        iconName: "cpu",
                        accentColor: AppTheme.goldPrimary
                    )

                    // Containers
                    StatusCard(
                        title: "Docker Containers",
                        value: "\(appState.snapshot?.containers?.count ?? 0)",
                        subtitle: "Running microservices",
                        iconName: "shippingbox",
                        accentColor: .blue
                    )

                    // Conflicts
                    let conflictCount = appState.conflicts.count
                    StatusCard(
                        title: "Port Conflicts",
                        value: "\(conflictCount)",
                        subtitle: conflictCount == 0 ? "All ports clear" : "Overlapping bindings detected",
                        iconName: "exclamationmark.triangle",
                        accentColor: conflictCount == 0 ? .green : .orange
                    )
                }

                // Warnings & Configuration Logs
                VStack(alignment: .leading, spacing: 12) {
                    Label("Discovery Notes & Diagnostics", systemImage: "text.alignleft")
                        .font(.headline)
                        .foregroundStyle(AppTheme.textSecondary)
                    
                    let notes = appState.snapshot?.notes ?? []
                    if notes.isEmpty {
                        Text("No errors or warnings. All local configuration channels loaded cleanly.")
                            .font(.callout)
                            .foregroundStyle(AppTheme.textMuted)
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .glassCard()
                    } else {
                        VStack(alignment: .leading, spacing: 8) {
                            ForEach(notes, id: \.self) { note in
                                HStack(alignment: .top, spacing: 8) {
                                    if note.contains("error") || note.contains("not reachable") {
                                        Image(systemName: "exclamationmark.circle.fill")
                                            .font(.caption)
                                            .foregroundStyle(.orange)
                                            .padding(.top, 2)
                                    } else {
                                        Image(systemName: "info.circle.fill")
                                            .font(.caption)
                                            .foregroundStyle(.blue)
                                            .padding(.top, 2)
                                    }
                                    
                                    Text(note)
                                        .font(.system(.caption2, design: .monospaced))
                                        .foregroundStyle(AppTheme.textSecondary)
                                        .lineLimit(nil)
                                        .fixedSize(horizontal: false, vertical: true)
                                }
                            }
                        }
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .glassCard()
                    }
                }
            }
        }
    }
}

// MARK: - Status Card Component

struct StatusCard: View {
    let title: String
    let value: String
    let subtitle: String
    let iconName: String
    let accentColor: Color

    var body: some View {
        HStack {
            VStack(alignment: .leading, spacing: 8) {
                Text(title)
                    .font(.subheadline)
                    .foregroundStyle(AppTheme.textSecondary)
                
                Text(value)
                    .font(.system(size: 36, weight: .bold, design: .rounded))
                    .foregroundStyle(AppTheme.textPrimary)
                
                Text(subtitle)
                    .font(.caption)
                    .foregroundStyle(AppTheme.textMuted)
            }
            Spacer()
            
            Image(systemName: iconName)
                .font(.system(size: 32))
                .foregroundStyle(accentColor.opacity(0.8))
                .frame(width: 60, height: 60)
                .background(accentColor.opacity(0.06))
                .clipShape(Circle())
                .overlay(
                    Circle().stroke(accentColor.opacity(0.15), lineWidth: 1)
                )
        }
        .interactiveGlassCard()
    }
}
