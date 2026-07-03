import XCTest
@testable import SymscopeKit

final class SymscopeKitTests: XCTestCase {
    
    func testPortDecoder() throws {
        let json = """
        {
            "port": 8080,
            "protocol": "tcp",
            "address": "127.0.0.1",
            "pid": 1234,
            "process": "node"
        }
        """.data(using: .utf8)!
        
        let decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase
        let port = try decoder.decode(Port.self, from: json)
        
        XCTAssertEqual(port.port, 8080)
        XCTAssertEqual(port.protocolString, "tcp")
        XCTAssertEqual(port.address, "127.0.0.1")
        XCTAssertEqual(port.pid, 1234)
        XCTAssertEqual(port.process, "node")
    }

    func testMCPServerDecoder() throws {
        let json = """
        {
            "name": "sqlite-mcp",
            "client": "cursor",
            "transport": "stdio",
            "command": "npx",
            "args": ["-y", "@modelcontextprotocol/server-sqlite"],
            "config_path": "/Users/daniel/.cursor/mcp.json",
            "env": {
                "SQLITE_DB": "test.db"
            },
            "secret_backed": true
        }
        """.data(using: .utf8)!
        
        let decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase
        let server = try decoder.decode(MCPServer.self, from: json)
        
        XCTAssertEqual(server.name, "sqlite-mcp")
        XCTAssertEqual(server.client, "cursor")
        XCTAssertEqual(server.transport, "stdio")
        XCTAssertEqual(server.command, "npx")
        XCTAssertEqual(server.args, ["-y", "@modelcontextprotocol/server-sqlite"])
        XCTAssertEqual(server.configPath, "/Users/daniel/.cursor/mcp.json")
        XCTAssertEqual(server.env?["SQLITE_DB"], "test.db")
        XCTAssertEqual(server.secretBacked, true)
    }

    func testSnapshotDecoder() throws {
        let json = """
        {
            "generated_at": "2026-07-03T09:30:00Z",
            "ports": [
                {
                    "port": 3000,
                    "protocol": "tcp",
                    "address": "*",
                    "pid": 5678,
                    "process": "go"
                }
            ],
            "mcp_servers": [
                {
                    "name": "test-server",
                    "client": "claude-desktop",
                    "transport": "stdio",
                    "command": "python",
                    "config_path": "/Users/daniel/claude_config.json"
                }
            ],
            "containers": [
                {
                    "id": "abc123xyz",
                    "name": "redis",
                    "image": "redis:latest",
                    "ports": [6379]
                }
            ],
            "notes": [
                "Docker not reachable"
            ]
        }
        """.data(using: .utf8)!
        
        let decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase
        let snap = try decoder.decode(Snapshot.self, from: json)
        
        XCTAssertEqual(snap.generatedAt, "2026-07-03T09:30:00Z")
        XCTAssertEqual(snap.ports.count, 1)
        XCTAssertEqual(snap.ports[0].port, 3000)
        XCTAssertEqual(snap.mcpServers.count, 1)
        XCTAssertEqual(snap.mcpServers[0].name, "test-server")
        XCTAssertEqual(snap.containers?.count, 1)
        XCTAssertEqual(snap.containers?[0].name, "redis")
        XCTAssertEqual(snap.notes?.count, 1)
        XCTAssertEqual(snap.notes?[0], "Docker not reachable")
    }
}
