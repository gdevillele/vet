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
    ],
    targets: [
        .target(
            name: "VetCore",
            dependencies: ["Yams"]
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
