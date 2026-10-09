# Confidentialité et localisation des données

État du 9 octobre 2026. **Le service Opale de Vaycode héberge les profils et les données financières**. Les clients utilisent `https://opale.vaycode.com` sans choix de serveur. « Privé » ne signifie pas que toutes les données restent sur l’iPhone ou au domicile ; le serveur Go traite les opérations et PostgreSQL les conserve.

## Où sont les données ?

| Donnée | Localisation et protection |
|---|---|
| Profil, actifs/dettes, transactions, contrats, revenus, objectifs | PostgreSQL côté Vaycode ; lectures et écritures contrôlées par profil ; HTTPS entre client et service |
| PIN et sessions | Empreintes serveur ; jeton iOS dans Keychain lié au backend, jeton web dans sessionStorage du seul onglet |
| Documents déposés au coffre | Contenu AES-256-GCM dans PostgreSQL, clé secrète serveur séparée ; téléchargements/exports lisibles autorisés |
| Facture/paie lue pour préremplissage iOS | Lecture PDF/OCR sur l’appareil ; pas de dépôt, copie persistante ou appel IA distant ; seuls les champs confirmés sont ensuite enregistrés |
| Cache financier iOS | Fichiers protégés, isolés par serveur/profil ; lecture après authentification locale, pas de file de mutations hors ligne |
| Web | Pas de cache financier persistant hors session, ni service worker hors ligne ; écran verrouillé à froid, au retour d’onglet et après inactivité |
| Favoris de l’explorateur/scénarios | Données locales séparées par profil ; ne font pas tous partie de l’export API |
| Conversation | Historique client limité ; questions transmises au serveur Opale pour traitement, sans base de conversations complète dédiée |
| Fournisseur IA privé | Le serveur peut joindre Ollama sur le PC Windows via réseau privé ; prompts prévus par le mode utilisé, à protéger comme des données sensibles |
| Fournisseur IA cloud | Seulement intention contrôlée et agrégats minimisés après les trois autorisations ; jamais question brute, document ou historique complet |
| Banque | Consentement chez le fournisseur/banque ; Opale conserve lien/comptes/opérations, pas le mot de passe bancaire |
| Clés et tokens fournisseur | Secrets d’exploitation serveur ; jamais saisis dans l’app ou committés |
| Widget | Instantané App Group sans réseau ni secret ; effacé au changement de session/déconnexion |

## Politique IA du profil

Les identifiants historiques N1/N2/N3 contrôlent **l’usage du cloud IA**, pas la localisation générale de la base financière. N1 et N3 interdisent le cloud. N2 permet uniquement la demande explicitement consentie, si l’administrateur a également activé/configuré le fournisseur.

Seul le contrat `ai.CloudFacts` est autorisé à franchir cette frontière. Montants/ratios sont arrondis et minimisés ; cela ne garantit pas l’anonymat absolu. Le détail des champs, des consentements et replis est dans [CONFIDENTIALITE-IA.md](CONFIDENTIALITE-IA.md). Le chat n’est plus intercepté automatiquement par le petit modèle de l’iPhone ; Foundation Models reste une suggestion locale explicite de catégorisation.

## Accès, partage et copies

Les droits sont contrôlés par le profil issu du bearer. Le partage d’une opération dans un espace n’ouvre pas le patrimoine privé. Un contact n’a pas d’accès implicite ; un droit d’urgence doit être sélectionné, activé, non expiré et non révoqué.

Le mode discret retire les données sensibles visibles et accessibles ; le verrouillage masque aussi les aperçus d’arrière-plan. Ce ne sont pas des protections d’une copie déjà exportée. Les téléchargements ZIP/documents et les sauvegardes doivent être chiffrés et conservés séparément. La révocation d’un droit ne peut rappeler une copie déjà téléchargée.

Les journaux d’accès privés tracent les événements sensibles. Les métriques sont bornées par route/méthode/statut sans identifiant ni libellé et ne sont pas exposées par nginx public. Le chiffrement des documents ne chiffre pas automatiquement tous les champs de PostgreSQL ; protéger le volume et les dumps est une responsabilité d’exploitation.

## Vérifications et limites

Tests serveur : isolation, consentement, minimisation, coffre altéré/mauvaise clé, export, droits d’urgence. Tests clients : jeton lié au backend, ancienne URL ignorée, changement de profil, 401, verrouillage, mode discret et téléchargement tardif. Références dans [les fiches](README.md), [livraison iOS](DELIVERY-IOS.md) et [livraison web](DELIVERY-WEB.md).

Les tests synthétiques ne constituent pas un audit de sécurité externe, une certification réglementaire ou une validation de chaque fournisseur réel. Les données personnelles ne sont jamais nécessaires aux tests d’intégration.
