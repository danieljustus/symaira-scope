import Foundation

public struct Snapshot: Codable, Sendable {
    public let generatedAt: String
    public let ports: [Port]
    public let mcpServers: [MCPServer]
    public let containers: [Container]?
    public let notes: [String]?

    public init(generatedAt: String, ports: [Port], mcpServers: [MCPServer], containers: [Container]?, notes: [String]?) {
        self.generatedAt = generatedAt
        self.ports = ports
        self.mcpServers = mcpServers
        self.containers = containers
        self.notes = notes
    }
}

public struct Port: Codable, Sendable, Identifiable {
    public var id: String {
        "\(port)/\(protocolString)/\(pid)/\(address)"
    }
    
    public let port: Int
    private let `protocol`: String
    public let address: String
    public let pid: Int
    public let process: String

    public var protocolString: String {
        self.protocol
    }

    enum CodingKeys: String, CodingKey {
        case port
        case `protocol`
        case address
        case pid
        case process
    }

    public init(port: Int, protocolString: String, address: String, pid: Int, process: String) {
        self.port = port
        self.protocol = protocolString
        self.address = address
        self.pid = pid
        self.process = process
    }
}

public struct MCPServer: Codable, Sendable, Identifiable {
    public var id: String {
        "\(client)/\(name)"
    }

    public let name: String
    public let client: String
    public let transport: String
    public let command: String?
    public let args: [String]?
    public let url: String?
    public let configPath: String
    public let env: [String: String]?
    public let secretBacked: Bool?

    public init(name: String, client: String, transport: String, command: String?, args: [String]?, url: String?, configPath: String, env: [String: String]?, secretBacked: Bool?) {
        self.name = name
        self.client = client
        self.transport = transport
        self.command = command
        self.args = args
        self.url = url
        self.configPath = configPath
        self.env = env
        self.secretBacked = secretBacked
    }
}

public struct Container: Codable, Sendable, Identifiable {
    public let id: String
    public let name: String
    public let image: String
    public let ports: [Int]?

    public init(id: String, name: String, image: String, ports: [Int]?) {
        self.id = id
        self.name = name
        self.image = image
        self.ports = ports
    }
}

public struct Conflict: Codable, Sendable, Identifiable {
    public var id: String {
        "\(port)/\(kind)"
    }

    public let port: Int
    public let holders: [String]
    public let kind: String

    public init(port: Int, holders: [String], kind: String) {
        self.port = port
        self.holders = holders
        self.kind = kind
    }
}

public struct ClientConfig: Codable, Sendable, Identifiable {
    public var id: String {
        client
    }

    public let client: String
    public let path: String
    public let present: Bool

    public init(client: String, path: String, present: Bool) {
        self.client = client
        self.path = path
        self.present = present
    }
}

public struct SuggestionResponse: Codable, Sendable {
    public let free: [Int]
}
