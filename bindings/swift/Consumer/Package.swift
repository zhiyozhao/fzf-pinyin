// swift-tools-version: 5.9
import Foundation
import PackageDescription

let dependency: Package.Dependency
if let path = ProcessInfo.processInfo.environment["FZFP_PACKAGE_PATH"] {
    dependency = .package(name: "fzf-pinyin", path: path)
} else {
    dependency = .package(url: "https://github.com/zhiyozhao/fzf-pinyin", exact: "0.2.1")
}
let package = Package(name: "Consumer", platforms: [.macOS(.v13)], dependencies: [dependency], targets: [
    .testTarget(name: "ConsumerTests", dependencies: [.product(name: "Fzfp", package: "fzf-pinyin")])
])
