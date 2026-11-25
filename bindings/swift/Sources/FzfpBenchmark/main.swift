import Foundation
import Fzfp

// The driver supplies the same newline-delimited corpus to all three clients.
let args = CommandLine.arguments
guard args.count == 5 else { fatalError("usage: fzfp-benchmark corpus query scheme repetitions") }
let corpus = try String(contentsOfFile: args[1], encoding: .utf8)
var candidates = corpus.components(separatedBy: "\n")
if candidates.last == "" { candidates.removeLast() }
let options = Options(scheme: Scheme(rawValue: args[3])!, positions: false)
var times: [Double] = []
var output: [Result] = []
for _ in 0..<Int(args[4])! {
    let start = DispatchTime.now().uptimeNanoseconds
    output = try Fzfp.filter(candidates, query: args[2], options: options)
    times.append(Double(DispatchTime.now().uptimeNanoseconds - start) / 1_000_000)
}
let object: [String: Any] = ["milliseconds": times, "indexes": output.map(\.index)]
let data = try JSONSerialization.data(withJSONObject: object, options: [.sortedKeys])
FileHandle.standardOutput.write(data)
