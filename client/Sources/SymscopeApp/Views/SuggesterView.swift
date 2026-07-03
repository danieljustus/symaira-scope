import SwiftUI
import SymscopeKit

struct SuggesterView: View {
    @Environment(AppState.self) private var appState
    
    @State private var count = 3
    @State private var fromPort = 3000
    @State private var toPort = 9999
    
    @State private var suggestedPorts: [Int] = []
    @State private var isCalculating = false
    @State private var errorMessage: String? = nil
    @State private var copiedPort: Int? = nil
    
    var body: some View {
        VStack(alignment: .leading, spacing: 20) {
            VStack(alignment: .leading, spacing: 4) {
                Text("Suggest Free Ports")
                    .font(.title2.bold())
                    .foregroundStyle(AppTheme.textPrimary)
                Text("Locate TCP ports that are currently unbound and available for services on this machine.")
                    .font(.subheadline)
                    .foregroundStyle(AppTheme.textSecondary)
            }
            .padding(.bottom, 8)
            
            // Settings Panel
            VStack(spacing: 16) {
                HStack(spacing: 20) {
                    // Count
                    VStack(alignment: .leading, spacing: 6) {
                        Text("Count")
                            .font(.caption.bold())
                            .foregroundStyle(AppTheme.textSecondary)
                        
                        Picker("", selection: $count) {
                            ForEach(1...10, id: \.self) { num in
                                Text("\(num)").tag(num)
                            }
                        }
                        .frame(width: 80)
                    }
                    
                    // From
                    VStack(alignment: .leading, spacing: 6) {
                        Text("Search Range Start")
                            .font(.caption.bold())
                            .foregroundStyle(AppTheme.textSecondary)
                        
                        TextField("3000", value: $fromPort, format: .number)
                            .textFieldStyle(.roundedBorder)
                            .frame(width: 100)
                    }
                    
                    // To
                    VStack(alignment: .leading, spacing: 6) {
                        Text("Search Range End")
                            .font(.caption.bold())
                            .foregroundStyle(AppTheme.textSecondary)
                        
                        TextField("9999", value: $toPort, format: .number)
                            .textFieldStyle(.roundedBorder)
                            .frame(width: 100)
                    }
                    
                    Spacer()
                    
                    // Action Button
                    Button {
                        Task {
                            await findPorts()
                        }
                    } label: {
                        HStack {
                            if isCalculating {
                                ProgressView().controlSize(.small)
                                    .padding(.trailing, 4)
                            }
                            Text("Find Ports")
                        }
                    }
                    .buttonStyle(ProminentGlassButtonStyle())
                    .disabled(isCalculating)
                    .padding(.top, 16)
                }
            }
            .padding()
            .glassCard()
            
            // Results Display
            VStack(alignment: .leading, spacing: 12) {
                Text("Suggested Available Ports")
                    .font(.headline)
                    .foregroundStyle(AppTheme.textSecondary)
                
                if isCalculating {
                    HStack(spacing: 12) {
                        ProgressView().controlSize(.small)
                        Text("Probing local sockets concurrently…")
                            .foregroundStyle(AppTheme.textSecondary)
                    }
                    .padding()
                    .frame(maxWidth: .infinity, minHeight: 120)
                    .glassCard()
                } else if let error = errorMessage {
                    HStack {
                        Image(systemName: "exclamationmark.triangle.fill")
                            .foregroundStyle(.red)
                        Text(error)
                            .foregroundStyle(AppTheme.textPrimary)
                    }
                    .padding()
                    .frame(maxWidth: .infinity, minHeight: 120)
                    .glassCard()
                } else if suggestedPorts.isEmpty {
                    VStack(spacing: 8) {
                        Image(systemName: "wand.and.stars")
                            .font(.system(size: 32))
                            .foregroundStyle(AppTheme.textMuted)
                        Text("Click 'Find Ports' to query open sockets.")
                            .font(.body)
                            .foregroundStyle(AppTheme.textMuted)
                    }
                    .frame(maxWidth: .infinity, minHeight: 120)
                    .glassCard()
                } else {
                    VStack(alignment: .leading, spacing: 10) {
                        Text("Click any port tag to copy it to clipboard.")
                            .font(.caption)
                            .foregroundStyle(AppTheme.textMuted)
                        
                        FlowLayout(spacing: 12) {
                            ForEach(suggestedPorts, id: \.self) { port in
                                Button {
                                    copyPort(port)
                                } label: {
                                    HStack(spacing: 6) {
                                        Image(systemName: "doc.on.doc.fill")
                                            .font(.caption2)
                                        Text("\(port)")
                                            .font(.system(.title3, design: .monospaced).bold())
                                    }
                                    .padding(.horizontal, 14)
                                    .padding(.vertical, 10)
                                    .background(copiedPort == port ? Color.green.opacity(0.15) : AppTheme.bgCard)
                                    .foregroundStyle(copiedPort == port ? Color.green : AppTheme.goldPrimary)
                                    .overlay(
                                        RoundedRectangle(cornerRadius: 8)
                                            .stroke(copiedPort == port ? Color.green.opacity(0.5) : AppTheme.borderGlass, lineWidth: 1)
                                    )
                                    .cornerRadius(8)
                                }
                                .buttonStyle(.plain)
                            }
                        }
                    }
                    .padding()
                    .frame(maxWidth: .infinity, minHeight: 120, alignment: .leading)
                    .glassCard()
                }
            }
            
            Spacer()
        }
    }
    
    private func findPorts() async {
        isCalculating = true
        errorMessage = nil
        suggestedPorts = []
        copiedPort = nil
        
        do {
            let res = try await appState.suggestPorts(count: count, from: fromPort, to: toPort)
            suggestedPorts = res
            if res.isEmpty {
                errorMessage = "No free ports found in specified range."
            }
        } catch {
            errorMessage = error.localizedDescription
        }
        isCalculating = false
    }
    
    private func copyPort(_ port: Int) {
        let pasteboard = NSPasteboard.general
        pasteboard.declareTypes([.string], owner: nil)
        pasteboard.setString("\(port)", forType: .string)
        
        withAnimation {
            copiedPort = port
        }
        
        // Reset copy notification after 2 seconds
        DispatchQueue.main.asyncAfter(deadline: .now() + 2.0) {
            if self.copiedPort == port {
                withAnimation {
                    self.copiedPort = nil
                }
            }
        }
    }
}
