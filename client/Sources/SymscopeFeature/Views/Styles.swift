import SwiftUI
import SymairaTheme

public enum AppTheme {
    // Shared brand tokens from symaira-appkit.
    public static let bgDark = SymairaTheme.bgDark
    public static let bgDarker = SymairaTheme.bgDarker
    public static let goldPrimary = SymairaTheme.goldPrimary
    public static let goldSecondary = SymairaTheme.goldSecondary
    public static let icePrimary = SymairaTheme.icePrimary
    public static let textPrimary = SymairaTheme.textPrimary
    public static let textSecondary = SymairaTheme.textSecondary
    public static let textMuted = SymairaTheme.textMuted

    // Scope-specific values that deviate from the shared tokens on purpose
    // (kept local for pixel-identical rendering; revisit in the hub).
    public static let bgCard = Color(white: 1.0, opacity: 0.04)
    public static let bgCardHover = Color(white: 1.0, opacity: 0.08)
    public static let borderGlass = Color(white: 1.0, opacity: 0.05)
    public static let borderGlassHover = SymairaTheme.goldPrimary.opacity(0.18)
    public static let accentGlow = SymairaTheme.goldPrimary.opacity(0.05)
    public static let blueGlow = SymairaTheme.icePrimary.opacity(0.08)
}

// MARK: - View Modifiers

public struct GlassCardModifier: ViewModifier {
    public init() {}
    public func body(content: Content) -> some View {
        content
            .padding()
            .background(
                RoundedRectangle(cornerRadius: 12, style: .continuous)
                    .fill(AppTheme.bgCard)
            )
            .overlay(
                RoundedRectangle(cornerRadius: 12, style: .continuous)
                    .stroke(AppTheme.borderGlass, lineWidth: 1)
            )
    }
}

public struct GlassCardInteractiveModifier: ViewModifier {
    @State private var isHovering = false
    public init() {}
    
    public func body(content: Content) -> some View {
        content
            .padding()
            .background(
                RoundedRectangle(cornerRadius: 12, style: .continuous)
                    .fill(isHovering ? AppTheme.bgCardHover : AppTheme.bgCard)
            )
            .overlay(
                RoundedRectangle(cornerRadius: 12, style: .continuous)
                    .stroke(isHovering ? AppTheme.borderGlassHover : AppTheme.borderGlass, lineWidth: 1)
            )
            .onHover { hovering in
                withAnimation(.easeOut(duration: 0.2)) {
                    isHovering = hovering
                }
            }
    }
}

extension View {
    public func glassCard() -> some View {
        self.modifier(GlassCardModifier())
    }
    
    public func interactiveGlassCard() -> some View {
        self.modifier(GlassCardInteractiveModifier())
    }
}

// MARK: - Ambient Background View

public struct AmbientGlowBackground: View {
    public init() {}
    
    public var body: some View {
        ZStack {
            AppTheme.bgDark
                .ignoresSafeArea()
            
            // Glowing blobs mimicking the web app glow
            GeometryReader { geo in
                ZStack {
                    // Top-left Amber glow
                    RadialGradient(
                        colors: [AppTheme.goldPrimary.opacity(0.12), .clear],
                        center: .topLeading,
                        startRadius: 0,
                        endRadius: geo.size.width * 0.5
                    )
                    .frame(width: geo.size.width * 0.8, height: geo.size.height * 0.8)
                    .position(x: 0, y: 0)
                    
                    // Center-right light sand glow
                    RadialGradient(
                        colors: [AppTheme.blueGlow.opacity(0.7), .clear],
                        center: .center,
                        startRadius: 0,
                        endRadius: geo.size.width * 0.4
                    )
                    .frame(width: geo.size.width * 0.8, height: geo.size.height * 0.8)
                    .position(x: geo.size.width * 0.8, y: geo.size.height * 0.4)
                    
                    // Bottom-left soft gold glow
                    RadialGradient(
                        colors: [AppTheme.accentGlow.opacity(0.8), .clear],
                        center: .bottomLeading,
                        startRadius: 0,
                        endRadius: geo.size.width * 0.4
                    )
                    .frame(width: geo.size.width * 0.8, height: geo.size.height * 0.8)
                    .position(x: geo.size.width * 0.2, y: geo.size.height * 0.9)
                }
                .blur(radius: 60)
            }
        }
    }
}

// MARK: - Glass Button Style

public struct GlassButtonStyle: ButtonStyle {
    public init() {}
    public func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .padding(.horizontal, 16)
            .padding(.vertical, 8)
            .background(
                RoundedRectangle(cornerRadius: 8)
                    .fill(configuration.isPressed ? AppTheme.bgCardHover : AppTheme.bgCard)
            )
            .overlay(
                RoundedRectangle(cornerRadius: 8)
                    .stroke(configuration.isPressed ? AppTheme.borderGlassHover : AppTheme.borderGlass, lineWidth: 1)
            )
            .foregroundColor(AppTheme.textPrimary)
            .scaleEffect(configuration.isPressed ? 0.98 : 1.0)
            .animation(.easeOut(duration: 0.1), value: configuration.isPressed)
    }
}

public struct ProminentGlassButtonStyle: ButtonStyle {
    public init() {}
    public func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .padding(.horizontal, 16)
            .padding(.vertical, 8)
            .background(
                RoundedRectangle(cornerRadius: 8)
                    .fill(AppTheme.goldPrimary.opacity(configuration.isPressed ? 0.75 : 0.85))
            )
            .foregroundColor(AppTheme.bgDarker)
            .font(.body.weight(.medium))
            .scaleEffect(configuration.isPressed ? 0.98 : 1.0)
            .animation(.easeOut(duration: 0.1), value: configuration.isPressed)
    }
}
