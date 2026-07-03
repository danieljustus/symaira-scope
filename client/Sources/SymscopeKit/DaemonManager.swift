import Foundation
import Observation

public enum DaemonState: Sendable, Equatable {
    case stopped
    case starting
    case running
    case failed(String)
}

@Observable
@MainActor
public final class DaemonManager {
    public private(set) var state: DaemonState = .stopped
    public private(set) var logs: [String] = []

    public var isRunning: Bool {
        state == .running
    }

    nonisolated(unsafe) private var process: Process?
    private var stdoutFH: FileHandle?
    private var stderrFH: FileHandle?
    
    private let maxLogs = 500

    public init() {}

    nonisolated deinit {
        process?.terminate()
    }

    private func locateBinary() -> URL? {
        if let bundleURL = Bundle.main.url(forResource: "symscope", withExtension: nil) {
            return bundleURL
        }
        
        let bundleDir = Bundle.main.bundleURL.deletingLastPathComponent()
        let devBinary = bundleDir.appendingPathComponent("symscope")
        if FileManager.default.fileExists(atPath: devBinary.path) {
            return devBinary
        }
        
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

    public func start() async {
        guard state != .starting && state != .running else { return }

        state = .starting
        appendLog("[daemon] Starting symscope serve…")

        guard let binaryURL = locateBinary() else {
            state = .failed("symscope binary not found")
            appendLog("[daemon] ERROR: symscope binary not found")
            return
        }

        guard FileManager.default.isExecutableFile(atPath: binaryURL.path) else {
            state = .failed("symscope binary not executable")
            appendLog("[daemon] ERROR: binary not executable at \(binaryURL.path)")
            return
        }

        let proc = Process()
        proc.executableURL = binaryURL
        proc.arguments = ["serve"]

        let stdoutPipe = Pipe()
        let stderrPipe = Pipe()
        proc.standardOutput = stdoutPipe
        proc.standardError = stderrPipe

        var env = ProcessInfo.processInfo.environment
        if let path = env["PATH"] {
            env["PATH"] = "/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:\(path)"
        } else {
            env["PATH"] = "/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin"
        }
        proc.environment = env

        let outFH = stdoutPipe.fileHandleForReading
        let errFH = stderrPipe.fileHandleForReading
        self.stdoutFH = outFH
        self.stderrFH = errFH

        outFH.readabilityHandler = { [weak self] handle in
            let data = handle.availableData
            guard !data.isEmpty, let text = String(data: data, encoding: .utf8) else { return }
            Task { @MainActor [weak self] in
                self?.processOutput(text, source: "stdout")
            }
        }

        errFH.readabilityHandler = { [weak self] handle in
            let data = handle.availableData
            guard !data.isEmpty, let text = String(data: data, encoding: .utf8) else { return }
            Task { @MainActor [weak self] in
                self?.processOutput(text, source: "stderr")
            }
        }

        proc.terminationHandler = { [weak self] proc in
            Task { @MainActor [weak self] in
                guard let self else { return }
                let exitCode = proc.terminationStatus
                if exitCode != 0 {
                    self.state = .failed("Process exited with code \(exitCode)")
                    self.appendLog("[daemon] Process exited with code \(exitCode)")
                } else {
                    self.state = .stopped
                    self.appendLog("[daemon] Process stopped cleanly")
                }
                self.cleanup()
            }
        }

        do {
            try proc.run()
            self.process = proc
            self.state = .running
            appendLog("[daemon] Process started (PID \(proc.processIdentifier))")
        } catch {
            state = .failed(error.localizedDescription)
            appendLog("[daemon] Failed to start: \(error.localizedDescription)")
            cleanup()
        }
    }

    public func stop() {
        guard let proc = process, proc.isRunning else {
            state = .stopped
            return
        }

        appendLog("[daemon] Stopping…")
        proc.terminate()

        Task {
            try? await Task.sleep(for: .seconds(3))
            if proc.isRunning {
                appendLog("[daemon] Force killing process")
                proc.interrupt()
            }
        }
    }

    private func cleanup() {
        stdoutFH?.readabilityHandler = nil
        stderrFH?.readabilityHandler = nil
        stdoutFH = nil
        stderrFH = nil
        process = nil
    }

    private func processOutput(_ text: String, source: String) {
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return }

        for line in trimmed.components(separatedBy: .newlines) {
            let trimmedLine = line.trimmingCharacters(in: .whitespacesAndNewlines)
            guard !trimmedLine.isEmpty else { continue }
            appendLog("[\(source)] \(trimmedLine)")
        }
    }

    private func appendLog(_ message: String) {
        logs.append(message)
        if logs.count > maxLogs {
            logs.removeFirst(logs.count - maxLogs)
        }
    }
}
