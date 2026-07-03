import SwiftUI

public enum AppTheme {
    // Brand Colors
    public static let bgDark = Color(hex: "0D0C0A")
    public static let bgDarker = Color(hex: "070605")
    public static let bgCard = Color(white: 1.0, opacity: 0.04)
    public static let bgCardHover = Color(white: 1.0, opacity: 0.08)
    
    public static let goldPrimary = Color(hex: "E5C397")
    public static let goldSecondary = Color(hex: "F8E6CD")
    public static let icePrimary = Color(hex: "EEDCC4")
    
    public static let textPrimary = Color(hex: "F5F4F0")   // Warm Bone
    public static let textSecondary = Color(hex: "B5AEA5") // Warm Sand Silver
    public static let textMuted = Color(hex: "6E6860")     // Soft Warm Gray
    
    public static let borderGlass = Color(white: 1.0, opacity: 0.05)
    public static let borderGlassHover = Color(hex: "E5C397").opacity(0.18)
    
    public static let accentGlow = Color(hex: "E5C397").opacity(0.05)
    public static let blueGlow = Color(hex: "EEDCC4").opacity(0.08)
}

extension Color {
    init(hex: String) {
        let hex = hex.trimmingCharacters(in: CharacterSet.alphanumerics.inverted)
        var int: UInt64 = 0
        Scanner(string: hex).scanHexInt64(&int)
        let a, r, g, b: UInt64
        switch hex.count {
        case 3: // RGB (12-bit)
            (a, r, g, b) = (255, (int >> 8) * 17, (int >> 4 & 0xF) * 17, (int & 0xF) * 17)
        case 6: // RGB (24-bit)
            (a, r, g, b) = (255, int >> 16, int >> 8 & 0xFF, int & 0xFF)
        case 8: // ARGB (32-bit)
            (a, r, g, b) = (int >> 24, int >> 16 & 0xFF, int >> 8 & 0xFF, int & 0xFF)
        default:
            (a, r, g, b) = (255, 0, 0, 0)
        }
        self.init(
            .sRGB,
            red: Double(r) / 255,
            green: Double(g) / 255,
            blue: Double(b) / 255,
            opacity: Double(a) / 255
        )
    }
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
