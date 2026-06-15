# Opale — Cahier des charges

| | |
|---|---|
| **Projet** | Opale — application personnelle de gestion de patrimoine |
| **Version** | 1.0 |
| **Date** | 2026-06-15 |
| **Statut** | Conception validée — prêt pour P0 |
| **Auteur produit** | Adam (product owner) — réalisation assistée par IA (Fable 5) |
| **Documents liés** | `CONCEPTION.md` (vision) |

> Légende priorités (MoSCoW) : **M** = Must (indispensable), **S** = Should
> (important), **C** = Could (souhaitable), **W** = Won't now (hors périmètre initial).
> Colonne **Palier** = jalon de livraison prévu (P0→P7, voir §14).

---

## 1. Présentation & contexte

**Opale** est une application personnelle de **gestion de patrimoine** (et non un
simple gestionnaire de budget), **belle, intelligente et privée**, auto-hébergée sur
le homelab de l'utilisateur. Elle vise à dépasser les apps du marché (Bankin', Linxo,
Monarch, Copilot, Empower) en réunissant trois qualités qu'aucune ne combine :
**Beau × Intelligent × Privé**.

Le produit est **réalisé principalement par IA (Claude Fable 5)**, l'utilisateur
agissant comme product owner / directeur. Cibles : **iOS natif (SwiftUI)** +
**Web**, backend **Go** commun auto-hébergé, **IA hybride en cascade**.

## 2. Objectifs & vision

> **Pas « où est parti mon argent ce mois-ci », mais « combien je vaux, et où je vais ».**

- **O1** — Afficher et faire grandir le **patrimoine net** (actifs − dettes) et sa trajectoire.
- **O2** — Aider à **piloter** (enveloppes, objectifs, projections) et **décider** (Mode Décision).
- **O3** — Offrir une **expérience premium** (design Revolut + Liquid Glass, animations satisfaisantes).
- **O4** — Garantir la **confidentialité maximale** : données privées par défaut, IA locale d'abord.
- **O5** — Rester **simple par défaut, puissant si on creuse** (couches optionnelles).

## 3. Périmètre

**Inclus**
- Gestion patrimoine net (actifs/passifs/valorisations), flux, projections, IA patrimoniale.
- Multi-profil léger (utilisateur + parents/famille).
- iOS natif + Web. Backend + base auto-hébergés.
- IA hybride en cascade (local iPhone → homelab GPU → cloud) + moteur financier déterministe.

**Exclu (au démarrage)**
- App publique / inscription ouverte / scaling multi-tenant.
- Trading / passage d'ordres.
- Conseil financier réglementé (l'app informe, ne se substitue pas à un CGP).

## 4. Parties prenantes & personas

| Persona | Description | Besoins clés |
|---|---|---|
| **Adam** (principal) | Étudiant DevOps, tech-savvy, homelab + RTX | Vue patrimoine, projections, décisions, contrôle total des données |
| **Parents** (secondaire) | Usage familial, moins techniques | Simplicité, lisibilité, profil séparé et privé |
| **Proche de confiance** | Accès d'urgence (transmission) | Accès restreint et sécurisé en cas de besoin |

## 5. Exigences fonctionnelles

### 5.1 Transverse (socle)

| ID | Exigence | Prio | Palier |
|---|---|---|---|
| EF-001 | Multi-profil : profils séparés, données privées par profil | M | P0 |
| EF-002 | Authentification par profil (Face ID / code), verrouillage auto | M | P0 |
| EF-003 | Navigation à 5 onglets (Accueil / Flux / Patrimoine / Projection / Assistant) | M | P1 |
| EF-004 | Mode discret : flouter les montants d'un geste | S | P1 |
| EF-005 | Système de thèmes : couleurs d'accent + clair/sombre/auto | S | P1 |
| EF-006 | Export complet des données du profil (portabilité) | S | P4 |
| EF-007 | Espace partagé optionnel (dépenses communes famille) | C | P7 |
| EF-008 | Multi-devises | C | P7 |

### 5.2 Onglet Accueil

| ID | Exigence | Prio | Palier |
|---|---|---|---|
| EF-010 | Afficher le **patrimoine net total** (actifs − dettes) | M | P1 |
| EF-011 | Compteur animé du patrimoine net à l'ouverture | S | P1 |
| EF-012 | Évolution mensuelle du patrimoine (variation + %) | M | P1 |
| EF-013 | Graphe d'évolution du patrimoine net dans le temps, scrubable | M | P1 |
| EF-014 | Afficher le **cash disponible réel** | M | P1 |
| EF-015 | **Score de santé financière** (/100) avec points forts/faibles | S | P4 |
| EF-016 | Jalons & paliers de patrimoine (10k, 100k, fonds d'urgence) + célébrations | C | P4 |

### 5.3 Onglet Flux

| ID | Exigence | Prio | Palier |
|---|---|---|---|
| EF-020 | Lister revenus/dépenses, recherche, filtres | M | P3 |
| EF-021 | Import de transactions CSV/OFX | M | P3 |
| EF-022 | Catégorisation automatique (IA) + correction manuelle (apprentissage) | M | P3 |
| EF-023 | Nettoyage des libellés bancaires (IA) | S | P3 |
| EF-024 | Détail/édition d'une transaction, split multi-catégories | S | P3 |
| EF-025 | Calendrier financier (flux futurs datés) | S | P3 |
| EF-026 | Détection d'abonnements récurrents | S | P3 |
| EF-027 | **Cashflow futur** : cash dispo projeté à une date donnée | S | P3 |
| EF-028 | Enveloppes budgétaires (allocation à l'avance, optionnel) | S | P4 |

### 5.4 Onglet Patrimoine

| ID | Exigence | Prio | Palier |
|---|---|---|---|
| EF-030 | Saisie manuelle d'actifs (comptes, livrets, AV, PEA, CTO, crypto, immo, or, parts) | M | P1 |
| EF-031 | Saisie manuelle de passifs (crédits immo/auto/conso) | M | P1 |
| EF-032 | Historique de valorisations par actif/passif | M | P1 |
| EF-033 | Centre immobilier (valeur achat/estimée, rendement, crédit restant, cashflow, taxe) | C | P6 |
| EF-034 | Centre investissement (suivi PEA/CTO/AV/crypto + répartition) | C | P6 |
| EF-035 | Valeur des objets (montres, or, voitures, œuvres… + photo/facture/assurance) | C | P6 |
| EF-036 | Module entrepreneur (valeur société, parts, CCA, dividendes, fiscalité) | C | P7 |

### 5.5 Onglet Projection

| ID | Exigence | Prio | Palier |
|---|---|---|---|
| EF-040 | **Date d'indépendance financière** (FIRE) calculée et affichée | M | P2 |
| EF-041 | Simulateur d'épargne (« si j'épargne X €/mois → patrimoine à N ans ») | M | P2 |
| EF-042 | Objectifs intelligents (date prévue, retard/avance, montant restant) | S | P4 |
| EF-043 | Projections en euros constants (corrigées de l'inflation) | C | P4 |
| EF-044 | Comparateur de scénarios (deux futurs côte à côte) | C | P7 |
| EF-045 | Timeline patrimoniale (chronologie de la vie financière) | C | P6 |

### 5.6 Onglet Assistant IA & moteur de décision

| ID | Exigence | Prio | Palier |
|---|---|---|---|
| EF-050 | Recherche en langage naturel sur les données (« combien en courses en mars ? ») | S | P5 |
| EF-051 | IA patrimoniale contextuelle (connaît la situation complète, raisonne) | S | P5 |
| EF-052 | **Mode Décision** (acheter/louer, cash/crédit, rembourser/investir…) avec impacts 0/5/10 ans, risque, reco, scénarios prudent/normal/ambitieux | S | P5 |
| EF-053 | Alertes intelligentes (dépassement enveloppe, solde bas prévu…) | S | P4 |

### 5.7 Briques transverses

| ID | Exigence | Prio | Palier |
|---|---|---|---|
| EF-060 | **Financial Twin** : double financier (revenus, charges, actifs, dettes, objectifs, habitudes, risques) servant de moteur de simulation | S | P5 |
| EF-061 | **Radar de risques** (cash dormant, surendettement, fonds d'urgence, dépendance revenu, illiquidité…) | S | P5 |
| EF-062 | **Bilan mensuel intelligent** (résumé humain rédigé par l'IA) | S | P5 |
| EF-063 | **Plan de transmission** (contrats, bénéficiaires, contacts, accès d'urgence) | C | P6 |
| EF-064 | **Coffre-fort** patrimonial (stockage chiffré de documents) | C | P6 |

## 6. Stratégie IA hybride en cascade

> **Principe directeur : l'intelligence maximale, avec le minimum de données exposées.**
> Privé d'abord, puissant si nécessaire.

### 6.1 Architecture en cascade (3 niveaux)

```
Demande utilisateur
        ↓
   AI Router  ──────────────────────────────────┐
        ↓                                        │
┌──────────────────────────────┐                │
│ 1. iPhone local              │  rapide /       │
│ Apple Foundation Models      │  privé /        │
│                              │  offline        │
└──────────────┬───────────────┘                │
               ↓ si trop complexe               │
┌──────────────────────────────┐                │
│ 2. Homelab GPU (RTX 5080)    │  privé /        │
│ Modèles locaux quantifiés    │  plus puissant  │
└──────────────┬───────────────┘                │
               ↓ si indispo / limite            │
┌──────────────────────────────┐                │
│ 3. Cloud premium (Fable 5)   │  très          │
│ Données anonymisées          │  puissant /     │
│                              │  ponctuel       │
└──────────────────────────────┘ ───────────────┘
```

| ID | Exigence | Prio | Palier |
|---|---|---|---|
| EIA-001 | Niveau 1 — IA locale iPhone (Apple Foundation Models) pour tâches simples/fréquentes/sensibles | S | P5 |
| EIA-002 | Niveau 2 — IA homelab GPU pour analyses privées lourdes | S | P5 |
| EIA-003 | Niveau 3 — IA cloud premium (Fable 5) pour décisions complexes, données anonymisées | S | P5 |

### 6.2 AI Router (composant central)

| ID | Exigence | Prio | Palier |
|---|---|---|---|
| EIA-010 | Module **AI Router** décidant du niveau cible pour chaque demande | S | P5 |
| EIA-011 | Critères de routage : complexité, sensibilité, confidentialité requise, disponibilité homelab, coût cloud, temps de réponse attendu, taille du contexte, niveau de raisonnement | S | P5 |
| EIA-012 | Journalisation des décisions de routage (quel niveau, pourquoi) | C | P5 |

**Exemples de routage**
- « Classe cette transaction » → **iPhone local**
- « Pourquoi mon cash baisse ce mois-ci ? » → **iPhone ou homelab**
- « Compare acheter un appart à 280k vs rester locataire » → **homelab d'abord, cloud si besoin**
- « Analyse tout mon patrimoine et propose 3 stratégies » → **cloud possible, données anonymisées**

### 6.3 Répartition des tâches par niveau

| Niveau | Tâches |
|---|---|
| **1 — iPhone local** | Catégorisation, nettoyage de libellés, détection d'abonnements, résumé court, recherche NL simple, explication rapide d'un graphe, suggestion d'enveloppe, extraction simple de reçu |
| **2 — Homelab GPU** | Bilan mensuel complet, radar de risques, analyse patrimoniale globale, explication d'une baisse de patrimoine, résumé de documents, analyse immobilière, projection multi-hypothèses, comparaisons de scénarios simples, assistant à historique long |
| **3 — Cloud premium** | Acheter/louer, cash/crédit, rembourser/investir, achat immobilier complexe, transmission/succession, stratégie entrepreneur, arbitrage multi-actifs, analyse longue multi-hypothèses |

### 6.4 Fallback & expérience utilisateur

| ID | Exigence | Prio | Palier |
|---|---|---|---|
| EIA-020 | Fallback fluide : si niveau 1 insuffisant → « Je vais faire une analyse plus poussée. » → homelab | S | P5 |
| EIA-021 | Si homelab indisponible : proposer le cloud — « Analyse avancée indisponible en local. Utiliser le modèle cloud ? » | S | P5 |
| EIA-022 | Pour données sensibles : confirmation avant cloud — « Cette analyse nécessite un modèle externe. Les données seront anonymisées avant envoi. Continuer ? » | M | P5 |

### 6.5 Niveaux de confidentialité des données

| ID | Niveau | Règle | Exemples |
|---|---|---|---|
| EIA-030 | **N1 — Local only** | Jamais hors appareil/homelab | identité, documents, IBAN, contrats, fiscalité, succession, détails familiaux |
| EIA-031 | **N2 — Cloud anonymisé possible** | Envoi sans nom/banque/document brut | montants agrégés, ratios, objectifs, scénarios, hypothèses |
| EIA-032 | **N3 — Cloud interdit par défaut** | Jamais envoyé au cloud | documents uploadés, pièce d'identité, testament, contrats notariaux, données bancaires brutes |

**Exemple d'anonymisation (N2)** — au lieu d'envoyer
`Adam a 42 300 € chez BNP, 18 000 € sur PEA Boursorama…`, envoyer :
```
Profil A :
  cash 42k
  placements actions 18k
  revenu mensuel 3.2k
  charges fixes 1.4k
  objectif achat immobilier 250k
```

| ID | Exigence | Prio | Palier |
|---|---|---|---|
| EIA-033 | Moteur d'anonymisation appliqué automatiquement avant tout envoi cloud (N2) | M | P5 |
| EIA-034 | Classification de chaque donnée selon son niveau (N1/N2/N3) | M | P5 |

### 6.6 Moteur financier déterministe (non négociable)

> L'IA est le **cerveau conversationnel** ; le **calcul** reste dans le moteur
> financier déterministe. L'AI Router ne remplace pas le moteur de calcul.

```
Utilisateur pose une question
        ↓
IA comprend l'intention
        ↓
Moteur financier calcule (déterministe)
        ↓
IA explique le résultat
```

**Exemple** — « Puis-je acheter une voiture à 35 000 € ? »
→ le **moteur** calcule cash restant, impact patrimoine, impact objectif, impact date
d'indépendance, risque de liquidité → l'**IA** explique : « Oui, mais ton fonds
d'urgence passe de 9 à 4 mois… ».

| ID | Exigence | Prio | Palier |
|---|---|---|---|
| EIA-040 | Tous les calculs financiers passent par un moteur déterministe (jamais l'IA seule) | M | P2 |
| EIA-041 | L'IA ne produit jamais de chiffre financier non issu du moteur | M | P5 |
| EIA-042 | Le moteur expose des fonctions de calcul testables unitairement (impact patrimoine, projection, fonds d'urgence, date d'indépendance…) | M | P2 |

## 7. Exigences non-fonctionnelles

| ID | Domaine | Exigence | Prio |
|---|---|---|---|
| ENF-001 | Performance | UI iOS fluide (cible 60–120 fps), animations sans à-coups | M |
| ENF-002 | Performance | Réponse API < 300 ms pour les lectures courantes | S |
| ENF-003 | Sécurité | Données chiffrées au repos et en transit (TLS) | M |
| ENF-004 | Sécurité | Auth par profil (Face ID/code), verrouillage auto, journal d'accès | M |
| ENF-005 | Confidentialité | Aucune revente de données ; IA locale d'abord ; cloud anonymisé/opt-in | M |
| ENF-006 | Confidentialité | Respect strict des niveaux N1/N2/N3 (§6.5) | M |
| ENF-007 | Fiabilité | Montants en **centimes (entiers)** ou DECIMAL — jamais de float | M |
| ENF-008 | Disponibilité | Fonctionnement dégradé si homelab/cloud indisponibles (fallback) | S |
| ENF-009 | Portabilité | Export complet des données utilisateur | S |
| ENF-010 | Maintenabilité | Code testé (unitaire sur le moteur financier), CI/CD | M |
| ENF-011 | Observabilité | Logs structurés, métriques, suivi des routages IA | C |
| ENF-012 | Accessibilité | Contraste, Dynamic Type, VoiceOver (iOS) | S |
| ENF-013 | Conformité | RGPD léger (usage familial), pas de tiers traceurs | S |

## 8. Architecture technique

```
┌─────────────┐     ┌─────────────┐
│ iOS SwiftUI │     │ Web (React) │
└──────┬──────┘     └──────┬──────┘
       └─────────┬─────────┘
            API (REST/GraphQL)
                 │
       ┌─────────▼──────────┐      ┌──────────────────┐
       │  Backend Go + PG   │◄────►│  Moteur financier │ (déterministe)
       └─────────┬──────────┘      └──────────────────┘
                 │
          ┌──────▼───────┐
          │  AI Router   │
          └──┬───┬───┬───┘
   iPhone ◄──┘   │   └──► Cloud (Fable 5, anonymisé)
                 ▼
           Homelab GPU (RTX 5080)
```

- **iOS** : SwiftUI natif (+ Apple Foundation Models pour l'IA locale N1). ✅
- **Web** : React (ou Svelte) + Tailwind. *(framework à trancher)*
- **Backend** : **Go** (API REST/GraphQL). ✅
- **Base** : PostgreSQL (ACID, précision décimale). ✅
- **Moteur financier** : module déterministe, testé unitairement (calculs purs).
- **IA** : cascade N1 (iPhone) / N2 (homelab GPU) / N3 (cloud Fable 5), via AI Router. ✅
- **Hébergement** : homelab (Docker/Coolify).
- **Données bancaires** : import CSV/OFX d'abord, puis GoCardless (DSP2, gratuit).

## 9. Modèle de données (conceptuel)

Actifs et passifs = **entités de premier rang**, avec historique de **valorisations**.

- `Profile` — utilisateur (toi, parents) + paramètres de confidentialité
- `Asset` / `Account` — comptes, livrets, placements, immobilier, objets, entreprise
- `Liability` — crédits
- `Valuation` — snapshot daté de la valeur (→ courbe patrimoine net)
- `Transaction` — montant (centimes), date, libellé, compte, catégorie, split
- `Category` / `Subcategory`, `Tag`, `Split`
- `Envelope` — budget ; `RecurringRule` — abonnements/revenus récurrents
- `Goal` — objectif de vie ; `Scenario` — simulation/décision
- `Document` — coffre-fort (avec niveau de confidentialité) ; `Beneficiary` / `Contact` — transmission
- `AIRequestLog` — trace de routage IA (niveau, motif, anonymisation appliquée)

## 10. Design & UX

Référence **Revolut** + **Liquid Glass** (iOS 26, natif SwiftUI). Identité inspirée
de l'**opale** : reflets nacrés, dégradés irisés, profondeur.

Mécaniques « satisfaisantes » : compteurs animés, haptique (Taptic Engine), cartes
translucides en couches, spring physics, célébrations (confettis), graphes scrubables,
cartes draggables, thèmes multiples, sound design optionnel. Objectif : ouvrir Opale =
micro-récompense → habitude. Détail des écrans : à produire (maquettes par palier).

## 11. Contraintes

- **Financière** : montants en centimes/DECIMAL (ENF-007) — règle absolue.
- **Confidentialité** : niveaux N1/N2/N3 strictement respectés.
- **Légale** : usage familial privé ; l'app informe mais ne fournit pas de conseil réglementé.
- **Technique** : auto-hébergement homelab ; dépendance à la dispo du homelab pour N2.
- **Budget** : projet perso ; cloud utilisé en « turbo » ponctuel pour limiter les coûts.

## 12. Données bancaires & intégrations

| ID | Exigence | Prio | Palier |
|---|---|---|---|
| EF-070 | Import manuel CSV/OFX | M | P3 |
| EF-071 | Synchro bancaire automatique via GoCardless (DSP2) | C | P7 |
| EF-072 | Saisie/maj manuelle des actifs non bancaires (immo, objets, crypto, entreprise) | M | P1 |

## 13. Livrables

- Backend Go + base PostgreSQL + moteur financier (testé).
- App iOS SwiftUI.
- App Web.
- AI Router + intégrations IA (N1/N2/N3) + moteur d'anonymisation.
- Documentation (ce cahier des charges, conception, schéma de données, README).
- Pipeline CI/CD + déploiement homelab.

## 14. Planning par paliers

| Palier | Contenu | Exigences principales |
|---|---|---|
| **P0** Fondations | Repo, backend Go + Postgres, modèle de données, auth profils, CI/CD | EF-001/002, EF-072 |
| **P1** Patrimoine net | Accueil (patrimoine net + graphe), saisie actifs/dettes, design system de base | EF-003/004/005, EF-010→014, EF-030→032 |
| **P2** Projection | Date d'indépendance, simulateur, **moteur financier déterministe** | EF-040/041, EIA-040/042 |
| **P3** Flux | Transactions, import CSV, catégorisation IA, calendrier, cashflow futur | EF-020→027 |
| **P4** Pilotage | Enveloppes, objectifs, score de santé, alertes | EF-015/016, EF-028, EF-042/043, EF-053 |
| **P5** Cerveau IA | Assistant, Mode Décision, Financial Twin, bilan, radar, **AI Router + cascade + anonymisation** | EF-050→052, EF-060→062, EIA-* |
| **P6** Profondeur | Immobilier, investissements, objets, timeline, coffre-fort, transmission | EF-033→035, EF-045, EF-063/064 |
| **P7** Confort | Synchro bancaire, comparateur de scénarios, entrepreneur, widgets, watch, partage | EF-007/008, EF-036, EF-044, EF-071 |

## 15. Critères d'acceptation (recette)

- **CA-1** Le patrimoine net affiché = somme des valorisations d'actifs − passifs (vérifiable).
- **CA-2** Aucun calcul monétaire en float ; tests unitaires du moteur passent (impact, projection, fonds d'urgence, date d'indépendance).
- **CA-3** Une donnée N1 n'est **jamais** transmise hors appareil/homelab (test de non-régression).
- **CA-4** Tout envoi cloud (N2) est anonymisé et fait l'objet d'une confirmation pour données sensibles.
- **CA-5** Le fallback IA fonctionne : iPhone → homelab → cloud, avec messages clairs.
- **CA-6** L'IA n'affiche aucun chiffre financier non issu du moteur déterministe.
- **CA-7** UI iOS fluide (pas de jank visible) sur les écrans principaux.
- **CA-8** Import CSV → transactions catégorisées automatiquement, corrigeables.

## 16. Risques & mitigations

| Risque | Impact | Mitigation |
|---|---|---|
| Périmètre trop large (9 modules) | Projet jamais fini | Construction par paliers ; cœur d'abord (P0→P2) |
| Exactitude des calculs financiers | Perte de confiance | Moteur déterministe + tests unitaires (CA-2) |
| Fuite de données sensibles vers le cloud | Critique (vie privée) | Niveaux N1/N2/N3 + anonymisation + opt-in (CA-3/4) |
| Dépendance au homelab allumé | Fonctions N2 indispo | Fallback cloud + dégradation gracieuse (ENF-008) |
| Qualité IA locale limitée (iPhone/3080) | Mauvaises réponses | Cascade : escalade vers homelab/cloud si insuffisant |
| Token/API exposés | Sécurité | Secrets hors code, en variables d'env / coffre |

## 17. Décisions

**Tranchées ✅** : Nom **Opale** ; iOS SwiftUI natif ; backend **Go** ; PostgreSQL ;
**IA hybride en cascade** (iPhone → homelab → cloud) + AI Router + moteur déterministe ;
dépôt dédié sur github.

**Ouvertes** : framework web (React/Svelte) ; identité visuelle précise (palette opale, typo).

## 18. Glossaire

- **Patrimoine net** : total des actifs − total des dettes.
- **FIRE / date d'indépendance** : moment où les revenus du patrimoine couvrent les besoins.
- **Financial Twin** : double financier simulable de l'utilisateur.
- **AI Router** : module qui choisit le niveau d'IA (local/homelab/cloud) par demande.
- **Moteur financier déterministe** : module de calcul (non-IA) source de vérité des chiffres.
- **N1/N2/N3** : niveaux de confidentialité des données (local only / cloud anonymisé / cloud interdit).

## 19. Références

Monarch Money, Copilot Money, Empower (patrimoine) ; Revolut (design/fluidité) ;
Apple Foundation Models (IA locale iOS) ; GoCardless Bank Account Data (DSP2).
