import SwiftUI
import UIKit

extension Notification.Name { static let opaleInteraction = Notification.Name("opale.interaction") }

/// Observes touches at the window without claiming gestures from buttons or scrolling.
private final class ActivityRecognizer: UIGestureRecognizer {
    override func touchesBegan(_ touches: Set<UITouch>, with event: UIEvent) {
        NotificationCenter.default.post(name: .opaleInteraction, object: nil)
        state = .failed
    }
    override func canPrevent(_ preventedGestureRecognizer: UIGestureRecognizer) -> Bool { false }
    override func canBePrevented(by preventingGestureRecognizer: UIGestureRecognizer) -> Bool { false }
}
struct ActivityObserver: UIViewRepresentable {
    final class ObserverView: UIView {
        private weak var observedWindow: UIWindow?
        private let observer = ActivityRecognizer()
        override func didMoveToWindow() {
            super.didMoveToWindow()
            observedWindow?.removeGestureRecognizer(observer)
            observer.cancelsTouchesInView = false
            observer.delaysTouchesBegan = false
            observer.delaysTouchesEnded = false
            window?.addGestureRecognizer(observer)
            observedWindow = window
        }
    }
    func makeUIView(context: Context) -> ObserverView { ObserverView() }
    func updateUIView(_ uiView: ObserverView, context: Context) {}
}
