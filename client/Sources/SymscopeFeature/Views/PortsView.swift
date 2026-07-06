import SwiftUI
import SymscopeKit

struct PortsView: View {
    @Environment(AppState.self) private var appState
    
    @State private var searchText = ""
    @State private var selectedProtocol: ProtocolFilter = .all
    
    enum ProtocolFilter: String, CaseIterable, Identifiable {
        case all = "All"
        case tcp = "TCP"
        case udp = "UDP"
        var id: String { self.rawValue }
    }
    
    var filteredPorts: [SymscopeKit.Port] {
        let ports = appState.snapshot?.ports ?? []
        return ports.filter { port in
            // Search text filter
            let matchesSearch = searchText.isEmpty ||
                String(port.port).contains(searchText) ||
                port.process.localizedCaseInsensitiveContains(searchText)
            
            // Protocol filter
            let matchesProtocol: Bool
            switch selectedProtocol {
            case .all:
                matchesProtocol = true
            case .tcp:
                matchesProtocol = port.protocolString.lowercased() == "tcp"
            case .udp:
                matchesProtocol = port.protocolString.lowercased() == "udp"
            }
            
            return matchesSearch && matchesProtocol
        }
    }
    
    var body: some View {
        VStack(spacing: 16) {
            // Controls
            HStack {
                // Search Bar
                HStack {
                    Image(systemName: "magnifyingglass")
                        .foregroundStyle(AppTheme.textMuted)
                    TextField("Search port number or process name…", text: $searchText)
                        .textFieldStyle(.plain)
                        .foregroundStyle(AppTheme.textPrimary)
                    if !searchText.isEmpty {
                        Button {
                            searchText = ""
                        } label: {
                            Image(systemName: "xmark.circle.fill")
                                .foregroundStyle(AppTheme.textMuted)
                        }
                        .buttonStyle(.plain)
                    }
                }
                .padding(.horizontal, 12)
                .padding(.vertical, 8)
                .background(AppTheme.bgCard)
                .cornerRadius(8)
                .overlay(
                    RoundedRectangle(cornerRadius: 8)
                        .stroke(AppTheme.borderGlass, lineWidth: 1)
                )
                .frame(maxWidth: 300)
                
                Spacer()
                
                // Protocol Filter Picker
                Picker("Protocol", selection: $selectedProtocol) {
                    ForEach(ProtocolFilter.allCases) { proto in
                        Text(proto.rawValue).tag(proto)
                    }
                }
                .pickerStyle(.segmented)
                .frame(width: 180)
            }
            
            // Ports Table
            if filteredPorts.isEmpty {
                VStack(spacing: 16) {
                    Spacer()
                    Image(systemName: "network.slash")
                        .font(.system(size: 40))
                        .foregroundStyle(AppTheme.textMuted)
                    Text("No ports matching criteria.")
                        .font(.body)
                        .foregroundStyle(AppTheme.textSecondary)
                    Spacer()
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .glassCard()
            } else {
                List {
                    // Header row
                    HStack(spacing: 12) {
                        Text("Port").frame(width: 80, alignment: .leading)
                        Text("Protocol").frame(width: 80, alignment: .center)
                        Text("Process").frame(width: 200, alignment: .leading)
                        Text("PID").frame(width: 80, alignment: .leading)
                        Text("Address").frame(width: 120, alignment: .leading)
                        Spacer()
                        Text("Status").frame(width: 100, alignment: .trailing)
                    }
                    .font(.caption.bold())
                    .foregroundStyle(AppTheme.textSecondary)
                    .padding(.vertical, 4)
                    .listRowSeparator(.visible, edges: .bottom)
                    .listRowBackground(Color.clear)
                    
                    ForEach(filteredPorts) { port in
                        let conflict = appState.conflicts.first { $0.port == port.port }
                        HStack(spacing: 12) {
                            // Port Number
                            Text("\(port.port)")
                                .font(.system(.body, design: .monospaced))
                                .frame(width: 80, alignment: .leading)
                                .foregroundStyle(conflict != nil ? Color.orange : AppTheme.textPrimary)
                            
                            // Protocol
                            Text(port.protocolString.uppercased())
                                .font(.caption.bold())
                                .padding(.horizontal, 6)
                                .padding(.vertical, 2)
                                .background(port.protocolString.lowercased() == "tcp" ? Color.blue.opacity(0.1) : Color.purple.opacity(0.1))
                                .foregroundStyle(port.protocolString.lowercased() == "tcp" ? Color.blue : Color.purple)
                                .clipShape(RoundedRectangle(cornerRadius: 4))
                                .frame(width: 80, alignment: .center)
                            
                            // Process Name
                            HStack(spacing: 6) {
                                Image(systemName: "square.stack.3d.up.fill")
                                    .font(.caption)
                                    .foregroundStyle(AppTheme.textMuted)
                                Text(port.process.isEmpty ? "System / Unknown" : port.process)
                                    .foregroundStyle(AppTheme.textPrimary)
                                    .lineLimit(1)
                            }
                            .frame(width: 200, alignment: .leading)
                            
                            // PID
                            Text(port.pid == 0 ? "-" : "\(port.pid)")
                                .font(.system(.body, design: .monospaced))
                                .frame(width: 80, alignment: .leading)
                                .foregroundStyle(AppTheme.textSecondary)
                            
                            // Address
                            Text(port.address)
                                .font(.system(.caption, design: .monospaced))
                                .frame(width: 120, alignment: .leading)
                                .foregroundStyle(AppTheme.textMuted)
                            
                            Spacer()
                            
                            // Conflict Indicator
                            if let conflict = conflict {
                                HStack(spacing: 4) {
                                    Image(systemName: "exclamationmark.triangle.fill")
                                        .font(.caption)
                                    Text(conflict.kind == "mcp-occupied" ? "MCP Overlap" : "Conflict")
                                        .font(.system(size: 10, weight: .bold))
                                }
                                .padding(.horizontal, 8)
                                .padding(.vertical, 4)
                                .background(Color.orange.opacity(0.12))
                                .foregroundStyle(Color.orange)
                                .clipShape(Capsule())
                                .help(conflict.holders.joined(separator: ", "))
                            } else {
                                Text("Active")
                                    .font(.system(size: 10, weight: .semibold))
                                    .padding(.horizontal, 8)
                                    .padding(.vertical, 4)
                                    .background(Color.green.opacity(0.08))
                                    .foregroundStyle(Color.green)
                                    .clipShape(Capsule())
                            }
                        }
                        .padding(.vertical, 6)
                        .listRowBackground(Color.clear)
                        .listRowSeparator(.visible, edges: .bottom)
                    }
                }
                .listStyle(.plain)
                .background(Color.clear)
                .glassCard()
            }
        }
    }
}
