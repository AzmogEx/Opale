import SwiftUI

/// Keep a visible gap between a label and its value; stack at accessibility sizes.
struct AdaptiveLabeledContentStyle: LabeledContentStyle {
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize
    func makeBody(configuration: Configuration) -> some View {
        if dynamicTypeSize.isAccessibilitySize {
            VStack(alignment: .leading, spacing: 8) {
                configuration.label
                configuration.content.foregroundStyle(.secondary)
            }
        } else {
            HStack(alignment: .firstTextBaseline, spacing: 12) {
                configuration.label
                Spacer(minLength: 12)
                configuration.content
                    .multilineTextAlignment(.trailing)
                    .foregroundStyle(.secondary)
            }
        }
    }
}
