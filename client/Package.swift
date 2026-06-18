// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "IM",
    platforms: [
        .macOS(.v15),
        .iOS(.v18),
    ],
    products: [
        .executable(name: "IMApp-macOS", targets: ["IMApp-macOS"]),
        .library(name: "IMCore", targets: ["IMCore"]),
    ],
    dependencies: [
        .package(path: "../deps/textual"),
    ],
    targets: [
        .target(
            name: "IMCore",
            path: "Packages/IMCore/Sources/IMCore"
        ),
        .executableTarget(
            name: "IMApp-macOS",
            dependencies: ["IMCore", .product(name: "Textual", package: "textual")],
            path: "macOS/Sources/IMApp",
            resources: []
        ),
        .testTarget(
            name: "IMCoreUnitTests",
            dependencies: ["IMCore"],
            path: "Packages/IMCore/Tests/IMCoreUnitTests"
        ),
        .testTarget(
            name: "IMCoreE2ETests",
            dependencies: ["IMCore"],
            path: "Packages/IMCore/Tests/IMCoreE2ETests"
        ),
    ]
)
