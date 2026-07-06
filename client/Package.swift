// swift-tools-version:6.0
import PackageDescription

let package = Package(
    name: "SymscopeClient",
    platforms: [
        .macOS(.v14),
    ],
    products: [
        .library(name: "SymscopeKit", targets: ["SymscopeKit"]),
    ],
    dependencies: [
        .package(url: "https://github.com/danieljustus/symaira-appkit.git", exact: "0.1.0"),
    ],
    targets: [
        .target(
            name: "SymscopeKit",
            dependencies: [
                .product(name: "SymairaCLIRunner", package: "symaira-appkit"),
                .product(name: "SymairaToolKit", package: "symaira-appkit"),
            ]
        ),
        .testTarget(
            name: "SymscopeKitTests",
            dependencies: ["SymscopeKit"]
        ),
    ]
)
