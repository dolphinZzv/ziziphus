import SwiftUI
import IMCore

// MARK: - Custom NSTextView with placeholder support
private class PlaceholderTextView: NSTextView {
    var placeholder: String = ""

    override func draw(_ dirtyRect: NSRect) {
        super.draw(dirtyRect)
        guard string.isEmpty else { return }
        let attrs: [NSAttributedString.Key: Any] = [
            .foregroundColor: NSColor.placeholderTextColor,
            .font: NSFont.systemFont(ofSize: AppleDesign.Typography.bodySize)
        ]
        let origin = CGPoint(x: textContainerInset.width, y: textContainerInset.height)
        (placeholder as NSString).draw(at: origin, withAttributes: attrs)
    }
}

struct ChatTextView: NSViewRepresentable {
    @Binding var text: String
    let placeholder: String
    let onTyping: () -> Void
    let onSend: () -> Void
    var onMentionChanged: ((String, Int) -> Void)?

    func makeNSView(context: Context) -> NSScrollView {
        let scrollView = NSScrollView()
        scrollView.hasVerticalScroller = false
        scrollView.hasHorizontalScroller = false
        scrollView.autohidesScrollers = true
        scrollView.borderType = .noBorder
        scrollView.drawsBackground = false

        let textView = PlaceholderTextView()
        textView.placeholder = placeholder
        textView.delegate = context.coordinator
        textView.isRichText = false
        textView.font = .systemFont(ofSize: AppleDesign.Typography.bodySize)
        textView.textContainerInset = NSSize(width: 12, height: 8)
        textView.isVerticallyResizable = true
        textView.isHorizontallyResizable = false
        textView.autoresizingMask = [.width, .height]
        textView.textContainer?.lineBreakMode = .byWordWrapping
        textView.allowsUndo = true
        textView.drawsBackground = false

        scrollView.documentView = textView
        return scrollView
    }

    func updateNSView(_ scrollView: NSScrollView, context: Context) {
        guard let textView = scrollView.documentView as? PlaceholderTextView else { return }
        if textView.string != text {
            textView.string = text
        }
        textView.placeholder = placeholder
        textView.needsDisplay = true
    }

    func makeCoordinator() -> Coordinator {
        Coordinator(parent: self)
    }

    @MainActor
    class Coordinator: NSObject, NSTextViewDelegate {
        let parent: ChatTextView

        init(parent: ChatTextView) {
            self.parent = parent
        }

        func textDidChange(_ notification: Notification) {
            guard let textView = notification.object as? NSTextView else { return }
            parent.text = textView.string
            parent.onTyping()
            detectMention(in: textView)
        }

        private func detectMention(in textView: NSTextView) {
            guard let callback = parent.onMentionChanged else { return }
            let nsString = textView.string as NSString
            let cursorPos = textView.selectedRange().location

            guard cursorPos > 0 else {
                callback("", -1)
                return
            }

            var atPos = -1
            let searchEnd = min(cursorPos - 1, nsString.length - 1)
            for i in stride(from: searchEnd, through: 0, by: -1) {
                let char = nsString.character(at: i)
                if char == UInt16(UnicodeScalar("@").value) {
                    if i == 0 {
                        atPos = i
                        break
                    }
                    let prev = nsString.character(at: i - 1)
                    if prev == UInt16(UnicodeScalar(" ").value)
                        || prev == UInt16(UnicodeScalar("\n").value)
                        || prev == UInt16(UnicodeScalar("\u{2028}").value)
                        || prev == UInt16(UnicodeScalar("\u{2029}").value) {
                        atPos = i
                        break
                    }
                    break
                } else if char == UInt16(UnicodeScalar(" ").value)
                    || char == UInt16(UnicodeScalar("\n").value)
                    || char == UInt16(UnicodeScalar("\u{2028}").value)
                    || char == UInt16(UnicodeScalar("\u{2029}").value) {
                    break
                }
            }

            if atPos >= 0 {
                let queryLength = cursorPos - atPos - 1
                let query = queryLength > 0
                    ? nsString.substring(with: NSRange(location: atPos + 1, length: queryLength))
                    : ""
                callback(query, atPos)
            } else {
                callback("", -1)
            }
        }

        func textView(_ textView: NSTextView, doCommandBy commandSelector: Selector) -> Bool {
            if commandSelector == #selector(NSResponder.insertNewline(_:)) {
                // Check for Shift+Enter → insert newline
                if NSApp.currentEvent?.modifierFlags.contains(.shift) == true {
                    textView.insertNewlineIgnoringFieldEditor(nil)
                    parent.text = textView.string
                    return true
                }
                // Enter → send
                parent.onSend()
                return true
            }
            return false
        }
    }
}
