import Foundation

extension String {
    public var isImageURL: Bool {
        let lowercased = lowercased()
        return lowercased.hasSuffix(".png") || lowercased.hasSuffix(".jpg")
            || lowercased.hasSuffix(".jpeg") || lowercased.hasSuffix(".gif")
            || lowercased.hasSuffix(".webp") || lowercased.hasSuffix(".svg")
    }

    public func extractImageURLs(baseURL: URL?) -> [URL] {
        let pattern = #"!\[[^\]]*\]\(([^)]+)\)"#
        guard let regex = try? NSRegularExpression(pattern: pattern, options: []) else { return [] }
        let nsString = self as NSString
        let matches = regex.matches(in: self, options: [], range: NSRange(location: 0, length: nsString.length))
        return matches.compactMap { match -> URL? in
            guard match.numberOfRanges > 1 else { return nil }
            let urlStr = nsString.substring(with: match.range(at: 1))
            return URL(string: urlStr, relativeTo: baseURL)?.absoluteURL
        }
    }
}
