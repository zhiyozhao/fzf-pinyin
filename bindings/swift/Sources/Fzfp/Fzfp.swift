import Foundation
import CFzfp

public enum Scheme: String, Codable, Sendable { case `default`, path, history }

public struct Options: Codable, Sendable {
    public var scheme: Scheme
    public var caseSensitive: Bool
    public var normalize: Bool
    public var positions: Bool
    public init(scheme: Scheme = .default, caseSensitive: Bool = false,
                normalize: Bool = true, positions: Bool = true) {
        self.scheme = scheme; self.caseSensitive = caseSensitive
        self.normalize = normalize; self.positions = positions
    }
    enum CodingKeys: String, CodingKey {
        case scheme, normalize, positions
        case caseSensitive = "case_sensitive"
    }
}

public struct Match: Codable, Equatable, Sendable {
    public let matched: Bool
    public let score: Int
    /// Unicode scalar offsets; end is exclusive. No match has start/end -1.
    public let start: Int
    public let end: Int
    public let positions: [Int]
    public let utf8Positions: [Int]
    public let utf16Positions: [Int]
    enum CodingKeys: String, CodingKey {
        case matched, score, start, end, positions
        case utf8Positions = "utf8_positions", utf16Positions = "utf16_positions"
    }
    /// One range per matched Unicode scalar. Adjacent ranges are not merged.
    /// Combining marks are not expanded to grapheme clusters.
    public func utf16Ranges(in text: String) -> [NSRange] {
        let scalars = Array(text.unicodeScalars)
        return zip(positions, utf16Positions).compactMap { index, offset in
            guard scalars.indices.contains(index) else { return nil }
            return NSRange(location: offset, length: scalars[index].value > 0xffff ? 2 : 1)
        }
    }
}

public struct Result: Equatable, Sendable {
    public let index: Int
    public let text: String
    public let match: Match
    public var score: Int { match.score }
    public var positions: [Int] { match.positions }
}

public struct Version: Decodable, Sendable {
    public let version: String
    public let coreRevision: String
    public let abiVersion: Int
    public let dictionaryVersion: String
    enum CodingKeys: String, CodingKey {
        case version
        case coreRevision = "core_revision", abiVersion = "abi_version"
        case dictionaryVersion = "dictionary_version"
    }
}

public enum FzfpError: Error { case abiMismatch, allocationFailed, core(status: Int32, message: String) }

public enum Fzfp {
    private struct Failure: Decodable { let error: String }
    private struct Request: Encodable {
        var operation: String
        var text: String = ""
        var candidates: [String] = []
        var query: String = ""
        var options: Options = Options()
    }
    private struct Response: Decodable {
        let match: Match?
        let results: [Hit]?
        struct Hit: Decodable {
            let index: Int
            let match: Match
            init(from decoder: Decoder) throws {
                let c = try decoder.container(keyedBy: Keys.self)
                index = try c.decode(Int.self, forKey: .index)
                match = try Match(from: decoder)
            }
            enum Keys: String, CodingKey { case index }
        }
    }
    private static func call<T: Decodable>(_ request: Request, as type: T.Type) throws -> T {
        guard fzfp_abi_version() == 1 else { throw FzfpError.abiMismatch }
        let data = try JSONEncoder().encode(request)
        var count = 0
        var status: Int32 = 0
        let pointer = data.withUnsafeBytes { bytes in
            fzfp_call(bytes.baseAddress?.assumingMemoryBound(to: CChar.self), bytes.count, &count, &status)
        }
        guard let pointer else { throw FzfpError.allocationFailed }
        defer { fzfp_free(pointer) }
        let output = Data(bytes: pointer, count: count)
        if status != 0 {
            let failure = try JSONDecoder().decode(Failure.self, from: output)
            throw FzfpError.core(status: status, message: failure.error)
        }
        return try JSONDecoder().decode(type, from: output)
    }
    public static func version() throws -> Version { try call(Request(operation: "version"), as: Version.self) }
    public static func match(_ text: String, query: String, options: Options = Options()) throws -> Match {
        try call(Request(operation: "match", text: text, query: query, options: options), as: Response.self).match!
    }
    public static func filter(_ candidates: [String], query: String, options: Options = Options()) throws -> [Result] {
        let response = try call(Request(operation: "filter", candidates: candidates, query: query, options: options), as: Response.self)
        return (response.results ?? []).map { Result(index: $0.index, text: candidates[$0.index], match: $0.match) }
    }
}
