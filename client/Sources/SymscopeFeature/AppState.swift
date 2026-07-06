import Foundation
import Observation
import SymscopeKit

@Observable
@MainActor
public final class AppState {
    public let client = CLIClient()
    public let daemon = DaemonManager()

    public private(set) var snapshot: Snapshot? = nil
    public private(set) var conflicts: [Conflict] = []
    public private(set) var clientConfigs: [ClientConfig] = []
    public private(set) var isScanning = false
    public private(set) var lastScanError: String? = nil
    public private(set) var lastScanDate: Date? = nil

    public init() {}

    public func refreshScan() async {
        guard !isScanning else { return }
        
        isScanning = true
        lastScanError = nil
        
        do {
            // Concurrently fetch the snapshot, conflicts, and client configs
            async let snapTask = client.scan(noCache: true)
            async let conflictsTask = client.listConflicts()
            async let configsTask = client.listClientConfigs()
            
            let (snap, confs, configs) = try await (snapTask, conflictsTask, configsTask)
            
            self.snapshot = snap
            self.conflicts = confs
            self.clientConfigs = configs
            self.lastScanDate = Date()
        } catch {
            self.lastScanError = error.localizedDescription
            self.snapshot = nil
            self.conflicts = []
            self.clientConfigs = []
        }
        
        isScanning = false
    }

    public func suggestPorts(count: Int, from: Int? = nil, to: Int? = nil) async throws -> [Int] {
        try await client.suggestPorts(count: count, from: from, to: to)
    }

    public func clearCache() async {
        do {
            try await client.clearCache()
        } catch {
            print("Failed to clear cache: \(error)")
        }
    }
}
