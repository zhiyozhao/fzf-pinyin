import Foundation
import Fzfp

let args = CommandLine.arguments
guard args.count == 8 else { fatalError("usage: fzfp-benchmark corpus query scheme repetitions mode workers positions") }
var candidates = try String(contentsOfFile: args[1], encoding: .utf8).components(separatedBy: "\n")
if candidates.last == "" { candidates.removeLast() }
let options = Options(scheme: Scheme(rawValue: args[3])!, positions: args[7] == "true")
let mode = args[5], workers = Int(args[6])!, positions = args[7] == "true"
var times: [Double] = [], phases: [[String: Double]] = [], indexes: [Int] = [], scores: [Int] = []
let createStart = DispatchTime.now().uptimeNanoseconds
let index = mode == "json" || mode == "convenience" ? nil : try Index(candidates, options: options, workers: workers)
let createMS = Double(DispatchTime.now().uptimeNanoseconds-createStart)/1_000_000
func run() throws {
    let start = DispatchTime.now().uptimeNanoseconds
    if mode == "json" || mode == "convenience" {
        let output = mode == "json" ? try Fzfp.filterJSON(candidates, query: args[2], options: options) : try Fzfp.filter(candidates, query: args[2], options: options)
        times.append(Double(DispatchTime.now().uptimeNanoseconds-start)/1_000_000)
        indexes = output.map(\.index); scores = output.map(\.score)
    } else {
        let output = try index!.find(args[2], limit: mode == "top50" ? 50 : nil, positions: positions)
        times.append(Double(DispatchTime.now().uptimeNanoseconds-start)/1_000_000); phases.append(index!.timings)
        indexes = output.map(\.index); scores = output.map(\.score)
    }
}
try run()
let cold = times.removeLast(); phases.removeAll()
for _ in 0..<Int(args[4])! { try run() }
try index?.close()
let object: [String: Any] = ["milliseconds": times, "phases": phases, "create_ms": createMS,
    "cold_ms": cold, "indexes": indexes, "scores": scores]
FileHandle.standardOutput.write(try JSONSerialization.data(withJSONObject: object, options: [.sortedKeys]))
