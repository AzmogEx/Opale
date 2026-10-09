import XCTest
import UIKit
import PDFKit
@testable import Opale

final class FinancialToolsTests: XCTestCase {
    @MainActor func testSalaryUsesNetPaidAfterTaxAndNotGrossOrTaxableNet() {
        let parsed = FinancialDocumentParser.parse("""
        Employeur : Atelier Exemple
        Salaire brut 4 100,00 EUR
        Net imposable 3 200,75 EUR
        NET À PAYER AVANT IMPÔT 3 050,40 EUR
        Prélèvement à la source 200,25 EUR
        NET PAYÉ
        2 850,15 EUR
        Cumul annuel net payé 25 000,00 EUR
        """, kind: .payslip)
        XCTAssertEqual(parsed.currency, "EUR")
        XCTAssertEqual(parsed.candidates.map(\.amount.raw), [285015])
        XCTAssertEqual(parsed.name, "Atelier Exemple")
        XCTAssertNil(parsed.dueDate)
    }
    @MainActor func testInvoiceUsesPayableTotalAndExplicitDueDate() {
        let parsed = FinancialDocumentParser.parse("""
        Fournisseur : Énergie Exemple
        Facture n° 202610091234
        Total HT 100,00 €
        TVA 20,00 €
        TOTAL TTC 120,00 €
        Acompte reçu 20,00 €
        NET À PAYER 100,00 €
        Échéance : 25/10/2026
        """, kind: .invoice)
        XCTAssertEqual(parsed.candidates.map(\.amount.raw), [10000])
        XCTAssertEqual(parsed.dueDate, "2026-10-25")
        XCTAssertEqual(parsed.name, "Énergie Exemple")
    }
    @MainActor func testAmbiguousAmountsAreKeptForExplicitSelection() {
        let parsed = FinancialDocumentParser.parse("Total TTC 10,99 €\nTotal TTC 12,99 €\nTotal TTC 10,99 €", kind: .invoice)
        XCTAssertEqual(parsed.candidates.count, 2)
        XCTAssertNotNil(parsed.warning)
        XCTAssertEqual(parsed.candidates[0].amountText, "10.99")
        let noNet = FinancialDocumentParser.parse("Brut 9999,00 EUR\nNet imposable 3000,00 EUR\nNet à payer avant impôt 2800,00 EUR", kind: .payslip)
        XCTAssertTrue(noNet.candidates.isEmpty)
        let noCurrency = FinancialDocumentParser.parse("Total TTC 100,00\nÉchéance 30/02/2026", kind: .invoice)
        XCTAssertNil(noCurrency.currency)
        XCTAssertNil(noCurrency.dueDate)
        XCTAssertNotNil(noCurrency.warning)
    }
    @MainActor func testNativeCurrencyAmountsAndExactAnnualisation() {
        let parsed = FinancialDocumentParser.parse("Net payé 1 234,567 KWD", kind: .payslip)
        XCTAssertEqual(parsed.candidates.first?.amount.raw, 1234567)
        XCTAssertNil(FinancialTools.amount("12.34", currency: "JPY"))
        XCTAssertEqual(FinancialTools.amount("1234", currency: "JPY")?.raw, 1234)
        var annual = FinancialContract(); annual.amount = Cents(100); annual.frequency = "yearly"
        XCTAssertEqual(FinancialTools.monthlyContracts(Array(repeating: annual, count: 6), currency: "EUR")?.raw, 50)
        var regular = VariableIncome(); regular.low = Cents(100); regular.usual = Cents(200); regular.high = Cents(300); regular.frequency = "quarterly"
        var oneOff = regular; oneOff.frequency = "once"; oneOff.usual = Cents(999999)
        var foreign = regular; foreign.currency = "JPY"
        var stopped = regular; stopped.active = false
        XCTAssertEqual(FinancialTools.monthlyIncome([regular, oneOff, foreign, stopped], currency: "EUR", keyPath: \.usual)?.raw, 67)
    }
    @MainActor func testInvestmentFiltersDistinguishCountryResidenceRiskAndHorizon() throws {
        let world = try XCTUnwrap(InvestmentCatalog.ideas.first { $0.id == "world-pea" })
        XCTAssertTrue(world.matches(risk: .high, country: "Monde", family: "etf", envelope: "PEA", residence: "France", horizon: 10, query: "IE0002XZSHO1"))
        XCTAssertFalse(world.matches(risk: .low, country: "", family: "", envelope: "", residence: "France", horizon: nil, query: ""))
        XCTAssertFalse(world.matches(risk: .high, country: "États-Unis", family: "", envelope: "", residence: "France", horizon: nil, query: ""))
        XCTAssertFalse(world.matches(risk: .high, country: "", family: "", envelope: "PEA", residence: "Suisse", horizon: nil, query: ""))
        XCTAssertFalse(world.matches(risk: .high, country: "", family: "", envelope: "", residence: "France", horizon: 1, query: ""))
        XCTAssertEqual(Set(InvestmentCatalog.ideas.map(\.id)).count, InvestmentCatalog.ideas.count)
        XCTAssertTrue(InvestmentCatalog.ideas.allSatisfy { !$0.sources.isEmpty && $0.sources.allSatisfy { URL(string: $0.url)?.scheme == "https" } })
    }
    @MainActor func testActualOnDeviceOCRAndPDFTextExtraction() async throws {
        let text = "FACTURE\nFournisseur : Exemple\nTotal TTC 120,50 EUR\nEcheance : 25/10/2026"
        let image = UIGraphicsImageRenderer(size: CGSize(width: 1200, height: 500)).image { context in
            UIColor.white.setFill(); context.fill(CGRect(x: 0, y: 0, width: 1200, height: 500))
            (text as NSString).draw(in: CGRect(x: 40, y: 35, width: 1100, height: 400), withAttributes: [.font: UIFont.systemFont(ofSize: 42), .foregroundColor: UIColor.black])
        }
        let url = FileManager.default.temporaryDirectory.appendingPathComponent("ocr-\(UUID().uuidString).png")
        try XCTUnwrap(image.pngData()).write(to: url)
        defer { try? FileManager.default.removeItem(at: url) }
        let result = try await LocalDocumentReader.read(url)
        let parsed = FinancialDocumentParser.parse(result.text, kind: .invoice)
        XCTAssertEqual(parsed.candidates.first?.amount.raw, 12050, result.text)
        let pdfURL = url.deletingPathExtension().appendingPathExtension("pdf")
        let renderer = UIGraphicsPDFRenderer(bounds: CGRect(x: 0, y: 0, width: 600, height: 400))
        try renderer.writePDF(to: pdfURL) { ctx in ctx.beginPage(); (text as NSString).draw(at: CGPoint(x: 20, y: 30), withAttributes: [.font: UIFont.systemFont(ofSize: 18)]) }
        defer { try? FileManager.default.removeItem(at: pdfURL) }
        let pdfText = try await LocalDocumentReader.read(pdfURL)
        XCTAssertEqual(pdfText.pagesRead, 1)
        XCTAssertEqual(FinancialDocumentParser.parse(pdfText.text, kind: .invoice).candidates.first?.amount.raw, 12050)
    }
}
