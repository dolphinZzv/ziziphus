// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "IMCore",
    platforms: [
        .macOS(.v15),
        .iOS(.v17),
    ],
    products: [
        .library(name: "IMCore", targets: ["IMCore"]),
    ],
    targets: [
        .target(name: "IMCore"),
        .testTarget(name: "IMCoreE2ETests", dependencies: ["IMCore"]),
    ]
)
