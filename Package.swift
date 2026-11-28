// swift-tools-version: 5.9
import PackageDescription
import Foundation

// Build source locally with scripts/build-bindings.sh swift, then opt in with
// FZFP_LOCAL_XCFRAMEWORK=1. Release consumers need only SwiftPM, not Go.
let native: Target
if ProcessInfo.processInfo.environment["FZFP_LOCAL_XCFRAMEWORK"] == "1" {
    native = .binaryTarget(name: "CFzfp", path: "bindings/swift/.cache/CFzfp.xcframework")
} else {
    native = .binaryTarget(
        name: "CFzfp",
        url: "https://github.com/zhiyozhao/fzf-pinyin/releases/download/fzfp-v0.2.1/CFzfp.xcframework.zip",
        checksum: "f824e26442dcd704cfdca88657f7f96eaf50ae352bd7a2a2a28e190971848f73")
}
let package = Package(
    name: "Fzfp",
    platforms: [.macOS(.v13)],
    products: [.library(name: "Fzfp", targets: ["Fzfp"]),
               .executable(name: "fzfp-benchmark", targets: ["FzfpBenchmark"])],
    targets: [
        native,
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
