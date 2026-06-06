import Foundation

public enum APIError: Error, LocalizedError {
    case network(Error)
    case server(code: Int, message: String)
    case unauthorized
    case decoding(Error)
    case timeout
    case wsError(code: Int, message: String)

    public var errorDescription: String? {
        switch self {
        case .network(let error):
            return error.localizedDescription
        case .server(_, let message):
            return message
        case .unauthorized:
            return "登录已过期，请重新登录"
        case .decoding:
            return "数据解析失败"
        case .timeout:
            return "请求超时"
        case .wsError(_, let message):
            return message
        }
    }
}
