import Foundation
import SymairaCLIRunner
import SymairaToolKit

public enum CLIError: Error, LocalizedError, Sendable {
    case binaryNotFound
    case executionFailed(code: Int, message: String)
    case invalidJSON(Error)

    public var errorDescription: String? {
        switch self {
        case .binaryNotFound:
            return "The symscope binary could not be found in the app bundle, PATH, or Homebrew paths. Install it via 'brew install danieljustus/tap/symscope'."
        case .executionFailed(let code, let message):
            return "CLI execution failed with exit code \(code): \(message)"
        case .invalidJSON(let err):
            return "Failed to parse CLI output: \(err.localizedDescription)"
        }
    }
}

public final class CLIClient: Sendable {
    private let decoder: JSONDecoder
    private let runner = CLIRunner(defaultTimeout: 60)
    private let locator: BinaryLocator

    public init() {
        decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase

        // Repo root (../symscope) as last resort keeps the pre-AppKit dev
        // workflow working when running tests without a bundled binary.
        let projectRoot = URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent() // SymscopeKit/
            .deletingLastPathComponent() // Sources/
            .deletingLastPathComponent() // client/
            .deletingLastPathComponent() // repo root
        locator = BinaryLocator(extraDirectories: ["/opt/homebrew/bin", "/usr/local/bin", projectRoot.path])
    }

    public func runCommand(args: [String]) async throws -> Data {
        guard let located = locator.locate("symscope") else {
            throw CLIError.binaryNotFound
        }

        do {
            return try await runner.runChecked(located.url, arguments: args)
        } catch let CLIRunnerError.executionFailed(code, stderr) {
            throw CLIError.executionFailed(code: Int(code), message: stderr)
        }
    }

    private func decode<T: Decodable>(_ type: T.Type, from data: Data) throws -> T {
        do {
            return try decoder.decode(T.self, from: data)
        } catch {
            throw CLIError.invalidJSON(error)
        }
    }

    public func scan(noCache: Bool = true) async throws -> Snapshot {
        var args = ["scan"]
        if noCache {
            args.append("--no-cache")
        }
        return try decode(Snapshot.self, from: try await runCommand(args: args))
    }

    public func suggestPorts(count: Int, from: Int? = nil, to: Int? = nil) async throws -> [Int] {
        var args = ["ports", "suggest", "--count", "\(count)"]
        if let from = from {
            args.append(contentsOf: ["--from", "\(from)"])
        }
        if let to = to {
            args.append(contentsOf: ["--to", "\(to)"])
        }
        let res = try decode(SuggestionResponse.self, from: try await runCommand(args: args))
        return res.free
    }

    public func listConflicts() async throws -> [Conflict] {
        try decode([Conflict].self, from: try await runCommand(args: ["conflicts"]))
    }

    public func listClientConfigs() async throws -> [ClientConfig] {
        try decode([ClientConfig].self, from: try await runCommand(args: ["clients", "list"]))
    }

    public func getVersion() async throws -> String {
        let data = try await runCommand(args: ["version"])
        guard let versionStr = String(data: data, encoding: .utf8) else {
            return "Unknown"
        }
        return versionStr.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    public func clearCache() async throws {
        _ = try await runCommand(args: ["cache", "clear"])
    }
}
