// swift-tools-version:6.0
import PackageDescription

// Root package: lets consumers (Symaira Hub) pin this repository by tag.
// SPM cannot pin a package that lives in a subdirectory (client/), so this
// mirror re-exposes the client package's library targets from the repo root.
// KEEP IN SYNC with client/Package.swift — that manifest is the source of
// truth for target definitions and dependencies.
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
        .package(url: "https://github.com/danieljustus/symaira-appkit.git", exact: "0.7.0"),
    ],
    targets: [
        .target(
            name: "SymscopeKit",
            dependencies: [
                .product(name: "SymairaCLIRunner", package: "symaira-appkit"),
                .product(name: "SymairaToolKit", package: "symaira-appkit"),
            ],
            path: "client/Sources/SymscopeKit"
        ),
        // Feature module (views + state, no app entry) — consumed by the
        // thin standalone app and the Symaira Hub.
        .target(
            name: "SymscopeFeature",
            dependencies: [
                "SymscopeKit",
                .product(name: "SymairaTheme", package: "symaira-appkit"),
            ],
            path: "client/Sources/SymscopeFeature"
        ),
    ]
)
