import Foundation

/// Données déclarées pour préparer le profil ; aucune opération réalisée n'est inventée.
struct FinancialSetupDraft: Codable, Equatable, Sendable {
    var income = FinancialSetupIncome()
    var account = FinancialSetupAccount()
    var expenses: [FinancialSetupEntry] = []
    var subscriptions: [FinancialSetupEntry] = []
    var goal = FinancialSetupGoal()
    var variableBudget = ""

    enum CodingKeys: String, CodingKey {
        case income, account, expenses, subscriptions, goal
        case variableBudget = "variable_budget"
    }

    var summary: FinancialSetupSummary { FinancialSetupSummary(draft: self) }
    var validationErrors: [String] { validationErrors(forStep: 5) }
    var isValid: Bool { validationErrors.isEmpty }

    func validationErrors(on today: Date) -> [String] { validationErrors(forStep: 5, on: today) }

    func validationErrors(forStep step: Int, on today: Date = .now) -> [String] {
        let day = today.opaleDayString
        let lastDay = Calendar.opale.date(byAdding: .year, value: 10, to: today)!.opaleDayString
        var errors: [String] = []
        func validFutureDate(_ value: String) -> Bool {
            Date.fromOpaleDay(value) != nil && value >= day && value <= lastDay
        }
        if (step == 0 || step == 5), income.enabled {
            if FinancialSetupMoney.positive(income.amount) == nil { errors.append("Indique un salaire net positif, avec deux décimales maximum.") }
            if !validFutureDate(income.nextDate) { errors.append("Choisis la prochaine date de versement, entre aujourd’hui et dans dix ans.") }
        }
        if step == 1 || step == 5 {
            if account.enabled {
                if account.existingAssetID.isEmpty {
                    if account.name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty || account.name.count > 200 { errors.append("Donne un nom à ton compte (200 caractères maximum).") }
                    if !account.balance.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                        if FinancialSetupMoney.parse(account.balance) == nil { errors.append("Le solde du compte doit être un montant valide, avec deux décimales maximum.") }
                        if Date.fromOpaleDay(account.balanceDate) == nil || account.balanceDate < "1900-01-01" || account.balanceDate > day { errors.append("La date du solde doit être comprise entre 1900 et aujourd’hui.") }
                    }
                }
            } else if income.enabled || !expenses.isEmpty || !subscriptions.isEmpty {
                errors.append("Ajoute ou sélectionne un compte pour enregistrer tes échéances.")
            }
        }
        var ids = Set<String>()
        for (index, item) in [(expenses, "charge"), (subscriptions, "abonnement")].enumerated() {
            guard step == index + 2 || step == 5 else { continue }
            let (entries, title) = item
            if entries.count > 100 { errors.append("Limite de 100 éléments par liste.") }
            for entry in entries {
                let label = entry.label.trimmingCharacters(in: .whitespacesAndNewlines)
                if UUID(uuidString: entry.id) == nil || !ids.insert(entry.id.lowercased()).inserted { errors.append("Une ligne a un identifiant invalide ou est présente deux fois.") }
                if label.isEmpty || label.count > 200 { errors.append("Donne un nom à chaque \(title) (200 caractères maximum).") }
                if FinancialSetupMoney.positive(entry.amount) == nil { errors.append("Le montant de chaque \(title) doit être positif, avec deux décimales maximum.") }
                if !["monthly", "quarterly", "yearly"].contains(entry.frequency) { errors.append("Choisis une fréquence mensuelle, trimestrielle ou annuelle.") }
                if !validFutureDate(entry.date) { errors.append("La prochaine échéance de chaque \(title) doit être comprise entre aujourd’hui et dans dix ans.") }
            }
        }
        if step == 4 || step == 5 {
            if FinancialSetupMoney.optionalNonnegative(variableBudget) == nil { errors.append("Le budget quotidien doit être positif ou nul, avec deux décimales maximum.") }
            if goal.enabled {
                if goal.name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty || goal.name.count > 200 { errors.append("Donne un nom à ton objectif (200 caractères maximum).") }
                if FinancialSetupMoney.positive(goal.target) == nil { errors.append("Indique un montant cible positif pour ton objectif.") }
            }
        }
        return errors
    }
}

struct FinancialSetupIncome: Codable, Equatable, Sendable {
    var enabled = true
    var amount = ""
    var nextDate = Date.now.opaleDayString
    enum CodingKeys: String, CodingKey { case enabled, amount; case nextDate = "next_date" }
}

struct FinancialSetupAccount: Codable, Equatable, Sendable {
    var enabled = true
    var existingAssetID = ""
    var name = "Compte courant"
    var balance = ""
    var balanceDate = Calendar.opale.date(byAdding: .day, value: -1, to: .now)!.opaleDayString
    enum CodingKeys: String, CodingKey {
        case enabled, name, balance
        case existingAssetID = "existing_asset_id"
        case balanceDate = "balance_date"
    }
}

struct FinancialSetupGoal: Codable, Equatable, Sendable {
    var enabled = false
    var name = ""
    var target = ""
}

struct FinancialSetupEntry: Codable, Equatable, Identifiable, Sendable {
    var id = UUID().uuidString
    var label = ""
    var amount = ""
    var date = Date.now.opaleDayString
    var frequency = "monthly"
}

struct FinancialSetupState: Codable, Equatable, Sendable {
    var version: Int
    var status: String
    var step: Int
    var revision: Int
    var canStart: Bool
    var shouldPrompt: Bool
    var draft: FinancialSetupDraft
    var result: FinancialSetupResult?
    enum CodingKeys: String, CodingKey {
        case version, status, step, revision, draft, result
        case canStart = "can_start"
        case shouldPrompt = "should_prompt"
    }
}

struct FinancialSetupResult: Codable, Equatable, Sendable {
    var assetID: String
    var incomeRuleID: String
    var expenseRuleIDs: [String]
    var subscriptionRuleIDs: [String]
    var goalID: String
    enum CodingKeys: String, CodingKey {
        case assetID = "asset_id"
        case incomeRuleID = "income_rule_id"
        case expenseRuleIDs = "expense_rule_ids"
        case subscriptionRuleIDs = "subscription_rule_ids"
        case goalID = "goal_id"
    }
}

/// Tous les totaux restent en Decimal ; division et arrondi se font après la somme.
/// Les montants de salaire et de budget sont mensuels. Les fréquences sont exactes
/// (trimestre = trois mois, année = douze mois), sans approximation en jours.
struct FinancialSetupSummary: Equatable, Sendable {
    let income: Cents?
    let expenses: Cents?
    let subscriptions: Cents?
    let variableBudget: Cents?
    let variableBudgetProvided: Bool
    let remaining: Cents?
    let isComplete: Bool

    init(draft: FinancialSetupDraft) {
        let expenseAnnual = Self.annualTotal(draft.expenses)
        let subscriptionAnnual = Self.annualTotal(draft.subscriptions)
        income = draft.income.enabled ? FinancialSetupMoney.positive(draft.income.amount) : nil
        expenses = expenseAnnual.flatMap { Self.roundedCents($0 / 12) }
        subscriptions = subscriptionAnnual.flatMap { Self.roundedCents($0 / 12) }
        variableBudget = FinancialSetupMoney.optionalNonnegative(draft.variableBudget)
        variableBudgetProvided = !draft.variableBudget.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
        if let income, let expenseAnnual, let subscriptionAnnual, let variableBudget {
            remaining = Self.roundedCents((Decimal(income.raw) * 12 - expenseAnnual - subscriptionAnnual - Decimal(variableBudget.raw) * 12) / 12)
        } else { remaining = nil }
        isComplete = (!draft.income.enabled || income != nil) && expenses != nil && subscriptions != nil && variableBudget != nil
    }

    static func annualTotal(_ entries: [FinancialSetupEntry]) -> Decimal? {
        var total = Decimal.zero
        for entry in entries {
            guard let amount = FinancialSetupMoney.positive(entry.amount) else { return nil }
            let multiplier: Int
            switch entry.frequency {
            case "monthly": multiplier = 12
            case "quarterly": multiplier = 4
            case "yearly": multiplier = 1
            default: return nil
            }
            total += Decimal(amount.raw) * Decimal(multiplier)
        }
        return total
    }

    static func roundedCents(_ value: Decimal) -> Cents? {
        guard !value.isNaN else { return nil }
        var original = value
        var rounded = Decimal.zero
        NSDecimalRound(&rounded, &original, 0, .plain)
        guard rounded >= Decimal(Int64.min), rounded <= Decimal(Int64.max) else { return nil }
        return Cents(NSDecimalNumber(decimal: rounded).int64Value)
    }
}

enum FinancialSetupMoney {
    static let maximumCents: Int64 = 999_999_999_999
    static func parse(_ text: String) -> Cents? {
        guard let amount = Cents.parse(text), amount.raw >= -maximumCents, amount.raw <= maximumCents else { return nil }
        return amount
    }
    static func positive(_ text: String) -> Cents? {
        guard let amount = parse(text), amount.raw > 0 else { return nil }
        return amount
    }
    static func optionalNonnegative(_ text: String) -> Cents? {
        if text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty { return .zero }
        guard let amount = parse(text), amount.raw >= 0 else { return nil }
        return amount
    }
}

struct FinancialSetupCachedDraft: Codable, Equatable, Sendable {
    var draft: FinancialSetupDraft
    var step: Int
    var revision: Int
}

/// Cache protégé, effacé avec les autres données lors de la déconnexion.
enum FinancialSetupCache {
    private static func key(_ profileKey: String) -> String { "financial-setup-v1|" + profileKey }
    static func save(draft: FinancialSetupDraft, step: Int, revision: Int, profileKey: String) {
        DiskCache.save(FinancialSetupCachedDraft(draft: draft, step: step, revision: revision), key: key(profileKey))
    }
    static func load(profileKey: String) -> FinancialSetupCachedDraft? {
        DiskCache.load(FinancialSetupCachedDraft.self, key: key(profileKey))?.value
    }
    static func clear(profileKey: String) {
        // Écrase le contenu financier avec un marqueur vide ; les autres caches sont conservés.
        DiskCache.save(Optional<FinancialSetupCachedDraft>.none, key: key(profileKey))
    }
}

extension APIClient {
    func fetchFinancialSetup() async throws -> FinancialSetupState {
        try await request("GET", "/v1/onboarding")
    }
    func saveFinancialSetup(draft: FinancialSetupDraft, step: Int, revision: Int) async throws -> FinancialSetupState {
        struct Body: Encodable {
            let expected_revision: Int
            let step: Int
            let draft: FinancialSetupDraft
        }
        return try await request("PUT", "/v1/onboarding", body: Body(expected_revision: revision, step: step, draft: draft))
    }
    func completeFinancialSetup(revision: Int) async throws -> FinancialSetupState {
        struct Body: Encodable { let expected_revision: Int }
        return try await request("POST", "/v1/onboarding/complete", body: Body(expected_revision: revision))
    }
    func skipFinancialSetup(revision: Int) async throws -> FinancialSetupState {
        struct Body: Encodable { let expected_revision: Int }
        return try await request("POST", "/v1/onboarding/skip", body: Body(expected_revision: revision))
    }
}
