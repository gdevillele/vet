// swift-tools-version: 6.0

import PackageDescription

let package = Package(
    name: "VetSwift",
    products: [
        .library(name: "VetCore", targets: ["VetCore"]),
        .executable(name: "vet", targets: ["vet"]),
    ],
    dependencies: [
        .package(url: "https://github.com/jpsim/Yams.git", from: "5.0.0"),
        .package(url: "https://github.com/swiftlang/swift-syntax.git", from: "600.0.1"),
    ],
    targets: [
        .target(
            name: "VetCore",
            dependencies: [
                "Yams",
                .product(name: "SwiftParser", package: "swift-syntax"),
                .product(name: "SwiftSyntax", package: "swift-syntax"),
            ]
        ),
        .executableTarget(
            name: "vet",
            dependencies: ["VetCore"]
        ),
        .testTarget(
            name: "VetCoreTests",
            dependencies: ["VetCore"]
        ),
    ]
)
