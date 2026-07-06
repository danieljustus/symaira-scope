import SwiftUI
import SymscopeKit

struct DaemonView: View {
    @Environment(AppState.self) private var appState
    @State private var autoScroll = true
    
    var body: some View {
        VStack(spacing: 16) {
            // Status Panel
            HStack(spacing: 16) {
                // Status indicator
                HStack(spacing: 8) {
                    Circle()
                        .fill(statusColor)
                        .frame(width: 10, height: 10)
                        .shadow(color: statusColor.opacity(0.5), radius: 4)
                    
                    Text(statusText)
                        .font(.headline)
                        .foregroundStyle(AppTheme.textPrimary)
                }
                
                Spacer()
                
                // Controls
                if appState.daemon.isRunning {
                    Button {
                        appState.daemon.stop()
                    } label: {
                        Label("Stop Daemon", systemImage: "stop.fill")
                    }
                    .buttonStyle(GlassButtonStyle())
                } else {
                    Button {
                        Task {
                            await appState.daemon.start()
                        }
                    } label: {
                        Label("Start Daemon", systemImage: "play.fill")
                    }
                    .buttonStyle(ProminentGlassButtonStyle())
                    .disabled(appState.daemon.state == .starting)
                }
            }
            .padding()
            .glassCard()
            
            // Console Logs
            VStack(alignment: .leading, spacing: 8) {
                HStack {
                    Label("Console Logs", systemImage: "terminal.fill")
                        .font(.headline)
                        .foregroundStyle(AppTheme.textSecondary)
                    
                    Spacer()
                    
                    Toggle("Auto-scroll", isOn: $autoScroll)
                        .toggleStyle(.checkbox)
                        .font(.caption)
                        .foregroundStyle(AppTheme.textMuted)
                }
                .padding(.horizontal, 4)
                
                VStack {
                    if appState.daemon.logs.isEmpty {
                        VStack(spacing: 8) {
                            Spacer()
                            Text("Daemon is stopped. Start the daemon to listen for MCP sessions.")
                                .font(.caption)
                                .foregroundStyle(AppTheme.textMuted)
                            Spacer()
                        }
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                    } else {
                        ScrollViewReader { proxy in
                            ScrollView([.horizontal, .vertical]) {
                                LazyVStack(alignment: .leading, spacing: 2) {
                                    ForEach(Array(appState.daemon.logs.enumerated()), id: \.offset) { index, line in
                                        Text(line)
                                            .font(.system(.caption2, design: .monospaced))
                                            .foregroundStyle(lineColor(line))
                                            .textSelection(.enabled)
                                            .id(index)
                                    }
                                }
                                .padding(10)
                            }
                            .onChange(of: appState.daemon.logs.count) {
                                if autoScroll {
                                    withAnimation {
                                        proxy.scrollTo(appState.daemon.logs.count - 1, anchor: .bottom)
                                    }
                                }
                            }
                        }
                    }
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .background(Color.black.opacity(0.35))
                .cornerRadius(8)
                .overlay(
                    RoundedRectangle(cornerRadius: 8)
                        .stroke(AppTheme.borderGlass, lineWidth: 1)
                )
            }
        }
    }
    
    private var statusColor: Color {
        switch appState.daemon.state {
        case .stopped: return AppTheme.textMuted
        case .starting: return Color.orange
        case .running: return Color.green
        case .failed: return Color.red
        }
    }
    
    private var statusText: String {
        switch appState.daemon.state {
        case .stopped: return "Stopped"
        case .starting: return "Starting…"
        case .running: return "Serving MCP Stdio"
        case .failed(let err): return "Failed: \(err)"
        }
    }
    
    private func lineColor(_ line: String) -> Color {
        let l = line.lowercased()
        if l.contains("error") || l.contains("fail") {
            return Color.red
        } else if l.contains("warn") {
            return Color.orange
        } else if l.contains("[stderr]") {
            return AppTheme.textSecondary
        } else if l.contains("[stdout]") {
            return AppTheme.goldPrimary
        }
        return AppTheme.textPrimary
    }
}
