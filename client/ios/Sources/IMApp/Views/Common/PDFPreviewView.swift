import SwiftUI
import PDFKit
import IMCore

struct PDFPreviewView: View {
    let url: URL
    let filename: String

    @State private var showPreview = false
    @State private var scaleFactor: CGFloat = 1.0

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Button {
                withAnimation(.easeInOut(duration: 0.15)) {
                    showPreview.toggle()
                    if !showPreview { scaleFactor = 1.0 }
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

                        Button { UIApplication.shared.open(url) } label: {
                            HStack(spacing: 2) {
                                Image(systemName: "arrow.up.forward.app")
                                    .font(.caption2)
                                Text(loc("common.open"))
                                    .font(.caption2)
                            }
                            .foregroundColor(.accentColor)
                        }
                    }
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                    .background(Color(.systemGray6))
                    .overlay(Divider(), alignment: .bottom)

                    // PDF content
                    PDFKitView(url: url, scaleFactor: $scaleFactor)
                        .frame(height: 360)
                }
                .clipShape(RoundedRectangle(cornerRadius: 8))
                .overlay(
                    RoundedRectangle(cornerRadius: 8)
                        .stroke(Color(.separator), lineWidth: 0.5)
                )
            }
        }
    }
}

struct PDFKitView: UIViewRepresentable {
    let url: URL
    @Binding var scaleFactor: CGFloat

    func makeUIView(context: Context) -> PDFView {
        let pdfView = PDFView()
        pdfView.autoScales = true
        pdfView.displayMode = .singlePageContinuous
        pdfView.displayDirection = .vertical
        pdfView.document = PDFDocument(url: url)
        return pdfView
    }

    func updateUIView(_ pdfView: PDFView, context: Context) {
        if let doc = pdfView.document, doc.documentURL != url {
            pdfView.document = PDFDocument(url: url)
        }
        pdfView.scaleFactor = pdfView.scaleFactorForSizeToFit * scaleFactor
    }
}
