# Opale — Modèle de données

Ce document décrit le modèle de données complet (cf. cahier des charges §9) et
indique, pour chaque entité, son **palier de livraison**. Les actifs et passifs
sont des **entités de premier rang**, chacune avec un **historique de
valorisations** qui alimente la courbe du patrimoine net.

> **Règle d'or (ENF-007)** : tout montant est un **entier en centimes** (`BIGINT`).
> Jamais de `float`/`double`. Côté Go, le type `money.Cents` est la seule porte
> d'entrée pour manipuler des montants.

## Implémenté — migration `0001_init` (palier P0)

| Table | Rôle | Exigences |
|---|---|---|
| `profiles` | Profils cloisonnés (utilisateur, parents) + niveau de confidentialité par défaut | EF-001 |
| `sessions` | Jetons de session opaques hachés, avec expiration | EF-002 |
| `assets` | Actifs : comptes, livrets, AV, PEA, CTO, crypto, immo, or, objets, parts | EF-030, EF-072 |
| `liabilities` | Passifs : crédits immo/auto/conso | EF-031 |
| `valuations` | Snapshot daté de la valeur d'un actif **ou** d'un passif (centimes) | EF-032 |
| `schema_migrations` | Suivi des migrations appliquées | — |

**Calcul du patrimoine net (CA-1)** : pour chaque actif et passif non archivé, on
prend sa dernière valorisation (`DISTINCT ON ... ORDER BY as_of DESC`), puis
`net = Σ actifs − Σ passifs`. Déterministe, en centimes, jamais via l'IA
(EIA-040/041).

## Cloisonnement par profil (EF-001)

Chaque entité porte un `profile_id` et toutes les requêtes du store sont filtrées
par profil. Une donnée d'un profil n'est jamais visible depuis un autre.

## À venir (paliers ultérieurs)

| Entité | Palier | Notes |
|---|---|---|
| `Category` / `Subcategory`, `Tag` | P3 | Catégorisation (IA + manuelle) |
| `Transaction`, `Split` | P3 | Mouvements (centimes), import CSV/OFX |
| `RecurringRule` | P3 | Abonnements / revenus récurrents |
| `Envelope` | P4 | Enveloppes budgétaires |
| `Goal` | P4 | Objectifs de vie |
| `Scenario` | P5 | Simulations / Mode Décision |
| `AIRequestLog` | P5 | Trace de routage IA (niveau, motif, anonymisation) |
| `Document` | P6 | Coffre-fort (avec niveau de confidentialité N1/N2/N3) |
| `Beneficiary` / `Contact` | P6 | Plan de transmission |

Chaque nouvelle entité fera l'objet d'une migration `000N_*.up.sql` /
`.down.sql` dédiée, appliquée automatiquement au démarrage.
