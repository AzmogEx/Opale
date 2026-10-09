# Sociétés, participations et compte courant d’associé

État du code au 9 octobre 2026. [Index](../README.md) · [API](../API.md).

## Utilisation

Patrimoine → Entreprise : suivre une participation, compléter les informations de société, rémunérations/dividendes et un éventuel compte courant d’associé. Relier la créance CCA aux données de la société plutôt que la ressaisir comme un patrimoine indépendant sans lien.

## Données et comportement

`company_details` enrichit l’actif de participation et lie une créance CCA du même profil. Sa valeur et celle de la créance suivent des valorisations distinctes. Les flux personnels déclarés et la valeur de parts ne sont pas interchangeables ; le moteur ne confond pas trésorerie professionnelle et argent personnel disponible.

## Limites et configuration

Pas de comptabilité légale, paie automatisée, estimation certifiée d’entreprise ou déclaration fiscale professionnelle. Les champs déclaratifs ne prouvent pas une distribution effective. Vérifier les hypothèses et éviter un double comptage du CCA.

Sur le web, le formulaire attend le chargement des détails avant d’autoriser la saisie. Une réponse lente ne doit pas remplacer un CCA en cours de saisie par zéro. En cas d’échec, réessayer le chargement ; le formulaire vide n’est pas présenté comme si les données existantes étaient absentes. Les centres immobilier, objets et cotations utilisent la même protection.

## Vérification

Intégration `web/tests/integration/advanced.spec.ts` : CCA à 500 EUR, créance dédiée créée et relue. `web/tests/e2e/asset-details.spec.ts` retarde volontairement les données et vérifie qu’un CCA de 500,25 EUR reste exact à l’enregistrement. Tests serveur des relations propriétaires et de l’atomicité. Recette d’une société réelle à effectuer avec ses documents, sans transformer les tests synthétiques en certification comptable.

## API et sources

- `GET /v1/company`
- `PUT /v1/assets/{id}/company`

- [backend/internal/api/confort.go](../../backend/internal/api/confort.go)
- [ios/Opale/Features/Wealth/Centers/CompanyView.swift](../../ios/Opale/Features/Wealth/Centers/CompanyView.swift)

- [web/src/lib/components/AssetDetails.svelte](../../web/src/lib/components/AssetDetails.svelte)
