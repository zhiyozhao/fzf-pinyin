// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "Fzfp",
    platforms: [.macOS(.v13)],
    products: [.library(name: "Fzfp", targets: ["Fzfp"]),
               .executable(name: "fzfp-benchmark", targets: ["FzfpBenchmark"])],
    targets: [
        .binaryTarget(name: "CFzfp", path: ".cache/CFzfp.xcframework"),
        .target(name: "Fzfp", dependencies: ["CFzfp"], linkerSettings: [
            .linkedFramework("CoreFoundation"), .linkedFramework("Security"),
            .linkedLibrary("resolv"), .linkedLibrary("pthread")]),
        .executableTarget(name: "FzfpBenchmark", dependencies: ["Fzfp"]),
        .testTarget(name: "FzfpTests", dependencies: ["Fzfp"])
    ]
)
