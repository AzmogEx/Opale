import Foundation
import CoreGraphics
import ImageIO
import PDFKit
import Vision

nonisolated struct DocumentText: Sendable {
    let text: String
    let pagesRead: Int
    let totalPages: Int
}
nonisolated enum DocumentReadError: LocalizedError {
    case unsupported, tooLarge, unreadable, protectedPDF, noText
    var errorDescription: String? {
        switch self {
        case .unsupported: "Choisis une image ou un PDF."
        case .tooLarge: "Le document doit faire moins de 10 Mo."
        case .unreadable: "Impossible de lire ce document. Essaie un PDF ou une photo plus nette."
        case .protectedPDF: "Ce PDF est protégé. Choisis une copie déverrouillée."
        case .noText: "Aucun texte lisible détecté. Tu peux saisir les champs manuellement."
        }
    }
}

/// No network and no persistent copy: PDF text first, otherwise Apple's on-device OCR.
nonisolated enum LocalDocumentReader {
    static func read(_ url: URL) async throws -> DocumentText {
        let worker = Task.detached(priority: .userInitiated) { try extract(url) }
        return try await withTaskCancellationHandler { try await worker.value } onCancel: { worker.cancel() }
    }
    private static func extract(_ url: URL) throws -> DocumentText {
        let scoped = url.startAccessingSecurityScopedResource()
        defer { if scoped { url.stopAccessingSecurityScopedResource() } }
        let values = try url.resourceValues(forKeys: [.fileSizeKey, .isRegularFileKey])
        guard values.isRegularFile == true else { throw DocumentReadError.unsupported }
        guard let size = values.fileSize, size > 0, size <= 10 * 1024 * 1024 else { throw DocumentReadError.tooLarge }
        let data = try Data(contentsOf: url, options: .mappedIfSafe)
        guard data.count <= 10 * 1024 * 1024 else { throw DocumentReadError.tooLarge }
        try Task.checkCancellation()
        if data.starts(with: Data("%PDF".utf8)) {
            guard let pdf = PDFDocument(data: data) else { throw DocumentReadError.unreadable }
            guard !pdf.isLocked else { throw DocumentReadError.protectedPDF }
            let count = min(5, pdf.pageCount)
            var texts: [String] = []
            for index in 0..<count {
                try Task.checkCancellation()
                guard let page = pdf.page(at: index) else { continue }
                let direct = page.string ?? ""
                if direct.trimmingCharacters(in: .whitespacesAndNewlines).count >= 40 {
                    texts.append(direct)
                } else if let image = render(page: page) {
                    texts.append(try recognize(image))
                }
            }
            let text = String(texts.joined(separator: "\n").prefix(200_000))
            guard !text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else { throw DocumentReadError.noText }
            return DocumentText(text: text, pagesRead: count, totalPages: pdf.pageCount)
        }
        guard let source = CGImageSourceCreateWithData(data as CFData, nil),
              let image = CGImageSourceCreateThumbnailAtIndex(source, 0, [kCGImageSourceCreateThumbnailFromImageAlways: true, kCGImageSourceThumbnailMaxPixelSize: 2000, kCGImageSourceCreateThumbnailWithTransform: true] as CFDictionary) else { throw DocumentReadError.unreadable }
        let text = try recognize(image)
        guard !text.isEmpty else { throw DocumentReadError.noText }
        return DocumentText(text: text, pagesRead: 1, totalPages: 1)
    }
    private static func render(page: PDFPage) -> CGImage? {
        guard let pdfPage = page.pageRef else { return nil }
        let rect = pdfPage.getBoxRect(.mediaBox)
        guard rect.width.isFinite, rect.height.isFinite, rect.width > 0, rect.height > 0 else { return nil }
        let scale = min(1600 / max(rect.width, rect.height), 3)
        let width = max(1, Int(rect.width * scale)), height = max(1, Int(rect.height * scale))
        guard let context = CGContext(data: nil, width: width, height: height, bitsPerComponent: 8, bytesPerRow: 0, space: CGColorSpaceCreateDeviceRGB(), bitmapInfo: CGImageAlphaInfo.premultipliedLast.rawValue) else { return nil }
        context.setFillColor(CGColor(gray: 1, alpha: 1)); context.fill(CGRect(x: 0, y: 0, width: width, height: height))
        context.concatenate(pdfPage.getDrawingTransform(.mediaBox, rect: CGRect(x: 0, y: 0, width: width, height: height), rotate: 0, preserveAspectRatio: true))
        context.drawPDFPage(pdfPage)
        return context.makeImage()
    }
    private static func recognize(_ image: CGImage) throws -> String {
        try Task.checkCancellation()
        let request = VNRecognizeTextRequest()
        request.recognitionLevel = .accurate
        request.recognitionLanguages = ["fr-FR", "en-US"]
        request.usesLanguageCorrection = true
        try VNImageRequestHandler(cgImage: image).perform([request])
        return (request.results ?? []).compactMap { $0.topCandidates(1).first?.string }.joined(separator: "\n")
    }
}
