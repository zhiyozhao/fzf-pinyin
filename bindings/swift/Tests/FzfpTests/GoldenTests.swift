import XCTest
@testable import Fzfp

final class GoldenTests: XCTestCase {
    func testCLITieHeavyOracle() throws {
        guard let directory = ProcessInfo.processInfo.environment["FZFP_CLI_ORACLE_DIR"] else {
            throw XCTSkip("scripts/test-bindings.sh generates the mandatory live CLI oracle")
        }
        struct Oracle: Decodable { let vectors: [Vector] }
        var count = 0
        for name in ["default", "path", "history"] {
            let url = URL(fileURLWithPath: directory).appendingPathComponent(name + ".json")
            for v in try JSONDecoder().decode(Oracle.self, from: Data(contentsOf: url)).vectors {
                count += 1
                let r = v.request, expected = v.expected.results ?? [], texts = v.request.candidates ?? []
                let actual = try Fzfp.filter(texts, query: r.query, options: r.options)
                XCTAssertEqual(actual.map(\.index), expected.map(\.index), v.name)
                XCTAssertEqual(actual.map(\.match), expected.map(\.match), v.name)
                let bytes: [UInt8] = actual.flatMap { Array($0.text.utf8) + [UInt8(0)] }
                let expectedBytes: [UInt8] = expected.flatMap { Array(texts[$0.index].utf8) + [UInt8(0)] }
                XCTAssertEqual(bytes, expectedBytes, v.name)
                for workers in [1, 4] {
                    let index = try Index(texts, options: r.options, workers: workers)
                    for (limit, positions): (Int?, Bool) in [(nil, true), (50, true), (50, false)] {
                        let hits = try index.find(r.query, limit: limit, positions: positions)
                        let want = Array(expected.prefix(limit ?? expected.count))
                        XCTAssertEqual(hits.map(\.index), want.map(\.index), v.name)
                        XCTAssertEqual(hits.map(\.score), want.map { $0.match.score }, v.name)
                        if positions { XCTAssertEqual(hits.map { $0.alignment! }, want.map(\.match), v.name) }
                    }
                    try index.close()
                }
            }
        }
        XCTAssertEqual(count, 1260)
        print("Verified \(count) live CLI tie-heavy datasets (Swift)")
    }
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
        init(index: Int, match: Match) { self.index = index; self.match = match }
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
            let texts = r.operation == "match" ? [r.text] : (r.candidates ?? [])
            let expected: [Hit] = r.operation == "match" ? ((vector.expected.match?.matched ?? false) ? [Hit(index: 0, match: vector.expected.match!)] : []) : (vector.expected.results ?? [])
            for workers in [1, 4] {
                let index = try Index(texts, options: r.options, workers: workers)
                for k: Int? in [nil, 0, 1, 7, 50] {
                    let want = Array(expected.prefix(k ?? expected.count))
                    for positions in [false, true] {
                        let hits = try index.find(r.query, limit: k, positions: positions)
                        XCTAssertEqual(hits.map(\.index), want.map(\.index), vector.name)
                        XCTAssertEqual(hits.map(\.score), want.map { $0.match.score }, vector.name)
                        if positions && r.options.positions { XCTAssertEqual(hits.map { $0.alignment! }, want.map(\.match), vector.name) }
                    }
                }
                try index.close(); try index.close()
                XCTAssertThrowsError(try index.find(r.query))
            }
        }
        print("Verified \(fixture.vectors.count) shared golden vectors")
    }
    func testIndexErrors() throws {
        XCTAssertThrowsError(try Index([], workers: 65))
        let index = try Index(["北京"])
        XCTAssertThrowsError(try index.find("bj", limit: -1))
        XCTAssertEqual(try index.find("bj", positions: false).first?.alignment, nil)
    }
    func testSameIndexConcurrencyAndClose() throws {
        let index = try Index(Array(repeating: "北京", count: 1100))
        let lock = NSLock()
        var failures = 0
        DispatchQueue.concurrentPerform(iterations: 20) { _ in
            do {
                let hits = try index.find("bj", limit: 7, positions: true)
                if hits.map(\.index) != Array(0..<7) { lock.lock(); failures += 1; lock.unlock() }
            } catch { lock.lock(); failures += 1; lock.unlock() }
        }
        XCTAssertEqual(failures, 0)
        try index.close(); try index.close()
        XCTAssertThrowsError(try index.find("bj"))
        let raced = try Index(Array(repeating: "北京", count: 1100))
        DispatchQueue.concurrentPerform(iterations: 21) { i in
            do {
                if i == 10 { try raced.close() }
                else if try raced.find("bj", limit: 7).map(\.index) != Array(0..<7) {
                    lock.lock(); failures += 1; lock.unlock()
                }
            } catch FzfpError.core(let status, _) where status == 3 {
                // close acquired the wrapper lock before this query.
            } catch { lock.lock(); failures += 1; lock.unlock() }
        }
        XCTAssertEqual(failures, 0)
        try raced.close()
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
