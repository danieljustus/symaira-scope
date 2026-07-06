import SwiftUI
import SymscopeKit

struct ContainersView: View {
    @Environment(AppState.self) private var appState
    
    var isDockerReachable: Bool {
        // If there are notes containing "Docker not reachable", it's unreachable.
        // Otherwise, if we have containers or no docker error note, we assume it's okay.
        let notes = appState.snapshot?.notes ?? []
        return !notes.contains(where: { $0.localizedCaseInsensitiveContains("Docker not reachable") })
    }
    
    var containers: [Container] {
        appState.snapshot?.containers ?? []
    }
    
    var body: some View {
        VStack(spacing: 16) {
            if !isDockerReachable {
                VStack(spacing: 20) {
                    Spacer()
                    Image(systemName: "shippingbox.and.arrow.slant.profile")
                        .font(.system(size: 48))
                        .foregroundStyle(AppTheme.textMuted)
                    
                    Text("Docker Daemon Unreachable")
                        .font(.title3.bold())
                        .foregroundStyle(AppTheme.textPrimary)
                    
                    Text("Symscope could not communicate with Docker. Make sure Docker Desktop or docker-cli is running on your Mac and in your shell path.")
                        .font(.body)
                        .foregroundStyle(AppTheme.textSecondary)
                        .multilineTextAlignment(.center)
                        .padding(.horizontal, 40)
                    
                    if let dockerNote = appState.snapshot?.notes?.first(where: { $0.localizedCaseInsensitiveContains("Docker") }) {
                        Text(dockerNote)
                            .font(.system(.caption2, design: .monospaced))
                            .foregroundStyle(Color.orange)
                            .padding()
                            .background(Color.black.opacity(0.2))
                            .cornerRadius(6)
                    }
                    Spacer()
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .glassCard()
            } else if containers.isEmpty {
                VStack(spacing: 16) {
                    Spacer()
                    Image(systemName: "shippingbox")
                        .font(.system(size: 40))
                        .foregroundStyle(AppTheme.textMuted)
                    Text("No containers running.")
                        .font(.body)
                        .foregroundStyle(AppTheme.textSecondary)
                    Text("Active containers with published ports will appear here.")
                        .font(.caption)
                        .foregroundStyle(AppTheme.textMuted)
                    Spacer()
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .glassCard()
            } else {
                ScrollView {
                    LazyVStack(spacing: 12) {
                        ForEach(containers) { container in
                            ContainerCard(container: container)
                        }
                    }
                }
            }
        }
    }
}

// MARK: - Container Card

struct ContainerCard: View {
    let container: Container
    
    var body: some View {
        HStack(alignment: .top) {
            VStack(alignment: .leading, spacing: 8) {
                // Container Name
                HStack(spacing: 8) {
                    Image(systemName: "cube.box.fill")
                        .foregroundStyle(Color.blue)
                    Text(container.name)
                        .font(.headline)
                        .foregroundStyle(AppTheme.textPrimary)
                    
                    Spacer()
                    
                    Text(container.id.prefix(12))
                        .font(.system(.caption, design: .monospaced))
                        .foregroundStyle(AppTheme.textMuted)
                }
                
                // Image Name
                Text(container.image)
                    .font(.system(.caption, design: .monospaced))
                    .foregroundStyle(AppTheme.textSecondary)
                    .textSelection(.enabled)
                
                // Published Ports
                if let ports = container.ports, !ports.isEmpty {
                    HStack(alignment: .center, spacing: 6) {
                        Text("Published Ports:")
                            .font(.caption.bold())
                            .foregroundStyle(AppTheme.textSecondary)
                        
                        ForEach(ports, id: \.self) { port in
                            Text("\(port)")
                                .font(.system(.caption2, design: .monospaced))
                                .padding(.horizontal, 6)
                                .padding(.vertical, 2)
                                .background(Color.blue.opacity(0.12))
                                .foregroundStyle(Color.blue)
                                .cornerRadius(4)
                        }
                    }
                    .padding(.top, 4)
                } else {
                    Text("No published ports")
                        .font(.caption)
                        .foregroundStyle(AppTheme.textMuted)
                        .padding(.top, 4)
                }
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
