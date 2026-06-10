// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "IM",
    platforms: [
        .macOS(.v15),
        .iOS(.v17),
    ],
    products: [
        .executable(name: "IMApp-macOS", targets: ["IMApp-macOS"]),
        .library(name: "IMCore", targets: ["IMCore"]),
    ],
    dependencies: [
        .package(url: "https://github.com/gonzalezreal/swift-markdown-ui", from: "2.4.0"),
    ],
    targets: [
        .target(
            name: "IMCore",
            path: "Packages/IMCore/Sources/IMCore"
        ),
        .executableTarget(
            name: "IMApp-macOS",
            dependencies: ["IMCore", .product(name: "MarkdownUI", package: "swift-markdown-ui")],
            path: "macOS/Sources/IMApp",
            resources: []
        ),
        .testTarget(
            name: "IMCoreE2ETests",
            dependencies: ["IMCore"],
            path: "Packages/IMCore/Tests/IMCoreE2ETests"
        ),
    ]
)
