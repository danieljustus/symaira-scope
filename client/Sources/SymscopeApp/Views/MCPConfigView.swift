import SwiftUI
import SymscopeKit

struct MCPConfigView: View {
    @Environment(AppState.self) private var appState
    
    // Group servers by client name
    var serversByClient: [String: [MCPServer]] {
        Dictionary(grouping: appState.snapshot?.mcpServers ?? [], by: { $0.client })
    }
    
    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 24) {
                // Section 1: Client Config Diagnostics
                VStack(alignment: .leading, spacing: 12) {
                    Label("AI Clients Status", systemImage: "checklist")
                        .font(.headline)
                        .foregroundStyle(AppTheme.textSecondary)
                    
                    if appState.clientConfigs.isEmpty {
                        Text("No client configs indexed.")
                            .font(.callout)
                            .foregroundStyle(AppTheme.textMuted)
                            .glassCard()
                    } else {
                        // Display horizontal badges of client configs
                        FlowLayout(spacing: 8) {
                            ForEach(appState.clientConfigs) { cfg in
                                HStack(spacing: 6) {
                                    Circle()
                                        .fill(cfg.present ? Color.green : Color.clear)
                                        .frame(width: 6, height: 6)
                                        .overlay(
                                            Circle().stroke(cfg.present ? Color.clear : AppTheme.textMuted, lineWidth: 1)
                                        )
                                    
                                    Text(formatClientName(cfg.client))
                                        .font(.caption.bold())
                                        .foregroundStyle(cfg.present ? AppTheme.textPrimary : AppTheme.textMuted)
                                }
                                .padding(.horizontal, 10)
                                .padding(.vertical, 6)
                                .background(cfg.present ? AppTheme.goldPrimary.opacity(0.08) : Color.clear)
                                .overlay(
                                    RoundedRectangle(cornerRadius: 6)
                                        .stroke(cfg.present ? AppTheme.goldPrimary.opacity(0.3) : AppTheme.borderGlass, lineWidth: 1)
                                )
                                .cornerRadius(6)
                                .help(cfg.path)
                            }
                        }
                    }
                }
                
                // Section 2: MCP Server Lists grouped by client
                if serversByClient.isEmpty {
                    VStack(spacing: 20) {
                        Spacer()
                        Image(systemName: "cpu.slash")
                            .font(.system(size: 40))
                            .foregroundStyle(AppTheme.textMuted)
                        Text("No MCP servers discovered on this machine.")
                            .font(.body)
                            .foregroundStyle(AppTheme.textSecondary)
                        Text("Configure MCP servers in Claude Desktop, VS Code, Cursor or Cline to register them.")
                            .font(.caption)
                            .foregroundStyle(AppTheme.textMuted)
                            .multilineTextAlignment(.center)
                        Spacer()
                    }
                    .frame(maxWidth: .infinity, minHeight: 250)
                    .glassCard()
                } else {
                    ForEach(serversByClient.keys.sorted(), id: \.self) { clientName in
                        let servers = serversByClient[clientName] ?? []
                        
                        VStack(alignment: .leading, spacing: 14) {
                            // Client Header
                            HStack {
                                Image(systemName: "app.badge.fill")
                                    .foregroundStyle(AppTheme.goldPrimary)
                                Text(formatClientName(clientName))
                                    .font(.title3.bold())
                                    .foregroundStyle(AppTheme.textPrimary)
                                Spacer()
                                Text("\(servers.count) configured")
                                    .font(.caption)
                                    .foregroundStyle(AppTheme.textMuted)
                            }
                            .padding(.bottom, 4)
                            
                            // Servers List
                            VStack(spacing: 12) {
                                ForEach(servers) { server in
                                    MCPServerCard(server: server)
                                }
                            }
                        }
                    }
                }
            }
        }
    }
    
    private func formatClientName(_ raw: String) -> String {
        switch raw.lowercased() {
        case "claude-desktop": return "Claude Desktop"
        case "cursor": return "Cursor"
        case "vscode": return "VS Code"
        case "windsurf": return "Windsurf"
        case "cline": return "Cline"
        case "continue": return "Continue"
        case "goose": return "Goose"
        case "aider": return "Aider"
        case "zed": return "Zed"
        case "claude-code": return "Claude Code"
        case "vscode-workspace": return "VS Code Workspace"
        case "project": return "Project-Local (.mcp.json)"
        default: return raw.capitalized
        }
    }
}

// MARK: - MCP Server Card Component

struct MCPServerCard: View {
    let server: MCPServer
    @State private var isExpanded = false
    
    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            // Summary Header
            Button {
                withAnimation(.easeInOut(duration: 0.2)) {
                    isExpanded.toggle()
                }
            } label: {
                HStack {
                    VStack(alignment: .leading, spacing: 4) {
                        Text(server.name)
                            .font(.headline)
                            .foregroundStyle(AppTheme.textPrimary)
                        
                        HStack(spacing: 8) {
                            Text(server.transport.uppercased())
                                .font(.system(size: 9, weight: .bold))
                                .padding(.horizontal, 4)
                                .padding(.vertical, 2)
                                .background(AppTheme.goldPrimary.opacity(0.12))
                                .foregroundStyle(AppTheme.goldPrimary)
                                .cornerRadius(4)
                            
                            if let command = server.command {
                                Text(command)
                                    .font(.system(.caption, design: .monospaced))
                                    .foregroundStyle(AppTheme.textMuted)
                            } else if let url = server.url {
                                Text(url)
                                    .font(.system(.caption, design: .monospaced))
                                    .foregroundStyle(AppTheme.textMuted)
                            }
                        }
                    }
                    
                    Spacer()
                    
                    if server.secretBacked == true {
                        Image(systemName: "lock.shield.fill")
                            .foregroundStyle(Color.green)
                            .font(.caption)
                            .help("Contains environment secrets")
                            .padding(.trailing, 8)
                    }
                    
                    Image(systemName: isExpanded ? "chevron.up" : "chevron.down")
                        .font(.caption)
                        .foregroundStyle(AppTheme.textMuted)
                }
            }
            .buttonStyle(.plain)
            
            // Expanded Detail View
            if isExpanded {
                Divider()
                    .padding(.vertical, 4)
                
                VStack(alignment: .leading, spacing: 8) {
                    if let args = server.args, !args.isEmpty {
                        DetailRow(label: "Arguments", value: args.joined(separator: " "), code: true)
                    }
                    
                    if let env = server.env, !env.isEmpty {
                        VStack(alignment: .leading, spacing: 4) {
                            Text("Environment Variables")
                                .font(.caption.bold())
                                .foregroundStyle(AppTheme.textSecondary)
                            
                            VStack(alignment: .leading, spacing: 4) {
                                ForEach(env.keys.sorted(), id: \.self) { key in
                                    HStack {
                                        Text(key)
                                            .font(.system(.caption2, design: .monospaced))
                                            .foregroundStyle(AppTheme.goldPrimary)
                                        Spacer()
                                        Text(env[key] ?? "")
                                            .font(.system(.caption2, design: .monospaced))
                                            .foregroundStyle(AppTheme.textSecondary)
                                    }
                                }
                            }
                            .padding(8)
                            .background(Color.black.opacity(0.2))
                            .cornerRadius(6)
                        }
                    }
                    
                    DetailRow(label: "Config File Path", value: server.configPath, code: true)
                }
                .transition(.opacity)
            }
        }
        .padding()
        .background(AppTheme.bgCard)
        .overlay(
            RoundedRectangle(cornerRadius: 8)
                .stroke(AppTheme.borderGlass, lineWidth: 1)
        )
    }
}

struct DetailRow: View {
    let label: String
    let value: String
    var code: Bool = false
    
    var body: some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(label)
                .font(.caption.bold())
                .foregroundStyle(AppTheme.textSecondary)
            
            if code {
                Text(value)
                    .font(.system(.caption, design: .monospaced))
                    .foregroundStyle(AppTheme.textPrimary)
                    .textSelection(.enabled)
                    .padding(6)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .background(Color.black.opacity(0.2))
                    .cornerRadius(6)
            } else {
                Text(value)
                    .font(.callout)
                    .foregroundStyle(AppTheme.textPrimary)
                    .textSelection(.enabled)
            }
        }
    }
}

// MARK: - FlowLayout Helper for horizontal tag lists

struct FlowLayout: Layout {
    var spacing: CGFloat = 8
    
    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        let width = proposal.width ?? .infinity
        var currentX: CGFloat = 0
        var currentY: CGFloat = 0
        var maxHeight: CGFloat = 0
        var totalHeight: CGFloat = 0
        
        for subview in subviews {
            let size = subview.sizeThatFits(.unspecified)
            if currentX + size.width > width {
                currentX = 0
                currentY += maxHeight + spacing
                maxHeight = 0
            }
            maxHeight = max(maxHeight, size.height)
            currentX += size.width + spacing
        }
        
        totalHeight = currentY + maxHeight
        return CGSize(width: width, height: totalHeight)
    }
    
    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        let width = bounds.width
        var currentX: CGFloat = bounds.minX
        var currentY: CGFloat = bounds.minY
        var maxHeight: CGFloat = 0
        
        for subview in subviews {
            let size = subview.sizeThatFits(.unspecified)
            if currentX + size.width > bounds.minX + width {
                currentX = bounds.minX
                currentY += maxHeight + spacing
                maxHeight = 0
            }
            subview.place(at: CGPoint(x: currentX, y: currentY), proposal: ProposedViewSize(size))
            maxHeight = max(maxHeight, size.height)
            currentX += size.width + spacing
        }
    }
}
