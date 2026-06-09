import Foundation

public struct APIResponse<T: Decodable & Sendable>: Decodable, Sendable {
    public let code: Int
    public let msg: String
    public let data: T?
}

public struct PaginatedData<T: Decodable & Sendable>: Decodable, Sendable {
    public let items: [T]
    public let total: Int
    public let page: Int
    public let size: Int
}

public enum HTTPMethod: String {
    case get = "GET"
    case post = "POST"
    case put = "PUT"
    case delete = "DELETE"
}

public class APIClient: @unchecked Sendable {
    public static let shared = APIClient()
    public var baseURL = "http://192.168.2.111:8080"
    // baseURL is overridden by AppSettings.shared on launch

    private let encoder: JSONEncoder
    private let decoder: JSONDecoder
    private let session: URLSession

    private init() {
        encoder = JSONEncoder()
        decoder = JSONDecoder()

        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = 30
        session = URLSession(configuration: config)
    }

    // MARK: - Request (wrapped response)
    public func request<T: Decodable & Sendable>(
        _ path: String,
        method: HTTPMethod = .get,
        body: (any Encodable & Sendable)? = nil,
        query: [String: String]? = nil
    ) async throws -> T {
        let (data, status) = try await performRequest(path, method: method, body: body, query: query)

        let wrapper = try decoder.decode(APIResponse<T>.self, from: data)

        // Use the server's actual message for 401 so login failures show
        // "密码错误" / "用户不存在" instead of a generic "登录已过期"
        if status == 401 {
            await MainActor.run { AuthManager.shared.logout() }
            throw APIError.server(code: wrapper.code, message: wrapper.msg)
        }

        if wrapper.code != 0 {
            throw APIError.server(code: wrapper.code, message: wrapper.msg)
        }
        guard let value = wrapper.data else {
            throw APIError.decoding(URLError(.cannotParseResponse))
        }
        return value
    }

    // MARK: - Request Direct (no wrapper)
    public func requestDirect<T: Decodable & Sendable>(
        _ path: String,
        method: HTTPMethod = .get,
        body: (any Encodable & Sendable)? = nil,
        query: [String: String]? = nil
    ) async throws -> T {
        let (data, status) = try await performRequest(path, method: method, body: body, query: query)

        if status == 401 {
            await MainActor.run { AuthManager.shared.logout() }
            // Try to extract the server message for a better error message
            if let wrapper = try? decoder.decode(APIResponse<EmptyData>.self, from: data) {
                throw APIError.server(code: wrapper.code, message: wrapper.msg)
            }
            throw APIError.unauthorized
        }

        do {
            return try decoder.decode(T.self, from: data)
        } catch {
            throw APIError.decoding(error)
        }
    }

    // MARK: - Core request
    private func performRequest(
        _ path: String,
        method: HTTPMethod,
        body: (any Encodable & Sendable)?,
        query: [String: String]?
    ) async throws -> (Data, Int) {
        var components = URLComponents(string: baseURL + path)
        if let query {
            components?.queryItems = query.map { URLQueryItem(name: $0.key, value: $0.value) }
        }
        guard let url = components?.url else {
            throw APIError.network(URLError(.badURL))
        }

        var req = URLRequest(url: url)
        req.httpMethod = method.rawValue
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")

        if let token = AuthManager.shared.readToken() {
            req.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }

        if let body {
            req.httpBody = try encoder.encode(AnyEncodable(body))
        }

        let data: Data
        let response: URLResponse
        do {
            (data, response) = try await session.data(for: req)
        } catch {
            throw APIError.network(error)
        }

        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.network(URLError(.badServerResponse))
        }

        return (data, httpResponse.statusCode)
    }
}

// MARK: - Helpers
private struct EmptyData: Codable, Sendable {}

private struct AnyEncodable: Encodable {
    private let value: any Encodable
    init(_ value: any Encodable) {
        self.value = value
    }
    func encode(to encoder: Encoder) throws {
        try value.encode(to: encoder)
    }
}
