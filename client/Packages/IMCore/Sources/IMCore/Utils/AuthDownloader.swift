import Foundation
import UIKit

/// Downloads a file with the Bearer auth token and presents a share sheet.
public enum AuthDownloader {

    /// Download the file at `url` with the stored auth token and present
    /// a system share sheet from the key window's root view controller.
    public static func downloadAndShare(url: URL) {
        Task {
            guard let data = await download(url) else { return }
            let tempURL = FileManager.default.temporaryDirectory
                .appendingPathComponent(url.lastPathComponent)
            do {
                try data.write(to: tempURL)
                let avc = UIActivityViewController(activityItems: [tempURL], applicationActivities: nil)
                if let scene = UIApplication.shared.connectedScenes.first as? UIWindowScene,
                   let root = scene.windows.first?.rootViewController {
                    root.present(avc, animated: true)
                }
            } catch {}
        }
    }

    private static func download(_ url: URL) async -> Data? {
        var request = URLRequest(url: url)
        if let token = AuthManager.shared.readToken() {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        return try? await URLSession.shared.data(for: request).0
    }
}
