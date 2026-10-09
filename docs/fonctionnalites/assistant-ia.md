# Assistant, guide intégré et fournisseurs IA

État du code au 9 octobre 2026. [Index](../README.md) · [API](../API.md).

## Utilisation

Assistant : poser une question ou utiliser un guide comme « Par où commencer ? ». Les réponses exposent les faits/provenances et des actions vers les modules utiles. Sur iPhone, « Choisir et connecter mon IA » propose Automatique, Mon PC ou Cloud et affiche les états de configuration/disponibilité.

## Données et comportement

Le chat passe par le backend Vaycode. Foundation Models n’intercepte plus ses questions ; la lecture documentaire et les suggestions locales restent des fonctions séparées. Sans modèle disponible, le parseur/guide et les explications contrôlées restent utilisables. Une sortie de modèle ne remplace pas les calculs financiers du moteur.

Ollama sur le PC Windows est joint par le serveur distant via un réseau privé ; son statut vérifie le modèle dans `/api/tags`. Mon PC ne cascade pas vers le cloud. Le cloud exige activation globale, profil N2 et consentement pour l’appel ; N1/N3 l’interdisent. Seuls intention contrôlée et agrégats minimisés/arrondis autorisés sont transmis, sans question brute, libellé, document ou historique. Le web limite la conversation en mémoire et la vide au changement de profil.

## Limites et configuration

Configurer les secrets et modèles côté API, jamais dans l’app. [COOLIFY.md](../COOLIFY.md#choisir-lia--pc-windows-ou-cloud) décrit PC Windows/Tailscale/Ollama et Claude. PC éteint, modèle absent, délai, clé invalide ou crédit insuffisant donnent un état/repli explicite. « Configuré » ne prouve pas un appel réel réussi. Agrégats minimisés ne signifie pas anonymat absolu.

## Vérification

Tests `internal/ai`, `assistant_modes_test.go`, tests d’intégrité/confidentialité et recette du guide iOS. Fournisseurs testés avec serveurs HTTP synthétiques ; le PC et la clé cloud personnels doivent être éprouvés après configuration. Contrat exhaustif des sorties : [CONFIDENTIALITE-IA.md](../CONFIDENTIALITE-IA.md).

## API et sources

- `GET /v1/assistant/status`
- `POST /v1/assistant/ask`

- [backend/internal/ai/ai.go](../../backend/internal/ai/ai.go)
- [backend/internal/api/brain.go](../../backend/internal/api/brain.go)
- [ios/Opale/Features/Assistant/AssistantSettingsView.swift](../../ios/Opale/Features/Assistant/AssistantSettingsView.swift)
- [ios/Opale/Features/Assistant/AssistantView.swift](../../ios/Opale/Features/Assistant/AssistantView.swift)
- [ios/Opale/Features/Auth/OnboardingView.swift](../../ios/Opale/Features/Auth/OnboardingView.swift)
- [web/src/routes/assistant/+page.svelte](../../web/src/routes/assistant/+page.svelte)
