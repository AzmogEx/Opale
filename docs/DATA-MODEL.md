# Opale — Modèle de données

État livré le 8 octobre 2026, migrations `0001` à `0021`. Le schéma exact et ses contraintes sont définis dans `backend/internal/migrations/`. Les conventions de calcul sont détaillées dans [CONVENTIONS-FINANCIERES.md](CONVENTIONS-FINANCIERES.md), les validations dans [LIVRAISON.md](LIVRAISON.md).

Les montants monétaires sont des `BIGINT` en **unités mineures de la devise native** : EUR deux décimales, JPY zéro, KWD trois. Le nom historique `*_cents` ne signifie donc pas systématiquement centimes EUR. `currency_exponent` et le type Go `money.Cents` contrôlent conversions, arrondis et débordements. Les taux de change et quantités utilisent leurs précisions explicites ; aucun taux manquant n’est remplacé par une parité implicite.

## Entités persistées

| Tables | Rôle et relations |
|---|---|
| `profiles`, `sessions` | Profil, PIN haché, politique de confidentialité, démo explicitement marquée et expirante ; sessions opaques hachées et révocables |
| `assets`, `liabilities`, `valuations` | Actifs/passifs de premier rang, devise, date d’archive, création initiale idempotente ; historique de valorisations de clôture lié au même propriétaire |
| `transactions`, `categories`, `merchant_rules` | Compte, montant natif, date civile, libellés actuel/brut, catégorie et règle apprise ; type de flux, état bancaire, virement lié et dette remboursée explicites |
| `imported_operations` | Identité durable d’une opération source, indépendante des lignes visibles après ventilation ou suppression |
| `envelopes`, `goals` | Budgets par catégorie et objectifs avec rythme d’épargne affecté ; allocations sérialisées et plafonnées à la capacité observée |
| `calendar_rules`, `calendar_occurrences`, `recurring_exclusions` | Échéance ponctuelle ou série hebdomadaire/mensuelle/trimestrielle/annuelle, modification d’occurrence, exclusion et lien vers la transaction réalisée |
| `property_details`, `object_details`, `company_details` | Informations spécialisées d’un actif ; bien/crédit et société/CCA liés explicitement au même propriétaire |
| `investment_flows`, `investment_coverage` | Apports, retraits, distributions, frais externes et confirmation de couverture nécessaires à une performance interprétable |
| `documents`, `contacts`, `beneficiaries`, `emergency_grants` | Documents AES-GCM, contacts, bénéficiaires liés aux contrats, droits d’urgence limités à une sélection d’actifs/documents, activation/expiration/révocation |
| `spaces`, `space_members` | Partage explicite des seuls mouvements sélectionnés ; retrait effectif des droits et détachement des opérations du membre retiré |
| `fx_rates`, `fx_history`, `profile_fx_history` | Taux publics avec date/source de publication et historique ; taux privés datés isolés par profil et prioritaires dans ses calculs |
| `bank_links`, `bank_accounts`, `bank_pending`, `bank_account_bindings` | Consentement et état de synchronisation, bail de travail/reprise, association distincte de chaque compte, observations provisoires et mapping conservé après déconnexion |
| `monthly_snapshots`, `allocation_targets`, `custom_alerts` | Observations mensuelles datées, répartition cible et seuils personnalisés |
| `push_tokens`, `push_deliveries` | Jetons liés au profil et à l’environnement APNs ; bail et déduplication persistante des notifications |
| `access_log`, `schema_migrations` | Événements d’accès et versions de schéma ; les logs de routage IA et métriques sont séparés des données financières |

Les scénarios enregistrés restent dans les clients, isolés par profil ; les conversations sont limitées, effaçables et ne deviennent pas une table exportée. Les calculs de projection, décisions, fiscalité, risques et assistant utilisent les données autorisées sans créer de copies persistées de leurs résultats par défaut.

## Valeurs et historique

Pour un compte, la valeur courante part de la dernière valorisation de clôture applicable et ajoute les mouvements comptabilisés strictement postérieurs à sa date. Les opérations du même jour sont déjà incluses dans la clôture. Les opérations futures et provisoires ne changent pas le solde courant. Pour une dette, les remboursements de principal liés réduisent le capital restant dû ; les intérêts et frais restent des dépenses.

Les fonctions SQL `current_asset_value`, `current_liability_value` et `amount_eur`, ainsi que la vue `financial_transactions`, partagent ces conventions entre agrégats. Un virement interne déplace les valeurs des comptes sans gonfler les revenus/dépenses ; ses frais restent comptés. Les valeurs natives sont converties avant agrégation en EUR. Les positions sans valorisation sont signalées comme incomplètes.

L’archive exclut la position des vues courantes selon sa date tout en préservant l’historique. La suppression d’un compte contenant des mouvements est refusée : archiver le compte ou corriger explicitement ses opérations. Les corrections de valorisations et remboursements de dette vérifient tous les jalons historiques concernés dans une transaction ; une valorisation ultérieure ne peut pas masquer un capital négatif antérieur.

## Isolation et atomicité

Les lectures/écritures sont filtrées par le profil issu de la session. Les clés étrangères composites `(profile_id, id)` protègent les relations compatibles avec ce modèle ; les contrôles serveur complètent les règles de catégories, espaces, bénéficiaires et droits d’urgence. Un contact ne reçoit aucun accès implicite.

La création avec valeur initiale, les virements et leurs frais, la ventilation, les imports et les mutations liées sont transactionnels. Les imports gardent leur identité durable après ventilation ; un changement fournisseur incompatible est refusé, sans recréation silencieuse. L’export utilise un instantané SQL cohérent, une liste explicite de tables et un manifeste final. Il exclut PIN, sessions et jetons secrets ; voir [le contrat d’export](CONVENTIONS-FINANCIERES.md#export).

Les migrations sont sérialisées et appliquées dans une transaction. Une incohérence historique de propriétaire ou d’échelle monétaire bloque la migration sans suppression ni réattribution automatique. Le retour arrière de `0021` refuse les règles trimestrielles existantes ; pour les migrations destructrices, la récupération passe par une sauvegarde restaurée dans une nouvelle base, selon [EXPLOITATION.md](EXPLOITATION.md).
