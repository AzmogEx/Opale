import Foundation
import FoundationModels

/// Niveau N1 de la cascade (EIA-001) : le modèle Apple Intelligence LOCAL
/// de l'iPhone — pour les questions simples, sans que rien ne quitte
/// l'appareil. Indisponible (simulateur, appareil non compatible, Apple
/// Intelligence désactivée) → la cascade continue vers le backend
/// (N2 homelab → N3 cloud anonymisé), sans friction (EIA-020).
@MainActor
enum LocalAI {
    /// Le modèle local est-il prêt sur cet appareil ?
    static var isAvailable: Bool {
        SystemLanguageModel.default.availability == .available
    }

    /// Cadrage identique à celui du backend : l'IA explique, ne calcule pas.
    private static let instructions = """
    Tu es l'assistant patrimonial de l'application Opale, exécuté LOCALEMENT sur l'iPhone.
    Règles absolues :
    - Tous les chiffres t'ont été fournis par un moteur de calcul déterministe : tu ne calcules jamais rien toi-même et tu n'inventes aucun chiffre.
    - Tu réponds en français, ton direct et chaleureux (tutoiement), 2 à 4 phrases, sans listes.
    - Tu ne recommandes jamais de produit financier précis. Pas de conseil fiscal ou juridique.
    - Si la question dépasse le contexte fourni, dis-le simplement.
    """

    /// Répond à une question simple avec le contexte chiffré fourni.
    /// Renvoie nil si le modèle est indisponible ou refuse — l'appelant
    /// bascule alors sur le backend (EIA-020).
    static func answer(context: String, question: String) async -> String? {
        guard isAvailable else { return nil }
        do {
            let session = LanguageModelSession(instructions: instructions + "\n\n" + context)
            let response = try await session.respond(to: question)
            let text = response.content.trimmingCharacters(in: .whitespacesAndNewlines)
            return text.isEmpty ? nil : text
        } catch {
            // Garde-fou du modèle local ou contexte trop grand : on cascade.
            return nil
        }
    }
}
