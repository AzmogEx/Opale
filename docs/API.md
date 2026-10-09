# Référence des routes HTTP

> Générée par `python3 scripts/check_docs.py --write` depuis le routeur Chi et le catalogue documentaire. Ne pas éditer la table à la main.

Backend public imposé : `https://opale.vaycode.com`. Les chemins de collection sont normalisés sans slash final dans cette table ; le routeur peut enregistrer leur variante `/`.

Les routes `/v1` sont privées avec `Authorization: Bearer <jeton>`, sauf la liste/création/démo de profils et la connexion. Les erreurs sont des réponses HTTP explicites ; un 401 invalide une session authentifiée, un 403 ou une panne réseau ne signifie pas une expiration. Les écritures à révision refusent les conflits (409). Les montants `*_cents` sont des entiers en unités mineures de la devise, jamais des nombres décimaux JSON.

`/healthz` vérifie la vie du processus ; `/readyz` vérifie sa disponibilité et PostgreSQL. `/metrics` ne demande pas de session au routeur mais reste bloqué par nginx public et réservé au réseau API privé.

Chaque ligne renvoie à la fiche fonctionnelle et au handler pour le contrat exact de paramètres, validation et réponse. Les conventions transverses sont dans [CONVENTIONS-FINANCIERES.md](CONVENTIONS-FINANCIERES.md) et [DATA-MODEL.md](DATA-MODEL.md).

| Méthode et chemin | Fonctionnalité | Handler |
|---|---|---|
| `GET /healthz` | [Architecture, démarrage, migrations et exploitation](fonctionnalites/architecture-exploitation.md) | [handleHealthz](../backend/internal/api/health.go) |
| `GET /readyz` | [Architecture, démarrage, migrations et exploitation](fonctionnalites/architecture-exploitation.md) | [handleReadyz](../backend/internal/api/health.go) |
| `GET /metrics` | [Architecture, démarrage, migrations et exploitation](fonctionnalites/architecture-exploitation.md) | [handleMetrics](../backend/internal/api/metrics.go) |
| `GET /v1/profiles` | [Connexion, profils et verrouillage](fonctionnalites/connexion-profils.md) | [handleListProfiles](../backend/internal/api/profiles.go) |
| `POST /v1/profiles` | [Connexion, profils et verrouillage](fonctionnalites/connexion-profils.md) | [handleCreateProfile](../backend/internal/api/profiles.go) |
| `POST /v1/profiles/demo` | [Connexion, profils et verrouillage](fonctionnalites/connexion-profils.md) | [handleCreateDemoProfile](../backend/internal/api/pilote.go) |
| `POST /v1/auth/login` | [Connexion, profils et verrouillage](fonctionnalites/connexion-profils.md) | [handleLogin](../backend/internal/api/profiles.go) |
| `POST /v1/auth/logout` | [Connexion, profils et verrouillage](fonctionnalites/connexion-profils.md) | [handleLogout](../backend/internal/api/profiles.go) |
| `GET /v1/me` | [Connexion, profils et verrouillage](fonctionnalites/connexion-profils.md) | [handleMe](../backend/internal/api/profiles.go) |
| `GET /v1/journey` | [Parcours manuel et situation de départ](fonctionnalites/parcours-guide.md) | [handleJourney](../backend/internal/api/journey.go) |
| `PUT /v1/journey` | [Parcours manuel et situation de départ](fonctionnalites/parcours-guide.md) | [handleSaveJourney](../backend/internal/api/journey.go) |
| `PATCH /v1/me` | [Connexion, profils et verrouillage](fonctionnalites/connexion-profils.md) | [handleUpdateMe](../backend/internal/api/crud.go) |
| `GET /v1/contracts` | [Contrats, abonnements et engagements](fonctionnalites/contrats-abonnements.md) | [handleContracts](../backend/internal/api/financial_tools.go) |
| `PUT /v1/contracts/{id}` | [Contrats, abonnements et engagements](fonctionnalites/contrats-abonnements.md) | [handleSaveContract](../backend/internal/api/financial_tools.go) |
| `DELETE /v1/contracts/{id}` | [Contrats, abonnements et engagements](fonctionnalites/contrats-abonnements.md) | [handleDeleteContract](../backend/internal/api/financial_tools.go) |
| `GET /v1/contracts/{id}/prices` | [Alertes de hausse et rappels de contrats](fonctionnalites/hausses-abonnements.md) | [handleContractPrices](../backend/internal/api/financial_tools.go) |
| `POST /v1/contracts/{id}/price-observation` | [Alertes de hausse et rappels de contrats](fonctionnalites/hausses-abonnements.md) | [handleResolveContractPrice](../backend/internal/api/financial_tools.go) |
| `GET /v1/incomes/variable` | [Salaires et revenus variables](fonctionnalites/revenus-variables.md) | [handleVariableIncomes](../backend/internal/api/financial_tools.go) |
| `PUT /v1/incomes/variable/{id}` | [Salaires et revenus variables](fonctionnalites/revenus-variables.md) | [handleSaveVariableIncome](../backend/internal/api/financial_tools.go) |
| `DELETE /v1/incomes/variable/{id}` | [Salaires et revenus variables](fonctionnalites/revenus-variables.md) | [handleDeleteVariableIncome](../backend/internal/api/financial_tools.go) |
| `GET /v1/onboarding` | [Parcours manuel et situation de départ](fonctionnalites/parcours-guide.md) | [handleGetOnboarding](../backend/internal/api/onboarding.go) |
| `PUT /v1/onboarding` | [Parcours manuel et situation de départ](fonctionnalites/parcours-guide.md) | [handleSaveOnboarding](../backend/internal/api/onboarding.go) |
| `POST /v1/onboarding/complete` | [Parcours manuel et situation de départ](fonctionnalites/parcours-guide.md) | [handleFinishOnboarding](../backend/internal/api/onboarding.go) |
| `POST /v1/onboarding/skip` | [Parcours manuel et situation de départ](fonctionnalites/parcours-guide.md) | [handleFinishOnboarding](../backend/internal/api/onboarding.go) |
| `PATCH /v1/valuations/{id}` | [Actifs, dettes, valorisations et devises](fonctionnalites/patrimoine-devises.md) | [handleUpdateValuation](../backend/internal/api/crud.go) |
| `DELETE /v1/valuations/{id}` | [Actifs, dettes, valorisations et devises](fonctionnalites/patrimoine-devises.md) | [handleDeleteValuation](../backend/internal/api/crud.go) |
| `POST /v1/categories` | [Catégories, règles et suggestions de libellés](fonctionnalites/categories-regles.md) | [handleSaveCategory](../backend/internal/api/crud.go) |
| `PATCH /v1/categories/{id}` | [Catégories, règles et suggestions de libellés](fonctionnalites/categories-regles.md) | [handleSaveCategory](../backend/internal/api/crud.go) |
| `DELETE /v1/categories/{id}` | [Catégories, règles et suggestions de libellés](fonctionnalites/categories-regles.md) | [handleDeleteCategory](../backend/internal/api/crud.go) |
| `GET /v1/rules` | [Catégories, règles et suggestions de libellés](fonctionnalites/categories-regles.md) | [handleListRules](../backend/internal/api/crud.go) |
| `POST /v1/rules` | [Catégories, règles et suggestions de libellés](fonctionnalites/categories-regles.md) | [handleSaveRule](../backend/internal/api/crud.go) |
| `PATCH /v1/rules/{id}` | [Catégories, règles et suggestions de libellés](fonctionnalites/categories-regles.md) | [handleSaveRule](../backend/internal/api/crud.go) |
| `DELETE /v1/rules/{id}` | [Catégories, règles et suggestions de libellés](fonctionnalites/categories-regles.md) | [handleDeleteRule](../backend/internal/api/crud.go) |
| `DELETE /v1/push/register` | [Alertes, notifications locales et APNs](fonctionnalites/alertes-notifications.md) | [handlePushUnregister](../backend/internal/api/crud.go) |
| `PATCH /v1/goals/{id}` | [Enveloppes budgétaires et objectifs](fonctionnalites/budget-objectifs.md) | [handleUpdateGoal](../backend/internal/api/pilotage.go) |
| `GET /v1/calendar` | [Calendrier, récurrences et prévision de trésorerie](fonctionnalites/calendrier-cashflow.md) | [handleCalendar](../backend/internal/api/delivery_calendar.go) |
| `POST /v1/calendar` | [Calendrier, récurrences et prévision de trésorerie](fonctionnalites/calendrier-cashflow.md) | [handleSaveCalendar](../backend/internal/api/delivery_calendar.go) |
| `PATCH /v1/calendar/{id}` | [Calendrier, récurrences et prévision de trésorerie](fonctionnalites/calendrier-cashflow.md) | [handleSaveCalendar](../backend/internal/api/delivery_calendar.go) |
| `DELETE /v1/calendar/{id}` | [Calendrier, récurrences et prévision de trésorerie](fonctionnalites/calendrier-cashflow.md) | [handleDeleteCalendar](../backend/internal/api/delivery_calendar.go) |
| `PUT /v1/calendar/{id}/occurrences` | [Calendrier, récurrences et prévision de trésorerie](fonctionnalites/calendrier-cashflow.md) | [handleCalendarOccurrence](../backend/internal/api/delivery_calendar.go) |
| `PUT /v1/recurring/exclusion` | [Calendrier, récurrences et prévision de trésorerie](fonctionnalites/calendrier-cashflow.md) | [handleRecurringExclusion](../backend/internal/api/delivery_calendar.go) |
| `POST /v1/decisions/compare` | [Décisions, crédit, capacité et fiscalité](fonctionnalites/decisions-credit-fiscalite.md) | [handleDecisionCompare](../backend/internal/api/delivery_decisions.go) |
| `POST /v1/transactions/{id}/label-suggestion` | [Catégories, règles et suggestions de libellés](fonctionnalites/categories-regles.md) | [handleLabelSuggestion](../backend/internal/api/label_suggestion.go) |
| `PATCH /v1/contacts/{id}` | [Contacts, bénéficiaires et accès d’urgence](fonctionnalites/transmission-urgence.md) | [handleUpdateContact](../backend/internal/api/delivery_metadata.go) |
| `PATCH /v1/documents/{id}` | [Coffre chiffré et documents](fonctionnalites/coffre-documents.md) | [handleUpdateDocument](../backend/internal/api/delivery_metadata.go) |
| `GET /v1/assets/{id}/investment` | [Placements, performance et cotations](fonctionnalites/investissements-cours.md) | [handleInvestmentDetail](../backend/internal/api/delivery_investments.go) |
| `POST /v1/assets/{id}/investment/flows` | [Placements, performance et cotations](fonctionnalites/investissements-cours.md) | [handleSaveInvestmentFlow](../backend/internal/api/delivery_investments.go) |
| `PATCH /v1/assets/{id}/investment/flows/{flowID}` | [Placements, performance et cotations](fonctionnalites/investissements-cours.md) | [handleSaveInvestmentFlow](../backend/internal/api/delivery_investments.go) |
| `DELETE /v1/assets/{id}/investment/flows/{flowID}` | [Placements, performance et cotations](fonctionnalites/investissements-cours.md) | [handleDeleteInvestmentFlow](../backend/internal/api/delivery_investments.go) |
| `PUT /v1/assets/{id}/investment/coverage` | [Placements, performance et cotations](fonctionnalites/investissements-cours.md) | [handleInvestmentCoverage](../backend/internal/api/delivery_investments.go) |
| `GET /v1/beneficiaries` | [Contacts, bénéficiaires et accès d’urgence](fonctionnalites/transmission-urgence.md) | [handleBeneficiaries](../backend/internal/api/delivery_emergency.go) |
| `PUT /v1/beneficiaries` | [Contacts, bénéficiaires et accès d’urgence](fonctionnalites/transmission-urgence.md) | [handleSaveBeneficiary](../backend/internal/api/delivery_emergency.go) |
| `DELETE /v1/beneficiaries/{id}` | [Contacts, bénéficiaires et accès d’urgence](fonctionnalites/transmission-urgence.md) | [handleDeleteBeneficiary](../backend/internal/api/delivery_emergency.go) |
| `GET /v1/emergency-grants` | [Contacts, bénéficiaires et accès d’urgence](fonctionnalites/transmission-urgence.md) | [handleEmergencyGrants](../backend/internal/api/delivery_emergency.go) |
| `POST /v1/emergency-grants` | [Contacts, bénéficiaires et accès d’urgence](fonctionnalites/transmission-urgence.md) | [handleCreateEmergencyGrant](../backend/internal/api/delivery_emergency.go) |
| `PATCH /v1/emergency-grants/{id}` | [Contacts, bénéficiaires et accès d’urgence](fonctionnalites/transmission-urgence.md) | [handleActivateEmergencyGrant](../backend/internal/api/delivery_emergency.go) |
| `DELETE /v1/emergency-grants/{id}` | [Contacts, bénéficiaires et accès d’urgence](fonctionnalites/transmission-urgence.md) | [handleActivateEmergencyGrant](../backend/internal/api/delivery_emergency.go) |
| `GET /v1/emergency/{id}` | [Contacts, bénéficiaires et accès d’urgence](fonctionnalites/transmission-urgence.md) | [handleEmergencyRead](../backend/internal/api/delivery_emergency.go) |
| `GET /v1/emergency/{id}/documents/{documentID}` | [Contacts, bénéficiaires et accès d’urgence](fonctionnalites/transmission-urgence.md) | [handleEmergencyDocument](../backend/internal/api/delivery_emergency.go) |
| `POST /v1/transfers` | [Virements, ventilation et espaces partagés](fonctionnalites/virements-partage.md) | [handleTransfer](../backend/internal/api/movements.go) |
| `DELETE /v1/transfers/{id}` | [Virements, ventilation et espaces partagés](fonctionnalites/virements-partage.md) | [handleDeleteTransfer](../backend/internal/api/movements.go) |
| `GET /v1/export` | [Export, remise à zéro et suppression du profil](fonctionnalites/export-effacement.md) | [handleExport](../backend/internal/api/export.go) |
| `GET /v1/access-log` | [Connexion, profils et verrouillage](fonctionnalites/connexion-profils.md) | [handleAccessLog](../backend/internal/api/export.go) |
| `DELETE /v1/me/data` | [Export, remise à zéro et suppression du profil](fonctionnalites/export-effacement.md) | [handleResetData](../backend/internal/api/export.go) |
| `DELETE /v1/me` | [Export, remise à zéro et suppression du profil](fonctionnalites/export-effacement.md) | [handleDeleteMe](../backend/internal/api/export.go) |
| `GET /v1/net-worth` | [Patrimoine net, trésorerie et santé financière](fonctionnalites/tableau-de-bord.md) | [handleNetWorth](../backend/internal/api/networth.go) |
| `GET /v1/net-worth/history` | [Patrimoine net, trésorerie et santé financière](fonctionnalites/tableau-de-bord.md) | [handleNetWorthHistory](../backend/internal/api/networth.go) |
| `GET /v1/projection` | [Projections, indépendance financière et chronologie](fonctionnalites/projections-scenarios.md) | [handleProjection](../backend/internal/api/projection.go) |
| `GET /v1/categories` | [Catégories, règles et suggestions de libellés](fonctionnalites/categories-regles.md) | [handleListCategories](../backend/internal/api/transactions.go) |
| `GET /v1/twin` | [Allocation, snapshots et bilans mensuel/annuel](fonctionnalites/pilotage-bilans.md) | [handleTwin](../backend/internal/api/brain.go) |
| `GET /v1/risks` | [Patrimoine net, trésorerie et santé financière](fonctionnalites/tableau-de-bord.md) | [handleRisks](../backend/internal/api/brain.go) |
| `POST /v1/decision` | [Décisions, crédit, capacité et fiscalité](fonctionnalites/decisions-credit-fiscalite.md) | [handleDecision](../backend/internal/api/brain.go) |
| `GET /v1/monthly-review` | [Allocation, snapshots et bilans mensuel/annuel](fonctionnalites/pilotage-bilans.md) | [handleMonthlyReview](../backend/internal/api/brain.go) |
| `POST /v1/assistant/ask` | [Assistant, guide intégré et fournisseurs IA](fonctionnalites/assistant-ia.md) | [handleAssistantAsk](../backend/internal/api/brain.go) |
| `GET /v1/assistant/status` | [Assistant, guide intégré et fournisseurs IA](fonctionnalites/assistant-ia.md) | [handleAssistantStatus](../backend/internal/api/brain.go) |
| `GET /v1/spaces` | [Virements, ventilation et espaces partagés](fonctionnalites/virements-partage.md) | [handleListSpaces](../backend/internal/api/partage.go) |
| `POST /v1/spaces` | [Virements, ventilation et espaces partagés](fonctionnalites/virements-partage.md) | [handleCreateSpace](../backend/internal/api/partage.go) |
| `GET /v1/spaces/{id}` | [Virements, ventilation et espaces partagés](fonctionnalites/virements-partage.md) | [handleSpaceDetail](../backend/internal/api/partage.go) |
| `POST /v1/spaces/{id}/members` | [Virements, ventilation et espaces partagés](fonctionnalites/virements-partage.md) | [handleAddSpaceMember](../backend/internal/api/partage.go) |
| `DELETE /v1/spaces/{id}/members/{profileID}` | [Virements, ventilation et espaces partagés](fonctionnalites/virements-partage.md) | [handleRemoveSpaceMember](../backend/internal/api/partage.go) |
| `GET /v1/fx` | [Actifs, dettes, valorisations et devises](fonctionnalites/patrimoine-devises.md) | [handleListFX](../backend/internal/api/partage.go) |
| `PUT /v1/fx/{currency}` | [Actifs, dettes, valorisations et devises](fonctionnalites/patrimoine-devises.md) | [handleUpsertFX](../backend/internal/api/partage.go) |
| `DELETE /v1/fx/{currency}` | [Actifs, dettes, valorisations et devises](fonctionnalites/patrimoine-devises.md) | [handleDeleteFX](../backend/internal/api/partage.go) |
| `POST /v1/quotes/refresh` | [Placements, performance et cotations](fonctionnalites/investissements-cours.md) | [handleQuotesRefresh](../backend/internal/api/pilote.go) |
| `GET /v1/snapshots` | [Allocation, snapshots et bilans mensuel/annuel](fonctionnalites/pilotage-bilans.md) | [handleSnapshots](../backend/internal/api/pilote.go) |
| `GET /v1/allocation` | [Allocation, snapshots et bilans mensuel/annuel](fonctionnalites/pilotage-bilans.md) | [handleAllocation](../backend/internal/api/pilote.go) |
| `PUT /v1/allocation` | [Allocation, snapshots et bilans mensuel/annuel](fonctionnalites/pilotage-bilans.md) | [handleSetAllocation](../backend/internal/api/pilote.go) |
| `GET /v1/alerts/custom` | [Alertes, notifications locales et APNs](fonctionnalites/alertes-notifications.md) | [handleListCustomAlerts](../backend/internal/api/pilote.go) |
| `POST /v1/alerts/custom` | [Alertes, notifications locales et APNs](fonctionnalites/alertes-notifications.md) | [handleCreateCustomAlert](../backend/internal/api/pilote.go) |
| `PATCH /v1/alerts/custom/{id}` | [Alertes, notifications locales et APNs](fonctionnalites/alertes-notifications.md) | [handleUpdateCustomAlert](../backend/internal/api/pilote.go) |
| `DELETE /v1/alerts/custom/{id}` | [Alertes, notifications locales et APNs](fonctionnalites/alertes-notifications.md) | [handleDeleteCustomAlert](../backend/internal/api/pilote.go) |
| `GET /v1/tax/estimate` | [Décisions, crédit, capacité et fiscalité](fonctionnalites/decisions-credit-fiscalite.md) | [handleTaxEstimateVerified](../backend/internal/api/delivery_tax.go) |
| `GET /v1/tax/deadlines` | [Décisions, crédit, capacité et fiscalité](fonctionnalites/decisions-credit-fiscalite.md) | [handleTaxDeadlines](../backend/internal/api/pilote.go) |
| `POST /v1/loan/simulate` | [Décisions, crédit, capacité et fiscalité](fonctionnalites/decisions-credit-fiscalite.md) | [handleLoanSimulate](../backend/internal/api/pilote.go) |
| `POST /v1/loan/capacity` | [Décisions, crédit, capacité et fiscalité](fonctionnalites/decisions-credit-fiscalite.md) | [handleLoanCapacity](../backend/internal/api/pilote.go) |
| `GET /v1/wrapped` | [Allocation, snapshots et bilans mensuel/annuel](fonctionnalites/pilotage-bilans.md) | [handleWrapped](../backend/internal/api/pilote.go) |
| `POST /v1/push/register` | [Alertes, notifications locales et APNs](fonctionnalites/alertes-notifications.md) | [handlePushRegister](../backend/internal/api/pilote.go) |
| `POST /v1/scenarios/compare` | [Projections, indépendance financière et chronologie](fonctionnalites/projections-scenarios.md) | [handleCompareScenarios](../backend/internal/api/scenarios.go) |
| `GET /v1/company` | [Sociétés, participations et compte courant d’associé](fonctionnalites/societes-cca.md) | [handleCompanies](../backend/internal/api/confort.go) |
| `GET /v1/bank/status` | [Connexion bancaire et association des comptes](fonctionnalites/synchronisation-bancaire.md) | [handleBankStatus](../backend/internal/api/confort.go) |
| `GET /v1/bank/accounts` | [Connexion bancaire et association des comptes](fonctionnalites/synchronisation-bancaire.md) | [handleBankAccounts](../backend/internal/api/delivery_bank.go) |
| `PUT /v1/bank/accounts/{id}` | [Connexion bancaire et association des comptes](fonctionnalites/synchronisation-bancaire.md) | [handleMapBankAccount](../backend/internal/api/delivery_bank.go) |
| `POST /v1/bank/links/{id}/renew` | [Connexion bancaire et association des comptes](fonctionnalites/synchronisation-bancaire.md) | [handleRenewBank](../backend/internal/api/delivery_bank.go) |
| `GET /v1/bank/institutions` | [Connexion bancaire et association des comptes](fonctionnalites/synchronisation-bancaire.md) | [handleBankInstitutions](../backend/internal/api/confort.go) |
| `POST /v1/bank/connect` | [Connexion bancaire et association des comptes](fonctionnalites/synchronisation-bancaire.md) | [handleBankConnect](../backend/internal/api/confort.go) |
| `POST /v1/bank/sync` | [Connexion bancaire et association des comptes](fonctionnalites/synchronisation-bancaire.md) | [handleBankSync](../backend/internal/api/confort.go) |
| `DELETE /v1/bank/links/{id}` | [Connexion bancaire et association des comptes](fonctionnalites/synchronisation-bancaire.md) | [handleBankDisconnect](../backend/internal/api/confort.go) |
| `GET /v1/real-estate` | [Immobilier et crédit lié](fonctionnalites/immobilier.md) | [handleRealEstate](../backend/internal/api/profondeur.go) |
| `GET /v1/investments` | [Placements, performance et cotations](fonctionnalites/investissements-cours.md) | [handleInvestments](../backend/internal/api/profondeur.go) |
| `GET /v1/objects` | [Objets de valeur et assurance](fonctionnalites/objets.md) | [handleObjects](../backend/internal/api/profondeur.go) |
| `GET /v1/timeline` | [Projections, indépendance financière et chronologie](fonctionnalites/projections-scenarios.md) | [handleTimeline](../backend/internal/api/profondeur.go) |
| `GET /v1/transmission` | [Contacts, bénéficiaires et accès d’urgence](fonctionnalites/transmission-urgence.md) | [handleTransmission](../backend/internal/api/profondeur.go) |
| `GET /v1/documents` | [Coffre chiffré et documents](fonctionnalites/coffre-documents.md) | [handleListDocuments](../backend/internal/api/profondeur.go) |
| `POST /v1/documents` | [Coffre chiffré et documents](fonctionnalites/coffre-documents.md) | [handleCreateDocument](../backend/internal/api/profondeur.go) |
| `GET /v1/documents/{id}/content` | [Coffre chiffré et documents](fonctionnalites/coffre-documents.md) | [handleDocumentContent](../backend/internal/api/profondeur.go) |
| `DELETE /v1/documents/{id}` | [Coffre chiffré et documents](fonctionnalites/coffre-documents.md) | [handleDeleteDocument](../backend/internal/api/profondeur.go) |
| `GET /v1/contacts` | [Contacts, bénéficiaires et accès d’urgence](fonctionnalites/transmission-urgence.md) | [handleListContacts](../backend/internal/api/profondeur.go) |
| `POST /v1/contacts` | [Contacts, bénéficiaires et accès d’urgence](fonctionnalites/transmission-urgence.md) | [handleCreateContact](../backend/internal/api/profondeur.go) |
| `DELETE /v1/contacts/{id}` | [Contacts, bénéficiaires et accès d’urgence](fonctionnalites/transmission-urgence.md) | [handleDeleteContact](../backend/internal/api/profondeur.go) |
| `GET /v1/recurring` | [Calendrier, récurrences et prévision de trésorerie](fonctionnalites/calendrier-cashflow.md) | [handleRecurring](../backend/internal/api/pilotage.go) |
| `GET /v1/cashflow` | [Calendrier, récurrences et prévision de trésorerie](fonctionnalites/calendrier-cashflow.md) | [handleCashflow](../backend/internal/api/pilotage.go) |
| `GET /v1/health-score` | [Patrimoine net, trésorerie et santé financière](fonctionnalites/tableau-de-bord.md) | [handleHealthScore](../backend/internal/api/pilotage.go) |
| `GET /v1/analytics` | [Patrimoine net, trésorerie et santé financière](fonctionnalites/tableau-de-bord.md) | [handleAnalytics](../backend/internal/api/pilotage.go) |
| `GET /v1/subscriptions` | [Calendrier, récurrences et prévision de trésorerie](fonctionnalites/calendrier-cashflow.md) | [handleSubscriptions](../backend/internal/api/pilotage.go) |
| `GET /v1/alerts` | [Alertes, notifications locales et APNs](fonctionnalites/alertes-notifications.md) | [handleAlerts](../backend/internal/api/pilotage.go) |
| `GET /v1/envelopes` | [Enveloppes budgétaires et objectifs](fonctionnalites/budget-objectifs.md) | [handleEnvelopeStatuses](../backend/internal/api/pilotage.go) |
| `PUT /v1/envelopes` | [Enveloppes budgétaires et objectifs](fonctionnalites/budget-objectifs.md) | [handleUpsertEnvelope](../backend/internal/api/pilotage.go) |
| `DELETE /v1/envelopes/{id}` | [Enveloppes budgétaires et objectifs](fonctionnalites/budget-objectifs.md) | [handleDeleteEnvelope](../backend/internal/api/pilotage.go) |
| `GET /v1/goals` | [Enveloppes budgétaires et objectifs](fonctionnalites/budget-objectifs.md) | [handleListGoals](../backend/internal/api/pilotage.go) |
| `POST /v1/goals` | [Enveloppes budgétaires et objectifs](fonctionnalites/budget-objectifs.md) | [handleCreateGoal](../backend/internal/api/pilotage.go) |
| `DELETE /v1/goals/{id}` | [Enveloppes budgétaires et objectifs](fonctionnalites/budget-objectifs.md) | [handleDeleteGoal](../backend/internal/api/pilotage.go) |
| `GET /v1/transactions` | [Transactions, recherche et import de relevés](fonctionnalites/transactions-imports.md) | [handleListTransactions](../backend/internal/api/transactions.go) |
| `POST /v1/transactions` | [Transactions, recherche et import de relevés](fonctionnalites/transactions-imports.md) | [handleCreateTransaction](../backend/internal/api/transactions.go) |
| `GET /v1/transactions/summary` | [Transactions, recherche et import de relevés](fonctionnalites/transactions-imports.md) | [handleMonthSummary](../backend/internal/api/transactions.go) |
| `POST /v1/transactions/import` | [Transactions, recherche et import de relevés](fonctionnalites/transactions-imports.md) | [handleImportCSV](../backend/internal/api/transactions.go) |
| `PATCH /v1/transactions/{id}` | [Transactions, recherche et import de relevés](fonctionnalites/transactions-imports.md) | [handleUpdateTransaction](../backend/internal/api/transactions.go) |
| `DELETE /v1/transactions/{id}` | [Transactions, recherche et import de relevés](fonctionnalites/transactions-imports.md) | [handleDeleteTransaction](../backend/internal/api/transactions.go) |
| `PUT /v1/transactions/{id}/space` | [Virements, ventilation et espaces partagés](fonctionnalites/virements-partage.md) | [handleSetTransactionSpace](../backend/internal/api/partage.go) |
| `POST /v1/transactions/{id}/split` | [Virements, ventilation et espaces partagés](fonctionnalites/virements-partage.md) | [handleSplitTransaction](../backend/internal/api/transactions.go) |
| `GET /v1/assets` | [Actifs, dettes, valorisations et devises](fonctionnalites/patrimoine-devises.md) | [handleListAssets](../backend/internal/api/assets.go) |
| `POST /v1/assets` | [Actifs, dettes, valorisations et devises](fonctionnalites/patrimoine-devises.md) | [handleCreateAsset](../backend/internal/api/assets.go) |
| `GET /v1/assets/{id}` | [Actifs, dettes, valorisations et devises](fonctionnalites/patrimoine-devises.md) | [handleGetAsset](../backend/internal/api/assets.go) |
| `PATCH /v1/assets/{id}` | [Actifs, dettes, valorisations et devises](fonctionnalites/patrimoine-devises.md) | [handleUpdateAsset](../backend/internal/api/assets.go) |
| `DELETE /v1/assets/{id}` | [Actifs, dettes, valorisations et devises](fonctionnalites/patrimoine-devises.md) | [handleDeleteAsset](../backend/internal/api/assets.go) |
| `GET /v1/assets/{id}/valuations` | [Actifs, dettes, valorisations et devises](fonctionnalites/patrimoine-devises.md) | [handleListAssetValuations](../backend/internal/api/assets.go) |
| `POST /v1/assets/{id}/valuations` | [Actifs, dettes, valorisations et devises](fonctionnalites/patrimoine-devises.md) | [handleAddAssetValuation](../backend/internal/api/assets.go) |
| `PUT /v1/assets/{id}/quote` | [Placements, performance et cotations](fonctionnalites/investissements-cours.md) | [handleSetAssetQuote](../backend/internal/api/pilote.go) |
| `GET /v1/assets/{id}/quote` | [Placements, performance et cotations](fonctionnalites/investissements-cours.md) | [handleQuoteMetadata](../backend/internal/api/crud.go) |
| `PUT /v1/assets/{id}/property` | [Immobilier et crédit lié](fonctionnalites/immobilier.md) | [handleUpsertProperty](../backend/internal/api/profondeur.go) |
| `PUT /v1/assets/{id}/object` | [Objets de valeur et assurance](fonctionnalites/objets.md) | [handleUpsertObject](../backend/internal/api/profondeur.go) |
| `PUT /v1/assets/{id}/company` | [Sociétés, participations et compte courant d’associé](fonctionnalites/societes-cca.md) | [handleUpsertCompany](../backend/internal/api/confort.go) |
| `GET /v1/liabilities` | [Actifs, dettes, valorisations et devises](fonctionnalites/patrimoine-devises.md) | [handleListLiabilities](../backend/internal/api/liabilities.go) |
| `POST /v1/liabilities` | [Actifs, dettes, valorisations et devises](fonctionnalites/patrimoine-devises.md) | [handleCreateLiability](../backend/internal/api/liabilities.go) |
| `GET /v1/liabilities/{id}` | [Actifs, dettes, valorisations et devises](fonctionnalites/patrimoine-devises.md) | [handleGetLiability](../backend/internal/api/liabilities.go) |
| `PATCH /v1/liabilities/{id}` | [Actifs, dettes, valorisations et devises](fonctionnalites/patrimoine-devises.md) | [handleUpdateLiability](../backend/internal/api/liabilities.go) |
| `DELETE /v1/liabilities/{id}` | [Actifs, dettes, valorisations et devises](fonctionnalites/patrimoine-devises.md) | [handleDeleteLiability](../backend/internal/api/liabilities.go) |
| `GET /v1/liabilities/{id}/valuations` | [Actifs, dettes, valorisations et devises](fonctionnalites/patrimoine-devises.md) | [handleListLiabilityValuations](../backend/internal/api/liabilities.go) |
| `POST /v1/liabilities/{id}/valuations` | [Actifs, dettes, valorisations et devises](fonctionnalites/patrimoine-devises.md) | [handleAddLiabilityValuation](../backend/internal/api/liabilities.go) |

158 routes, 30 fiches, 90 écrans/composants couverts explicitement.
