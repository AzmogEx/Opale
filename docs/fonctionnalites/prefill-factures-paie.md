# Préremplissage depuis une facture ou fiche de paie

État du code au 9 octobre 2026. [Index](../README.md) · [API](../API.md).

## Utilisation

Sur iPhone, dans un contrat ou revenu/salaire, choisir « Lire une facture/fiche de paie », sélectionner un PDF ou une image dans Fichiers, relire le texte justificatif, choisir un montant candidat et vérifier la devise/date. Appliquer remplit le formulaire ; l’enregistrement reste une action distincte.

## Données et comportement

`LocalDocumentReader` lit le texte PDF ou utilise Vision OCR sur l’appareil. Il ne crée pas de copie persistante ni d’appel réseau. Au maximum 10 Mio, cinq pages PDF et 200 000 caractères sont traités ; les PDF protégés sont refusés.

Le parseur privilégie total à payer/TTC pour une facture et net payé/après impôt pour la paie. Brut, net imposable, net social et cumuls sont exclus de la sélection salariale. Plusieurs candidats ou devise ambiguë imposent une vérification ; aucun calcul brut-vers-net n’est inventé. Seuls les champs confirmés partent ensuite vers l’API lors de la sauvegarde du formulaire.

## Limites et configuration

iOS uniquement, sans dépendance à Ollama/cloud. Une image floue, un format inhabituel ou une page au-delà des cinq premières peut manquer le montant. Le document original et son texte ne sont ni déposés au coffre ni transmis à l’IA distante. La saisie manuelle reste disponible.

## Vérification

`FinancialToolsTests` vérifie facture, net payé, ambiguïtés, devises, extraction PDF et OCR réel d’une image synthétique. Recette : un brut et un net fiscal plus grands que le net payé ne doivent pas être choisis à sa place.

## API et sources

Pas de route HTTP propre à ce module ; les opérations passent par les modules associés ou restent locales.

- [ios/Opale/Core/DocumentPrefill.swift](../../ios/Opale/Core/DocumentPrefill.swift)
- [ios/Opale/Core/LocalDocumentReader.swift](../../ios/Opale/Core/LocalDocumentReader.swift)
- [ios/Opale/Features/Documents/DocumentPrefillSheet.swift](../../ios/Opale/Features/Documents/DocumentPrefillSheet.swift)
- [ios/OpaleTests/FinancialToolsTests.swift](../../ios/OpaleTests/FinancialToolsTests.swift)
