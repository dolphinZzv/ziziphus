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
            return loc("error.unauthorized")
        case .decoding:
            return loc("error.decoding_failed")
        case .timeout:
            return loc("error.timeout")
        case .wsError(_, let message):
            return message
        }
    }
}
