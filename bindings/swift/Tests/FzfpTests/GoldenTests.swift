import XCTest
@testable import Fzfp

final class GoldenTests: XCTestCase {
    struct Fixture: Decodable {
        let abi_version: Int
        let core_revision: String
        let vectors: [Vector]
    }
    struct Vector: Decodable {
        let name: String
        let request: Request
        let expected: Expected
    }
    struct Request: Decodable {
        let operation: String
        let text: String
        let query: String
        let candidates: [String]?
        let options: Options
    }
    struct Expected: Decodable {
        let match: Match?
        let results: [Hit]?
    }
    struct Hit: Decodable {
        let index: Int
        let match: Match
        enum Keys: String, CodingKey { case index }
        init(from decoder: Decoder) throws {
            index = try decoder.container(keyedBy: Keys.self).decode(Int.self, forKey: .index)
            match = try Match(from: decoder)
        }
    }
    func testGoldenVectors() throws {
        let fallback = URL(fileURLWithPath: #filePath).deletingLastPathComponent().deletingLastPathComponent()
            .deletingLastPathComponent().deletingLastPathComponent().appendingPathComponent("testdata/golden.json")
        let url = ProcessInfo.processInfo.environment["FZFP_GOLDEN"].map { URL(fileURLWithPath: $0) } ?? fallback
        let fixture = try JSONDecoder().decode(Fixture.self, from: Data(contentsOf: url))
        let version = try Fzfp.version()
        XCTAssertEqual(version.abiVersion, fixture.abi_version)
        XCTAssertEqual(version.coreRevision, fixture.core_revision)
        for vector in fixture.vectors {
            let r = vector.request
            if r.operation == "match" {
                XCTAssertEqual(try Fzfp.match(r.text, query: r.query, options: r.options), vector.expected.match, vector.name)
            } else {
                let actual = try Fzfp.filter(r.candidates ?? [], query: r.query, options: r.options)
                let expected = vector.expected.results ?? []
                XCTAssertEqual(actual.map(\.index), expected.map(\.index), vector.name)
                XCTAssertEqual(actual.map(\.match), expected.map(\.match), vector.name)
            }
        }
        print("Verified \(fixture.vectors.count) shared golden vectors")
    }
    func testUnicodeRanges() throws {
        let m = try Fzfp.match("😀中国e\u{301}", query: "zg")
        XCTAssertEqual(m.positions, [1,2])
        XCTAssertEqual(m.utf16Ranges(in: "😀中国e\u{301}"), [NSRange(location:2,length:1), NSRange(location:3,length:1)])
        XCTAssertEqual(try Fzfp.match("😀", query: "😀").utf16Ranges(in:"😀"), [NSRange(location:0,length:2)])
    }
    func testConcurrentSchemes() throws {
        let group = DispatchGroup()
        let lock = NSLock()
        var failures = 0
        for scheme in [Scheme.default, .path, .history] {
            let expected = try Fzfp.match("/马超", query:"machao", options:Options(scheme:scheme))
            for _ in 0..<20 {
                group.enter()
                DispatchQueue.global().async {
                    defer { group.leave() }
                    if (try? Fzfp.match("/马超", query:"machao", options:Options(scheme:scheme))) != expected {
                        lock.lock(); failures += 1; lock.unlock()
                    }
                }
            }
        }
        group.wait()
        XCTAssertEqual(failures, 0)
    }
}
