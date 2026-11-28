import XCTest
import Fzfp

final class ConsumerTests: XCTestCase {
    func testReleasedAPI() throws {
        XCTAssertEqual(try Fzfp.version().version, "0.2.1")
        XCTAssertEqual(try Fzfp.match("中国", query: "zg").positions, [0, 1])
        let index = try Index(["👋abc", "a-b-c", "备忘录", "备 忘录"])
        XCTAssertEqual(try index.find("bwl", exact: true).map(\.index), [2])
        let exact = try XCTUnwrap(index.find("abc", positions: true, exact: true).first)
        XCTAssertEqual(exact.index, 0)
        XCTAssertEqual(exact.alignment?.utf16Ranges(in: "👋abc"), (2..<5).map { NSRange(location: $0, length: 1) })
        try index.close()
    }
}
