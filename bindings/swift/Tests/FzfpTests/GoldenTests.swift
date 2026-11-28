import XCTest
@testable import Fzfp

final class GoldenTests: XCTestCase {
    func testExactCLIQuoteAndAscendingAlignment() throws {
        guard let binary = ProcessInfo.processInfo.environment["FZFP_CLI"] else {
            throw XCTSkip("set FZFP_CLI for exact CLI differential")
        }
        struct Cases: Decodable { let candidates: [String]; let queries: [String] }
        let url = URL(fileURLWithPath: #filePath).deletingLastPathComponent().deletingLastPathComponent()
            .deletingLastPathComponent().deletingLastPathComponent().appendingPathComponent("testdata/exact.json")
        let fixture = try JSONDecoder().decode(Cases.self, from: Data(contentsOf: url))
        let texts = fixture.candidates
        for scheme in [Scheme.default, .path, .history] {
            let options = Options(scheme: scheme)
            for query in fixture.queries {
                let process = Process(), input = Pipe(), output = Pipe()
                process.executableURL = URL(fileURLWithPath: binary)
                process.arguments = ["-i", "--read0", "--print0", "--scheme=" + scheme.rawValue, "--filter='" + query]
                var env = ProcessInfo.processInfo.environment
                env["FZF_DEFAULT_OPTS"] = ""; env["FZF_DEFAULT_OPTS_FILE"] = "/dev/null"
                process.environment = env
                process.standardInput = input; process.standardOutput = output
                try process.run()
                input.fileHandleForWriting.write(Data((texts.joined(separator: "\0") + "\0").utf8))
                try input.fileHandleForWriting.close()
                let bytes = output.fileHandleForReading.readDataToEndOfFile()
                process.waitUntilExit()
                XCTAssertTrue([Int32(0), 1].contains(process.terminationStatus))
                let expected = try Fzfp.filter(texts, query: query, options: options, exact: true)
                XCTAssertEqual(Data(expected.flatMap { Array($0.text.utf8) + [UInt8(0)] }), bytes)
                for workers in [1, 4] {
                    let index = try Index(texts, options: options, workers: workers)
                    for limit: Int? in [nil, 0, 1, 50] {
                        let want = Array(expected.prefix(limit ?? expected.count))
                        for positions in [false, true] {
                            let hits = try index.find(query, limit: limit, positions: positions, exact: true)
                            XCTAssertEqual(hits.map(\.index), want.map(\.index))
                            XCTAssertEqual(hits.map(\.score), want.map { $0.match.score })
                            if positions { XCTAssertEqual(hits.map { $0.alignment! }, want.map(\.match)) }
                        }
                    }
                    try index.close()
                }
                for (i, text) in texts.enumerated() {
                    let m = try Fzfp.match(text, query: query, options: options, exact: true)
                    let want = expected.first { $0.index == i }
                    XCTAssertEqual(m.matched, want != nil)
                    if let want { XCTAssertEqual(m, want.match); XCTAssertEqual(m.positions, m.positions.sorted()) }
                }
            }
        }
        XCTAssertEqual(try Fzfp.match("😀ab", query: "ab").positions, [1, 2])
        XCTAssertEqual(try Fzfp.match("😀ab", query: "ab").utf16Positions, [2, 3])
    }
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
    func testSparseLongAlignmentAndLastScalarWidth() throws {
        for (text, query, expected) in [
            ("👋收货地址" + String(repeating: "synthetic line 你好 0123456789\n", count: 39_999), "👋shdz", [NSRange(location: 0, length: 2)] + (2..<6).map { NSRange(location: $0, length: 1) }),
            (String(repeating: "0", count: 70_000) + "𰻝👋", "biang👋", [NSRange(location: 70_000, length: 2), NSRange(location: 70_002, length: 2)])
        ] {
            let index = try Index([text], options: Options(scheme: .history), workers: 1)
            let hit = try XCTUnwrap(index.find(query, positions: true).first?.alignment)
            XCTAssertEqual(hit.utf16Ranges(in: text), expected)
            XCTAssertEqual(hit.score, try index.find(query).first?.score)
        }
        XCTAssertEqual(try Fzfp.match("tail", query: "absent").utf16Ranges(in: "tail"), [])
    }
    func testExactContiguousMatching() throws {
        for scheme in [Scheme.default, .path, .history] {
            for texts in [["👋abc"], ["👋abc", "👋a-b-c", "备忘录", "备 忘录"]] {
                let index = try Index(texts, options: Options(scheme: scheme))
                let exact = try index.find("abc", positions: true, exact: true)
                XCTAssertEqual(exact.map(\.index), [0])
                XCTAssertEqual(exact[0].alignment?.positions, [1, 2, 3])
                XCTAssertEqual(exact[0].alignment?.utf16Ranges(in: texts[0]), (2..<5).map { NSRange(location: $0, length: 1) })
                XCTAssertEqual(try index.find("abc", exact: true).map(\.score), exact.map(\.score))
                XCTAssertEqual(try index.find("abc", limit: 0, exact: true), [])
                if texts.count > 1 {
                    XCTAssertEqual(try index.find("abc").count, 2)
                    XCTAssertEqual(try index.find("bwl", exact: true).map(\.index), [2])
                }
                XCTAssertEqual(try index.find("absent", exact: true), [])
                try index.close()
            }
        }
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
