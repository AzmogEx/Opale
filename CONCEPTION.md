# Opale — Conception complète

> **Opale** — Application personnelle de **gestion de patrimoine** (pas un simple
> budget), belle, intelligente et privée, auto-hébergée sur le homelab.
>
> *Opale : une pierre précieuse aux reflets changeants — la valeur, la profondeur,
> et l'esthétique « liquid glass » du produit.*

---

## Contexte

Construire une application de finances personnelles **bien supérieure aux apps du
marché** (Bankin', Linxo, et même Monarch / Copilot / Empower). Le projet est parti
d'une app de budget pour devenir une **plateforme de gestion de patrimoine personnel**
centrée sur le **pilotage de vie financière**, pas sur la comptabilité.

Le produit sera **construit principalement par IA (Claude Fable 5)**, le porteur du
projet jouant le rôle de product owner / directeur.

**Cibles plateformes :** iOS natif (SwiftUI) ultra-optimisé + Web. Backend commun
auto-hébergé sur le homelab. IA hybride : RTX 5080 en local + Fable 5 pour les
analyses lourdes.

---

## 1. Vision & positionnement

> **Pas « où est parti mon argent ce mois-ci », mais « combien je vaux, et où je vais ».**

L'écran d'accueil n'affiche pas un solde de compte courant, mais le **patrimoine
net** et sa **trajectoire**. Le produit répond à la question que presque aucune app
ne traite : *« Combien je vaux aujourd'hui, et combien je vaudrai dans 5, 10, 20 ans ? »*

**Triangle différenciateur : Beau × Intelligent × Privé.** Aucune app n'a les trois
en même temps.

| Problème des apps actuelles | Réponse d'Opale |
|---|---|
| Moches, lentes, web déguisé en app | Natif SwiftUI, fluide, Liquid Glass, animations satisfaisantes |
| Catégorisation faible | IA locale qui catégorise vraiment et apprend |
| Historique passif | Pilotage : enveloppes, projections, décisions |
| Vendent les données | Self-hosted, chiffré, zéro tiers — argument de marque |
| Suivi de dépenses uniquement | Patrimoine, investissements, immobilier, objectifs de vie |
| Synchro bancaire fragile | Robuste + import manuel en secours |

## 2. Public cible

- **Principalement le porteur du projet** (usage perso).
- **Aussi ses parents / sa famille** → multi-profil léger.
- Pas d'app publique au départ (pas d'inscription, pas de scaling, RGPD léger).
- Modèle : **profils séparés**, données privées par profil, déverrouillage Face ID,
  espace partagé optionnel plus tard.

## 3. Principes directeurs

1. **Beau et simple d'abord** — l'expérience attire (design Revolut-like).
2. **Intelligent et proactif dessous** — l'IA retient (Financial Twin, décisions).
3. **Privé par conception** — les données ne quittent jamais le homelab.
4. **Simple par défaut, puissant si on creuse** — couches optionnelles.
5. **Le chiffre est roi** — patrimoine net lisible en un coup d'œil.

## 4. Architecture en 5 onglets

```
┌─────────┬─────────┬───────────┬────────────┬───────────┐
│ Accueil │  Flux   │ Patrimoine│ Projection │ Assistant │
├─────────┼─────────┼───────────┼────────────┼───────────┤
│Patrim.  │Revenus  │Immobilier │Objectifs   │Recherche  │
│net      │Dépenses │Placements │Simulations │naturelle  │
│Cash     │Envelop. │Dettes     │Retraite    │Conseils   │
│dispo    │Calendr. │Objets     │Indépend.   │Décisions  │
│Score    │Abonnem. │Entreprise │Timeline    │Alertes    │
└─────────┴─────────┴───────────┴────────────┴───────────┘
```

## 5. Catalogue complet des fonctionnalités

### Onglet 1 — Accueil
- **Patrimoine net total** (Actifs − Dettes) avec évolution mensuelle, compteur animé.
- **Cash disponible** réel.
- **Score de santé financière** (/100).
- **Jalons & paliers de patrimoine** (gamification : 10k, 100k, fonds d'urgence…).
- **Mode discret** (flouter les montants d'un geste).

### Onglet 2 — Flux
- **Revenus / dépenses**, recherche, filtres, catégorisation auto (IA, apprend).
- **Enveloppes budgétaires** (allocation à l'avance, style YNAB — couche optionnelle).
- **Calendrier financier** (flux futurs datés : salaires, crédits, abonnements, impôts).
- **Détection d'abonnements** récurrents.
- **Cashflow futur** : « ton cash dispo réel au 30 juin sera ~2 430 € » (intègre
  salaire, loyers, abonnements, crédits, impôts, dépenses moyennes + exceptionnelles).

### Onglet 3 — Patrimoine
- **Actifs** : comptes, livrets, assurance-vie, PEA, CTO, crypto, immobilier, or,
  participation entreprise.
- **Passifs** : crédit immo, auto, conso.
- **Centre immobilier** : valeur d'achat / estimée, rendement, crédit restant,
  cashflow, taxe foncière (mini-app immobilière intégrée).
- **Centre investissement** (suivi, pas trading) : PEA, CTO, AV, crypto + répartition
  (actions / obligations / immobilier / liquidités).
- **Valeur des objets** : montres, bijoux, or, voitures, œuvres, luxe, matériel pro,
  parts de société, collections (valeur achat/estimée, photo, facture, assurance).
- **Module entrepreneur** (V2/V3) : valeur entreprise, parts, compte courant
  d'associé, dividendes, rému, fiscalité prévisionnelle, tréso perso vs pro,
  scénario « si je me verse X €/mois ».

### Onglet 4 — Projection
- **Date d'indépendance financière** (FIRE rendu beau : « libre à 47 ans »).
- **Simulateur de vie** : « si j'épargne 500 €/mois → 38k à 5 ans, 82k à 10 ans… ».
- **Comparateur de scénarios** (deux futurs côte à côte : louer vs acheter, etc.).
- **Objectifs intelligents** (résidence, retraite, voyage, voiture, création
  d'entreprise) : date prévue, retard/avance, montant restant.
- **Projections en euros constants** (corrigées de l'inflation).
- **Timeline patrimoniale** : vue chronologique de la vie financière (2026 fonds
  d'urgence atteint → 2028 apport immo → 2032 bien locatif → 2040 indépendance).

### Onglet 5 — Assistant IA
- **Recherche en langage naturel** (« combien en courses en mars ? »).
- **IA patrimoniale contextuelle** : connaît toute la situation, raisonne dessus.
- **Mode Décision** : « acheter ou louer ? », « cash ou crédit ? », « rembourser ou
  investir ? », « quitter mon job ? » → impact immédiat / 5 ans / 10 ans, risque,
  reco, scénarios prudent / normal / ambitieux.
- **Alertes intelligentes** (dépassement enveloppe, solde bas prévu…).

### Briques transverses
- **Financial Twin** — double financier de l'utilisateur (revenus, charges, actifs,
  dettes, objectifs, habitudes, risques). Moteur de simulation = cœur du produit.
- **Radar de risques** — détecte ce qui peut casser le plan (cash dormant,
  surendettement, fonds d'urgence insuffisant, dépendance à un revenu, immobilier
  trop dominant, illiquidité, dépenses fixes trop hautes…).
- **Bilan mensuel intelligent** — résumé humain rédigé par l'IA en fin de mois
  (patrimoine +2 140 €, taux d'épargne 18 %, dépenses inutiles 312 €, meilleure
  décision, point faible).
- **Plan de transmission** (mode Succession / famille) — contrats importants,
  bénéficiaires AV, biens, dettes, documents, contacts (notaire/banque/assurance),
  accès d'urgence à un proche de confiance. Premium et différenciant.
- **Coffre-fort patrimonial** — stockage sécurisé des documents (actes, AV, contrats,
  carte grise, passeport, testament, diagnostics).

## 6. Fonctionnalités signature (le fossé concurrentiel)

1. **Financial Twin + Mode Décision** — le cœur : l'argent sert à décider.
2. **IA patrimoniale contextuelle** — inimitable sans données complètes + IA + privé.
3. **Date d'indépendance financière** — le hook addictif (un chiffre qui motive).
4. **Confidentialité extrême** — axe de marque (voir §10).

## 7. Système de design

**Référence : Revolut** (moderne, ultra-fluide, satisfaisant) **+ Liquid Glass**
(matériau translucide iOS 26, natif SwiftUI — alignement parfait avec la stack).
Le nom **Opale** inspire l'identité : reflets nacrés, profondeur, jeux de lumière.

| Mécanique | Effet |
|---|---|
| Compteurs animés | Le patrimoine net défile en montant à l'ouverture |
| Haptique (Taptic Engine) | Vibration subtile à chaque action clé |
| Liquid Glass en couches | Cartes translucides, profondeur, flou dynamique, reflets |
| Spring physics | Tout glisse/rebondit naturellement |
| Célébrations | Confettis/particules à l'atteinte d'objectifs |
| Graphes scrubables | Glisser le doigt sur la courbe, les chiffres suivent |
| Cartes draggables | Glisser pour fermer, parallaxe, empilement à la Wallet |
| Système de thèmes | Couleurs d'accent multiples + clair/sombre/auto + ambiances |
| Sound design (option) | Petits sons discrets et satisfaisants |

**Objectif :** ouvrir Opale = micro-récompense → création d'habitude.
**Identité visuelle** (à finaliser) : palette inspirée de l'opale (reflets nacrés,
dégradés irisés), typo soignée, solde en grand, mode sombre premium.

## 8. Architecture technique

```
┌─────────────┐     ┌─────────────┐
│ iOS SwiftUI │     │ Web (React) │
└──────┬──────┘     └──────┬──────┘
       └─────────┬─────────┘
            API (REST/GraphQL)
                 │
       ┌─────────▼──────────┐
       │  Backend Go + PG   │  ← homelab
       └─────────┬──────────┘
                 │
       ┌─────────▼──────────┐
       │ IA : RTX 5080 local │  + Fable 5 pour analyses lourdes
       └────────────────────┘
```

- **iOS** : SwiftUI natif (perf/beauté maximales) — ✅ décidé.
- **Web** : React (ou Svelte) + Tailwind, UI séparée, optimisée à fond. *(à trancher)*
- **Backend** : **Go** (API REST/GraphQL) — perf, simple, compétence recherchée. ✅ décidé.
- **Base** : PostgreSQL (ACID, précision décimale).
- **IA — stratégie hybride** ✅ décidé :
  - **RTX 5080 (local)** → tâches fréquentes : catégorisation auto, recherche en
    langage naturel, insights simples. Privé, gratuit.
  - **Fable 5 (API)** → raisonnement complexe : Mode Décision, bilans mensuels,
    analyses patrimoniales. Données minimisées/anonymisées avant envoi.
- **Données bancaires** : démarrer en **import CSV/OFX**, puis **GoCardless Bank
  Account Data** (ex-Nordigen, gratuit, DSP2). Actifs non bancaires (immo, objets,
  crypto, entreprise) saisis manuellement ou via connecteurs dédiés plus tard.
- **Hébergement** : homelab (Docker/Coolify déjà en place).

### ⚠️ Règle d'or technique
**Ne jamais stocker l'argent en `float`/`double`.** Montants en **centimes (entiers)**
ou type `DECIMAL`. Règle absolue de toute app financière.

## 9. Modèle de données (conceptuel)

Le repositionnement « patrimoine » fait des **actifs et passifs des entités de
premier rang**, chacun avec un **historique de valorisations** (pour tracer
l'évolution du patrimoine dans le temps). Une transaction n'est plus le centre :
c'est un mouvement qui affecte un actif/compte.

Entités principales (à détailler) :
- `Profile` (utilisateur : toi, parents)
- `Account` / `Asset` (comptes, livrets, placements, immobilier, objets, entreprise)
- `Liability` (crédits)
- `Valuation` (snapshot daté de la valeur d'un actif/passif → courbe patrimoine net)
- `Transaction` (montant en centimes, date, libellé, compte, catégorie, split)
- `Category` / `Subcategory`, `Tag`, `Split`
- `Envelope` (budget), `RecurringRule` (abonnements/revenus récurrents)
- `Goal` (objectif de vie), `Scenario` (simulation / décision)
- `Document` (coffre-fort), `Beneficiary` / `Contact` (plan de transmission)

## 10. Confidentialité & sécurité (axe de marque)

> « Votre patrimoine ne devrait pas devenir le business model de quelqu'un d'autre. »

- Données chiffrées, IA locale.
- Face ID par profil, verrouillage automatique.
- Mode discret (flou des montants).
- Export complet des données (portabilité).
- Aucune revente de données.
- Journal d'accès (« qui a consulté quoi »).

## 11. Timeline de développement (par paliers)

Construction dans cet ordre — chaque palier est utilisable et livre une vraie valeur.
Pas de dates calendaires (projet perso, rythme libre) : on avance palier par palier.

| Palier | Nom | Contenu | Pourquoi à ce stade |
|---|---|---|---|
| **P0** | Fondations | Repo, backend Go + Postgres, modèle de données (actifs/passifs/valorisations/transactions), auth profils, CI/CD homelab | La colonne vertébrale technique |
| **P1** | La colonne (le « waouh ») | Accueil : **patrimoine net** + saisie manuelle actifs/dettes + graphe d'évolution. Design system de base (Liquid Glass, thème, compteurs animés) | Impact émotionnel immédiat sans dépendances |
| **P2** | Le futur | Projection : **date d'indépendance** + 1 simulateur d'épargne | Hook addictif, peu de code, gros effet |
| **P3** | Le quotidien | Flux : transactions, **import CSV**, catégorisation IA, calendrier financier, cashflow futur | Le carburant des données |
| **P4** | Le pilotage | Enveloppes, objectifs intelligents, score de santé, alertes | Le contrôle du budget |
| **P5** | Le cerveau | **Assistant IA patrimonial** + **Mode Décision** + Financial Twin + bilan mensuel + radar de risques | Le vrai fossé concurrentiel |
| **P6** | La profondeur | Centre immobilier, investissements, valeur des objets, timeline patrimoniale, coffre-fort, plan de transmission | Modules riches, un par un |
| **P7** | Le confort | Synchro bancaire (GoCardless), comparateur de scénarios, module entrepreneur, widgets, Apple Watch, espace partagé | Quand le cœur est solide |

**App web** : suit en parallèle à partir de P1 (dashboard patrimoine), priorité
secondaire à iOS au début.

## 12. Décisions

**Tranchées ✅**
- **Nom** : **Opale**.
- **iOS** : SwiftUI natif.
- **Backend** : Go.
- **Base de données** : PostgreSQL.
- **Stratégie IA** : hybride (RTX locale pour catégorisation/recherche, Fable 5 pour
  décisions/analyses).
- **Dépôt** : nouveau dépôt dédié (sur la forge).

**Encore ouvertes**
- **Web** : React ou Svelte.
- **Identité visuelle** : palette précise (autour de l'opale), typo.

## 13. Références produit

Monarch Money, Copilot Money, Empower (Personal Capital) pour le patrimoine ;
Revolut pour le design et la fluidité. Objectif : approche plus moderne, centrée
utilisateur, privée et auto-hébergée.

## 14. Prochaines étapes

1. ✅ Nom choisi : **Opale**.
2. Trancher : framework web, identité visuelle (palette/typo).
3. Créer le dépôt dédié sur la forge et y déposer ce document.
4. Démarrer **P0** : projet Go + Postgres, schéma de base, modèle de données.
