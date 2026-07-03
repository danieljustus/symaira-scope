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
    targets: [
        .target(name: "SymscopeKit"),
        .testTarget(
            name: "SymscopeKitTests",
            dependencies: ["SymscopeKit"]
        ),
    ]
)
