import SwiftUI
import PDFKit
import IMCore

/// PDF preview that loads the document via an authenticated URLSession
/// using the Bearer token from AuthManager. Falls back to the remote URL
/// when no token is available (public files).
struct AuthPDFPreviewView: View {
    let url: URL
    let filename: String

    @State private var showPreview = false
    @State private var scaleFactor: CGFloat = 1.0
    @State private var document: PDFDocument?

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Button {
                withAnimation(.easeInOut(duration: 0.15)) {
                    showPreview.toggle()
                    if !showPreview { scaleFactor = 1.0 }
                    if showPreview, document == nil { loadDocument() }
                }
            } label: {
                HStack(spacing: 4) {
                    Image(systemName: showPreview ? "eye.slash" : "eye")
                        .font(.caption2)
                    Text(showPreview ? loc("chat.hide_preview") : loc("chat.preview"))
                        .font(.caption)
                }
                .foregroundColor(.accentColor)
            }

            if showPreview {
                VStack(spacing: 0) {
                    // Toolbar
                    HStack {
                        Button { scaleFactor = max(0.5, scaleFactor - 0.2) } label: {
                            Image(systemName: "minus.magnifyingglass")
                                .font(.caption)
                        }

                        Text("\(Int(scaleFactor * 100))%")
                            .font(.system(size: 10))
                            .foregroundColor(.secondary)
                            .frame(width: 36)

                        Button { scaleFactor = min(3.0, scaleFactor + 0.2) } label: {
                            Image(systemName: "plus.magnifyingglass")
                                .font(.caption)
                        }

                        Spacer()
                    }
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                    .background(Color(.systemGray6))
                    .overlay(Divider(), alignment: .bottom)

                    // PDF content
                    if let doc = document {
                        PDFKitView(doc: doc, scaleFactor: $scaleFactor)
                            .frame(height: 360)
                    } else {
                        ProgressView()
                            .frame(height: 360)
                    }
                }
                .clipShape(RoundedRectangle(cornerRadius: 8))
                .overlay(
                    RoundedRectangle(cornerRadius: 8)
                        .stroke(Color(.separator), lineWidth: 0.5)
                )
            }
        }
    }

    private func loadDocument() {
        Task {
            var request = URLRequest(url: url)
            if let token = AuthManager.shared.readToken() {
                request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
            }
            guard let (data, _) = try? await URLSession.shared.data(for: request),
                  let doc = PDFDocument(data: data) else { return }
            await MainActor.run { document = doc }
        }
    }
}

struct PDFKitView: UIViewRepresentable {
    let doc: PDFDocument
    @Binding var scaleFactor: CGFloat

    func makeUIView(context: Context) -> PDFView {
        let pdfView = PDFView()
        pdfView.autoScales = true
        pdfView.displayMode = .singlePageContinuous
        pdfView.displayDirection = .vertical
        pdfView.document = doc
        return pdfView
    }

    func updateUIView(_ pdfView: PDFView, context: Context) {
        pdfView.document = doc
        pdfView.scaleFactor = pdfView.scaleFactorForSizeToFit * scaleFactor
    }
}
