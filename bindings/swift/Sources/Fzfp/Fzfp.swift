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
        guard !positions.isEmpty else { return [] }
        var result: [NSRange] = [], p = 0
        for (index, scalar) in text.unicodeScalars.enumerated() {
            if index == positions[p] {
                guard p < utf16Positions.count else { break }
                result.append(NSRange(location: utf16Positions[p], length: scalar.value > 0xffff ? 2 : 1))
                p += 1
                if p == positions.count { break }
            }
        }
        return result
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
    public var typedABIVersion: Int { 2 }
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
    public static func match(_ text: String, query: String, options: Options = Options(), exact: Bool = false) throws -> Match {
        let index = try Index([text], options: options)
        return try index.exactFind(query, positions: options.positions, contiguous: exact).first?.match ?? Match(
            matched: false, score: 0, start: -1, end: -1, positions: [], utf8Positions: [], utf16Positions: [])
    }
    public static func filter(_ candidates: [String], query: String, options: Options = Options(), exact: Bool = false) throws -> [Result] {
        try Index(candidates, options: options).exactFind(query, positions: options.positions, contiguous: exact)
    }
    /// Original ABI 1 compatibility path, also used as a benchmark baseline.
    public static func filterJSON(_ candidates: [String], query: String, options: Options = Options()) throws -> [Result] {
        let response = try call(Request(operation: "filter", candidates: candidates, query: query, options: options), as: Response.self)
        return (response.results ?? []).map { Result(index: $0.index, text: candidates[$0.index], match: $0.match) }
    }
}

public struct Hit: Equatable, Sendable {
    public let index: Int
    public let score: Int
    public let alignment: Match?
    public var positions: [Int] { alignment?.positions ?? [] }
}

/// Owned immutable corpus. Calls and close are serialized; this class is not
/// Sendable. An application may own it in a background actor. Cancellation of
/// a Swift Task does not interrupt the synchronous native query.
public final class Index {
    public let candidates: [String]
    public private(set) var timings: [String: Double] = [:]
    private let lock = NSLock()
    private var handle: UInt64 = 0

    public init(_ candidates: [String], options: Options = Options(), workers: Int = 0) throws {
        guard fzfp_typed_abi_version() == 2 else { throw FzfpError.abiMismatch }
        guard (0...64).contains(workers) else { throw FzfpError.core(status: 1, message: "workers must be 0..64") }
        self.candidates = candidates
        var bytes: [UInt8] = [], offsets: [UInt64] = [0]
        for text in candidates { bytes.append(contentsOf: text.utf8); offsets.append(UInt64(bytes.count)) }
        let scheme: UInt32 = options.scheme == .default ? 0 : (options.scheme == .path ? 1 : 2)
        var native = fzfp_options(abi_version: 2, struct_size: UInt32(MemoryLayout<fzfp_options>.size),
            scheme: scheme, case_sensitive: options.caseSensitive ? 1 : 0, normalize: options.normalize ? 1 : 0,
            workers: UInt32(workers), reserved0: 0, reserved1: 0)
        let status = bytes.withUnsafeMutableBytes { data in
            offsets.withUnsafeMutableBufferPointer { off in
                fzfp_index_create(data.baseAddress?.assumingMemoryBound(to: CChar.self), UInt64(data.count),
                                  off.baseAddress, UInt64(candidates.count), &native, &handle)
            }
        }
        try Self.check(status)
    }

    private static func check(_ status: Int32) throws {
        if status != 0 { throw FzfpError.core(status: status, message: "native typed call failed (invalid input, closed index, or internal error)") }
    }
    public func close() throws {
        lock.lock(); defer { lock.unlock() }
        if handle != 0 { try Self.check(fzfp_index_destroy(handle)); handle = 0 }
    }
    deinit { if handle != 0 { _ = fzfp_index_destroy(handle) } }

    private func rows(_ query: String, limit: Int?, positions: Bool, exact: Bool, contiguous: Bool = false) throws -> [Hit] {
        if let limit, limit < 0 { throw FzfpError.core(status: 1, message: "limit must be nonnegative") }
        lock.lock(); defer { lock.unlock() }
        var native = fzfp_find_options(abi_version: 2, struct_size: UInt32(MemoryLayout<fzfp_find_options>.size),
            limit: Int64(limit ?? -1), flags: (positions ? 1 : 0) | (exact ? 2 : 0) | (contiguous ? 4 : 0), reserved: 0)
        var output: UnsafeMutablePointer<fzfp_results>?
        let status = Array(query.utf8).withUnsafeBytes { data in
            fzfp_index_find(handle, UnsafeMutablePointer(mutating: data.baseAddress?.assumingMemoryBound(to: CChar.self)),
                            UInt64(data.count), &native, &output)
        }
        try Self.check(status)
        guard let output else { throw FzfpError.allocationFailed }
        defer { fzfp_results_free(output) }
        let r = output.pointee
        timings = ["prepare": r.prepare_ms, "scan": r.scan_ms, "select": r.select_ms, "trace": r.trace_ms, "copy": r.copy_ms, "native": r.native_ms]
        return UnsafeBufferPointer(start: r.hits, count: Int(r.count)).map { row in
            let idx = Int(row.index)
            var alignment: Match?
            if positions || exact {
                let pp = (0..<Int(row.position_count)).map { Int(r.positions[Int(row.position_offset) + $0]) }
                var u8: [Int] = [], u16: [Int] = [], b = 0, u = 0, p = 0
                if !pp.isEmpty {
                    for (i, scalar) in candidates[idx].unicodeScalars.enumerated() {
                        if pp[p] == i {
                            u8.append(b); u16.append(u); p += 1
                            if p == pp.count { break }
                        }
                        b += scalar.utf8.count; u += scalar.value > 0xffff ? 2 : 1
                    }
                }
                alignment = Match(matched: true, score: Int(row.score), start: Int(row.start), end: Int(row.end),
                                  positions: pp, utf8Positions: u8, utf16Positions: u16)
            }
            return Hit(index: idx, score: Int(row.score), alignment: alignment)
        }
    }
    public func find(_ query: String, limit: Int? = nil, positions: Bool = false, exact: Bool = false) throws -> [Hit] {
        try rows(query, limit: limit, positions: positions, exact: false, contiguous: exact)
    }
    fileprivate func exactFind(_ query: String, positions: Bool, contiguous: Bool = false) throws -> [Result] {
        try rows(query, limit: nil, positions: positions, exact: true, contiguous: contiguous).map {
            Result(index: $0.index, text: candidates[$0.index], match: $0.alignment!)
        }
    }
}
