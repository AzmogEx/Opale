# Confidentialité — classification des données (EIA-034)

> « Votre patrimoine ne devrait pas devenir le business model de quelqu'un
> d'autre. » Ce document formalise le niveau de confidentialité de CHAQUE
> donnée d'Opale (EIA-030/031/032) et le point du code qui l'applique.

## Les trois niveaux (§6.5 du cahier des charges)

| Niveau | Règle | Application dans le code |
|---|---|---|
| **N1 — Local only** | Ne quitte jamais l'appareil / le homelab | Par défaut : tout ce qui n'est pas explicitement autorisé plus bas |
| **N2 — Cloud anonymisé possible** | Envoi cloud UNIQUEMENT après anonymisation | `twin.Anonymize()` (montants arrondis « 42k », profil anonyme) — seul contenu que `ai.Router` accepte de transmettre au N3, et seulement avec consentement (`allow_cloud`, EIA-022) |
| **N3 — Cloud interdit** | Jamais envoyé au cloud, même anonymisé | Aucun chemin de code ne les fait entrer dans un prompt N3 |

## Classification par donnée

| Donnée | Niveau | Justification / mécanisme |
|---|---|---|
| Nom du profil, PIN (haché bcrypt) | **N1** | Jamais dans un prompt IA ; l'anonymiseur remplace le nom par « Profil A » |
| Transactions brutes (libellés bancaires, marchands) | **N1** | Transmises au N2 homelab (privé) uniquement ; jamais au cloud |
| Documents du coffre (actes, identité, contrats) | **N3** | Chiffrés AES-256-GCM au repos (`internal/vault`) ; ne sortent QUE via l'export utilisateur (EF-006) |
| Identifiants bancaires | **N3 par construction** | N'existent nulle part : le consentement DSP2 se donne chez la banque (GoCardless) |
| Montants agrégés, ratios, taux d'épargne | **N2** | Arrondis au millier (`twin.compactK`) avant tout envoi cloud |
| Objectifs (nom générique + montant) | **N2** | Cités par l'anonymiseur avec montant arrondi |
| Score de santé, risques détectés (titres) | **N2** | Verdicts du moteur, sans données brutes |
| Clés/API tokens (vault, GoCardless, Anthropic) | **N1** | Variables d'environnement uniquement (jamais en base, jamais loggés) |
| Journal d'accès | **N1** | Consultable uniquement par le profil concerné |

## Garanties vérifiées par des tests

- `internal/twin` : `TestAnonymizeNeverLeaksExactAmounts` — aucun montant
  exact ne survit à l'anonymisation.
- `internal/ai` : `TestRouterCloudRequiresConsent`,
  `TestRouterCloudRequiresAnonymizedPrompt` — le cloud n'est jamais appelé
  sans consentement NI sans variante anonymisée.
- `internal/vault` : chiffrement au repos vérifié (aller-retour, altération,
  mauvaise clé) + contrôle en base lors des E2E.

## Le niveau N1 de la cascade (EIA-001)

Les questions simples de l'assistant sont d'abord tentées sur le **modèle
Apple Intelligence local de l'iPhone** (`LocalAI`, FoundationModels) : le
contexte chiffré calculé par le moteur ne quitte alors jamais l'appareil.
Indisponible → cascade N2 (homelab, données complètes) → N3 (cloud,
anonymisé + consenti) → repli déterministe.
