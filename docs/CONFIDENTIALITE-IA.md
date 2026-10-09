# Contrat de confidentialité et assistant

Version du 9 octobre 2026. Le backend des clients est imposé sur `https://opale.vaycode.com` : les données financières et questions sont traitées par le service Vaycode, indépendamment du fournisseur IA. Le moteur financier calcule ; les modèles sélectionnent ou interprètent, sans fabriquer les chiffres affichés.

## Sortie cloud autorisée

Seul `ai.CloudFacts` franchit la frontière cloud : une intention issue d’un catalogue contrôlé, patrimoine et cash en milliers d’euros, revenus et dépenses en centaines d’euros, score et taux d’épargne entier. Ce sont des **données sensibles minimisées**, pas une garantie d’anonymat. Aucun nom, question brute, libellé, banque, IBAN, catégorie, objectif nommé, document, note ou historique conversationnel n’est inclus. Un champ ajouté ailleurs n’est jamais transmis par défaut.

Le cloud exige les trois conditions : configuration globale active, profil N2, consentement `allow_cloud` à cet appel. N1 et N3 l’interdisent. Le homelab privé reste possible selon le cahier des charges ; l’IA sur iPhone fonctionne sur un appareil compatible. Une demande arbitraire est traitée par le parseur déterministe ou interprétée par le homelab en une intention strictement validée. Si cette transformation échoue, l’assistant demande une précision ou explique son périmètre ; il ne transmet pas le texte au cloud.

Les fournisseurs ne peuvent pas déclencher de mutation, accéder à d’autres profils ou changer le consentement. L’historique est fourni comme donnée non fiable, limité à 20 messages de 4 000 caractères maximum. Il reste en mémoire des clients, isolé par profil, effaçable et purgé au changement/déconnexion. Les demandes de données interrogent les agrégats exhaustifs du backend.

## Réponses et replis

Les réponses possèdent `state` : `grounded`, `clarification_needed` ou `unsupported`, ainsi que `provider_state` qui distingue fournisseur disponible, indisponible ou réponse invalide. Une réponse déterministe reste fondée même quand aucun modèle ne fonctionne ; elle est identifiée comme telle avec `tier:data`.

Les faits portent identifiant, unité, période, provenance, texte serveur et, pour un montant, `value_cents`. Le modèle renvoie uniquement des identifiants de faits, d’explications et de destinations autorisés. Le texte visible est assemblé à partir des faits du moteur et d’un catalogue explicatif contrôlé ; les actions ouvrent des écrans, sans mutation ni ordre financier. Un chiffre, un champ supplémentaire, un identifiant inconnu ou du texte libre dans la réponse fournisseur invalide cette réponse. L’utilisateur reçoit alors les faits déterministes et un état fournisseur explicite.

Les explications intégrées restent pédagogiques : les notions de diversification et d’ETF suivent [les repères de l’AMF](https://www.amf-france.org/fr/etf-exchange-traded-fund), et la réserve disponible [ceux du portail Banque de France](https://www.mesquestionsdargent.fr/pourquoi-epargner/une-epargne-de-precaution), vérifiés le 9 octobre 2026. Elles ne sélectionnent aucun produit pour l’utilisateur.

Les demandes sont annulables et bornées en temps. Les erreurs et décisions de routage sont journalisées sans payload, question ou montant. Les documents restent N3 et ne sont pas une entrée du modèle cloud. Le mode discret masque les réponses, leurs chiffres et leurs représentations accessibles.

## Choix de moteur et préparation

Le chat iOS ne lance plus Foundation Models avant le serveur. Les fonctions locales de l’iPhone restent séparées (lecture de documents, libellés). Le choix de fournisseur est propre au profil : `auto`, `homelab` ou `cloud`. Le mode homelab interdit toute cascade cloud ; le mode cloud saute le fournisseur privé et ne lui transmet pas la question pour interprétation. Aucun choix de mode ne vaut consentement. Les requêtes de données restent déterministes ; certaines explications courantes sont intégrées et identifiées `tier:guide`, sans appel fournisseur.

Le statut distingue homelab configuré, serveur/modèle joignable, clé cloud renseignée, interrupteur cloud et autorisation du profil. Ollama vérifie que le modèle est installé, supporte un jeton Bearer de proxy privé et demande un JSON structuré. Le modèle Claude est configurable côté serveur. Aucun test local ne certifie les performances du PC personnel ni l’accès au compte cloud ; [les étapes de connexion et recette](COOLIFY.md#choisir-lia--pc-windows-ou-cloud) restent explicites.

## Preuves et validation extérieure

Le nettoyage de libellé est une proposition distincte, sans calcul : sur iOS, Foundation Models travaille sur l’appareil ; sur le web, une action explicite appelle uniquement le homelab privé. `POST /v1/transactions/{id}/label-suggestion` vérifie d’abord le propriétaire et transmet seulement le libellé bancaire, sans montant, note, compte ou historique. Il n’existe aucune cascade cloud pour cette route, y compris en profil N2. Une réponse JSON stricte, sans champ supplémentaire et limitée à 120 caractères, est nécessaire. L’interface présente le résultat avant confirmation ; seul le PATCH explicite enregistre le nouveau libellé. L’indisponibilité et une proposition invalide laissent la transaction intacte. Le test `api.TestLabelSuggestionPrivatePreviewAndOwnership` intercepte le vrai client HTTP Ollama sur un serveur local, vérifie les trois politiques de profil, les refus croisés et l’absence d’écriture ou d’appel cloud.

`internal/ai/cloud_http_test.go` intercepte une requête HTTP produite par le véritable client SDK vers un serveur local : champs sensibles injectés, contenu autorisé seulement, zéro appel après désactivation et rejet d’affirmations inventées. `internal/api/integrity_test.go` vérifie les règles de profil, consentement et interrupteur global. Les tests de routeur couvrent indisponibilité et cascade.

Cela ne valide pas un modèle hébergé réel. Pour terminer la recette sur un fournisseur configuré : utiliser un profil exclusivement synthétique, examiner les requêtes sortantes, vérifier les états d’échec et de reprise, puis désactiver le cloud et confirmer l’absence totale de trafic. Aucun secret ni donnée financière personnelle n’a été envoyé pendant les tests de cette livraison.
