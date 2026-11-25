// swift-tools-version: 5.9
import PackageDescription

// Generate the local binary with: bash scripts/build-bindings.sh swift
// Remote SwiftPM distribution requires a release XCFramework URL and checksum.
let package = Package(
    name: "Fzfp",
    platforms: [.macOS(.v13)],
    products: [.library(name: "Fzfp", targets: ["Fzfp"]),
               .executable(name: "fzfp-benchmark", targets: ["FzfpBenchmark"])],
    targets: [
        .binaryTarget(name: "CFzfp", path: "bindings/swift/.cache/CFzfp.xcframework"),
        .target(name: "Fzfp", dependencies: ["CFzfp"],
                path: "bindings/swift/Sources/Fzfp", linkerSettings: [
            .linkedFramework("CoreFoundation"), .linkedFramework("Security"),
            .linkedLibrary("resolv"), .linkedLibrary("pthread")]),
        .executableTarget(name: "FzfpBenchmark", dependencies: ["Fzfp"],
                          path: "bindings/swift/Sources/FzfpBenchmark"),
        .testTarget(name: "FzfpTests", dependencies: ["Fzfp"],
                    path: "bindings/swift/Tests/FzfpTests")
    ]
)
