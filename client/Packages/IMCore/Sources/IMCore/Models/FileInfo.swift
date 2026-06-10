import Foundation

public struct FileInfo: Codable, Sendable {
    public let fileID: String
    public let url: String
    public let thumbnailURL: String?
    public let size: Int64
    public let name: String
    public let width: Int?
    public let height: Int?
    public let contentType: Int?

    enum CodingKeys: String, CodingKey {
        case fileID = "file_id"
        case url, size, name, width, height
        case thumbnailURL = "thumbnail_url"
        case contentType = "content_type"
    }
}

public struct FileMessageBody: Codable, Sendable {
    public let fileID: String
    public let url: String
    public let name: String
    public let size: Int64?

    enum CodingKeys: String, CodingKey {
        case fileID = "file_id"
        case url, name, size
    }

    public init(fileID: String, url: String, name: String, size: Int64?) {
        self.fileID = fileID
        self.url = url
        self.name = name
        self.size = size
    }
}
