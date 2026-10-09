import Foundation

nonisolated enum FinancialDocumentKind: String, Sendable { case invoice, payslip }
struct DocumentAmountCandidate: Identifiable, Equatable, Sendable {
    let id: Int
    let amount: Cents
    let amountText: String
    let evidence: String
}
struct ParsedFinancialDocument: Sendable {
    var candidates: [DocumentAmountCandidate]
    var currency: String?
    var name: String
    var dueDate: String?
    var warning: String?
}
struct DocumentPrefillResult: Sendable {
    var name: String
    var amount: Cents
    var currency: String
    var dueDate: String?
}

/// Label-based candidates, never the largest number. No inferred gross-to-net calculation.
enum FinancialDocumentParser {
    static func parse(_ text: String, kind: FinancialDocumentKind) -> ParsedFinancialDocument {
        let lines = String(text.prefix(200_000)).components(separatedBy: .newlines).map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }.filter { !$0.isEmpty }
        let normalized = lines.map(normalize)
        var detected = Set<String>()
        let all = " " + normalize(text) + " "
        for (code, symbols) in [("EUR", ["€", " eur ", " euros"]), ("USD", [" usd ", "us$"]), ("GBP", [" gbp ", "£"]), ("CHF", [" chf "]), ("JPY", [" jpy ", " yen"]), ("KWD", [" kwd "]), ("CAD", [" cad ", "c$"]), ("AUD", [" aud ", "a$"])] {
            if symbols.contains(where: { all.contains($0) }) { detected.insert(code) }
        }
        let currency = detected.count == 1 ? detected.first : nil
        var found: [(Cents, String, Int)] = []
        for (index, line) in normalized.enumerated() {
            let score: Int
            if kind == .payslip {
                guard !line.contains("avant impot"), !line.contains("imposable"), !line.contains("net social"), !line.contains("brut"), !line.contains("cumul"), !line.contains("annuel") else { continue }
                if line.contains("net paye") || line.contains("net verse") || line.contains("net a payer apres") || line.contains("net to pay") || line.contains("net paid") { score = 3 }
                else if line.contains("net a payer") { score = 2 }
                else { continue }
            } else {
                if line.contains("net a payer") || line.contains("total a payer") || line.contains("amount due") { score = 3 }
                else if line.contains("total ttc") || line.contains("total toutes taxes") { score = 2 }
                else { continue }
            }
            // OCR may put the amount on the next line. Only accept that line if it is just an amount/currency.
            var context = lines[index]
            var amounts = extractAmounts(context, currency: currency ?? "EUR")
            if amounts.isEmpty, index + 1 < lines.count, amountOnly(lines[index + 1]) {
                context += " · " + lines[index + 1]
                amounts = extractAmounts(lines[index + 1], currency: currency ?? "EUR")
            }
            for amount in amounts { found.append((amount, String(context.prefix(200)), score)) }
        }
        let highest = found.map(\.2).max() ?? 0
        var seen = Set<Int64>()
        let selected = found.filter { $0.2 == highest }.filter { seen.insert($0.0.raw).inserted }
        let candidates = selected.enumerated().map { DocumentAmountCandidate(id: $0.offset, amount: $0.element.0, amountText: MoneyFormat.input($0.element.0, currency: currency ?? "EUR"), evidence: $0.element.1) }
        var name = ""
        for (index, line) in normalized.enumerated() {
            let prefix = kind == .invoice ? ["fournisseur:", "fournisseur :"] : ["employeur:", "employeur :"]
            if prefix.contains(where: { line.hasPrefix($0) }), let colon = lines[index].firstIndex(of: ":") {
                name = String(lines[index][lines[index].index(after: colon)...]).trimmingCharacters(in: .whitespaces).prefix(200).description
                break
            }
        }
        var dueDate: String?
        if kind == .invoice {
            for (index, line) in normalized.enumerated() where line.contains("echeance") || line.contains("due date") || line.contains("payer avant") {
                let possible = parseDay(lines[index])
                if possible != nil { dueDate = possible; break }
            }
        }
        var warning: String?
        if candidates.isEmpty { warning = "Aucun net payé / total à payer fiable trouvé. Saisis le montant en vérifiant le document." }
        else if candidates.count > 1 { warning = "Plusieurs montants possibles : sélectionne celui à utiliser après vérification." }
        if currency == nil { warning = [warning, "Devise absente ou ambiguë : choisis-la explicitement."].compactMap { $0 }.joined(separator: " ") }
        return ParsedFinancialDocument(candidates: candidates, currency: currency, name: name, dueDate: dueDate, warning: warning)
    }
    private static func normalize(_ text: String) -> String {
        text.folding(options: [.diacriticInsensitive, .caseInsensitive], locale: Locale(identifier: "fr_FR"))
            .replacingOccurrences(of: "’", with: "'").lowercased()
    }
    private static func extractAmounts(_ text: String, currency: String) -> [Cents] {
        // A decimal part is mandatory: do not turn account numbers, dates or quantities into money.
        let regex = try! NSRegularExpression(pattern: #"(?<![\p{L}\d.,/-])(?:\d{1,3}(?:[ \u00A0\u202F]\d{3})+|\d+)[,.]\d{2,3}(?![\d.,])"#)
        let ns = text as NSString
        return regex.matches(in: text, range: NSRange(location: 0, length: ns.length)).compactMap { match in
            FinancialTools.amount(ns.substring(with: match.range), currency: currency)
        }
    }
    private static func amountOnly(_ line: String) -> Bool {
        let allowed = CharacterSet(charactersIn: "0123456789., €$£¥\u{00A0}\u{202F}EURUSDGBPCHFJPYKWD CADAUD")
        return !line.isEmpty && line.uppercased().unicodeScalars.allSatisfy { allowed.contains($0) }
    }
    private static func parseDay(_ text: String) -> String? {
        for pattern in [#"\b(\d{4})-(\d{2})-(\d{2})\b"#, #"\b(\d{2})[/.](\d{2})[/.](\d{4})\b"#] {
            let re = try! NSRegularExpression(pattern: pattern), ns = text as NSString
            guard let match = re.firstMatch(in: text, range: NSRange(location: 0, length: ns.length)) else { continue }
            let a = ns.substring(with: match.range(at: 1)), b = ns.substring(with: match.range(at: 2)), c = ns.substring(with: match.range(at: 3))
            let day = a.count == 4 ? "\(a)-\(b)-\(c)" : "\(c)-\(b)-\(a)"
            if Date.fromOpaleDay(day) != nil { return day }
        }
        return nil
    }
}
