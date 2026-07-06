// swift-tools-version:6.0
import PackageDescription

let package = Package(
    name: "SymscopeClient",
    platforms: [
        .macOS(.v14),
    ],
    products: [
        .library(name: "SymscopeKit", targets: ["SymscopeKit"]),
        .library(name: "SymscopeFeature", targets: ["SymscopeFeature"]),
    ],
    dependencies: [
        .package(url: "https://github.com/danieljustus/symaira-appkit.git", exact: "0.1.1"),
    ],
    targets: [
        .target(
            name: "SymscopeKit",
            dependencies: [
                .product(name: "SymairaCLIRunner", package: "symaira-appkit"),
                .product(name: "SymairaToolKit", package: "symaira-appkit"),
            ]
        ),
        // Feature module (views + state, no app entry) — consumed by the
        // thin standalone app and the Symaira Hub.
        .target(
            name: "SymscopeFeature",
            dependencies: [
                "SymscopeKit",
                .product(name: "SymairaTheme", package: "symaira-appkit"),
            ]
        ),
        .testTarget(
            name: "SymscopeKitTests",
            dependencies: ["SymscopeKit"]
        ),
    ]
)
