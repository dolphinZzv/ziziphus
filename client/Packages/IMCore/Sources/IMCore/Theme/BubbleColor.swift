import Foundation
import SwiftUI

// MARK: - Bubble Color Presets

public struct BubblePreset: Hashable, Identifiable, Sendable {
    public let id: String
    public let hex: String
    public let name: String

    public init(id: String = UUID().uuidString, hex: String, name: String) {
        self.id = id
        self.hex = hex
        self.name = name
    }
}

public let bubblePresets: [BubblePreset] = [
    BubblePreset(hex: "#d5e3f8", name: "气泡蓝"),
    BubblePreset(hex: "#aec6f5", name: "天蓝"),
    BubblePreset(hex: "#b7e1cd", name: "薄荷绿"),
    BubblePreset(hex: "#fce8b2", name: "暖黄"),
    BubblePreset(hex: "#f5c6d0", name: "粉红"),
    BubblePreset(hex: "#d4c5f9", name: "淡紫"),
    BubblePreset(hex: "#fad7b5", name: "杏色"),
    BubblePreset(hex: "#d1d1d6", name: "浅灰"),
]

public extension Color {
    init(bubbleHex: String) {
        let h = bubbleHex.trimmingCharacters(in: CharacterSet.alphanumerics.inverted)
        var int: UInt64 = 0
        Scanner(string: h).scanHexInt64(&int)
        let r = Double((int >> 16) & 0xFF) / 255
        let g = Double((int >> 8) & 0xFF) / 255
        let b = Double(int & 0xFF) / 255
        self.init(.displayP3, red: r, green: g, blue: b)
    }
}

// MARK: - Bubble Color Picker

public struct BubbleColorPicker: View {
    @Binding public var selectedHex: String

    private let columns = [GridItem(.adaptive(minimum: 44, maximum: 52), spacing: 10)]

    public init(selectedHex: Binding<String>) {
        _selectedHex = selectedHex
    }

    public var body: some View {
        LazyVGrid(columns: columns, spacing: 10) {
            ForEach(bubblePresets) { preset in
                Circle()
                    .fill(Color(bubbleHex: preset.hex))
                    .frame(width: 40, height: 40)
                    .overlay(
                        Circle()
                            .stroke(preset.hex == selectedHex ? Color.primary : Color.clear, lineWidth: 2.5)
                    )
                    .overlay(
                        Circle()
                            .stroke(.separator.opacity(0.3), lineWidth: 0.5)
                    )
                    .shadow(color: .black.opacity(0.1), radius: 1, x: 0, y: 1)
                    .onTapGesture {
                        selectedHex = preset.hex
                    }
            }
        }
    }
}
