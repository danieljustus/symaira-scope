import Foundation

public enum CLIError: Error, LocalizedError, Sendable {
    case binaryNotFound
    case executionFailed(code: Int, message: String)
    case invalidJSON(Error)
    
    public var errorDescription: String? {
        switch self {
        case .binaryNotFound:
            return "The symscope binary could not be found in the app bundle resources or build paths."
        case .executionFailed(let code, let message):
            return "CLI execution failed with exit code \(code): \(message)"
        case .invalidJSON(let err):
            return "Failed to parse CLI output: \(err.localizedDescription)"
        }
    }
}

public final class CLIClient: Sendable {
    private let decoder: JSONDecoder

    public init() {
        decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase
    }

    private func locateBinary() -> URL? {
        // 1. Look in the main App bundle's resources
        if let bundleURL = Bundle.main.url(forResource: "symscope", withExtension: nil) {
            return bundleURL
        }
        
        // 2. Look in the same directory as the executable (useful during local runs/tests)
        let bundleDir = Bundle.main.bundleURL.deletingLastPathComponent()
        let devBinary = bundleDir.appendingPathComponent("symscope")
        if FileManager.default.fileExists(atPath: devBinary.path) {
            return devBinary
        }
        
        // 3. Look in project root folder (fallback for CLI test environments)
        let projectRoot = URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .deletingLastPathComponent()
        let projectBinary = projectRoot.appendingPathComponent("symscope")
        if FileManager.default.fileExists(atPath: projectBinary.path) {
            return projectBinary
        }
        
        return nil
    }

    public func runCommand(args: [String]) async throws -> Data {
        guard let binaryURL = locateBinary() else {
            throw CLIError.binaryNotFound
        }
        
        let process = Process()
        process.executableURL = binaryURL
        process.arguments = args
        
        let pipe = Pipe()
        let errorPipe = Pipe()
        process.standardOutput = pipe
        process.standardError = errorPipe
        
        // Inherit path environment so standard commands like docker are available
        var env = ProcessInfo.processInfo.environment
        if let path = env["PATH"] {
            env["PATH"] = "/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:\(path)"
        } else {
            env["PATH"] = "/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin"
        }
        process.environment = env
        
        return try await withCheckedThrowingContinuation { continuation in
            process.terminationHandler = { proc in
                let exitStatus = proc.terminationStatus
                if exitStatus != 0 {
                    let errData = errorPipe.fileHandleForReading.readDataToEndOfFile()
                    let errMsg = String(data: errData, encoding: .utf8)?
                        .trimmingCharacters(in: .whitespacesAndNewlines) ?? "Unknown error"
                    continuation.resume(throwing: CLIError.executionFailed(code: Int(exitStatus), message: errMsg))
                } else {
                    let outData = pipe.fileHandleForReading.readDataToEndOfFile()
                    continuation.resume(returning: outData)
                }
            }
            
            do {
                try process.run()
            } catch {
                continuation.resume(throwing: error)
            }
        }
    }

    public func scan(noCache: Bool = true) async throws -> Snapshot {
        var args = ["scan"]
        if noCache {
            args.append("--no-cache")
        }
        let data = try await runCommand(args: args)
        do {
            return try decoder.decode(Snapshot.self, from: data)
        } catch {
            throw CLIError.invalidJSON(error)
        }
    }

    public func suggestPorts(count: Int, from: Int? = nil, to: Int? = nil) async throws -> [Int] {
        var args = ["ports", "suggest", "--count", "\(count)"]
        if let from = from {
            args.append(contentsOf: ["--from", "\(from)"])
        }
        if let to = to {
            args.append(contentsOf: ["--to", "\(to)"])
        }
        let data = try await runCommand(args: args)
        do {
            let res = try decoder.decode(SuggestionResponse.self, from: data)
            return res.free
        } catch {
            throw CLIError.invalidJSON(error)
        }
    }

    public func listConflicts() async throws -> [Conflict] {
        let data = try await runCommand(args: ["conflicts"])
        do {
            return try decoder.decode([Conflict].self, from: data)
        } catch {
            throw CLIError.invalidJSON(error)
        }
    }

    public func listClientConfigs() async throws -> [ClientConfig] {
        let data = try await runCommand(args: ["clients", "list"])
        do {
            return try decoder.decode([ClientConfig].self, from: data)
        } catch {
            throw CLIError.invalidJSON(error)
        }
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
