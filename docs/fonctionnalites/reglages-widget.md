# Personnalisation, accessibilité et widget iPhone

État du code au 9 octobre 2026. [Index](../README.md) · [API](../API.md).

## Utilisation

Réglages : thème système/clair/sombre, accent, confidentialité, mode discret et options de confort présentes. Le mode discret est accessible rapidement pour masquer chiffres et textes financiers. Ajouter le widget Opale depuis l’écran d’accueil iOS ; toucher ouvre l’app, l’action de discrétion reste disponible.

## Données et comportement

Les préférences financières sont isolées par profil ; l’apparence iOS comporte aussi des préférences appareil. Le mode discret remplace les contenus sensibles dans l’arbre accessible, au lieu d’appliquer seulement un flou visuel. Les textes longs, tailles Dynamic Type et réduction du mouvement sont pris en compte dans les composants.

Le widget lit un instantané écrit par l’app dans l’App Group : aucun réseau ni jeton n’y est stocké. Changement de session/déconnexion efface cet instantané. Les graphiques et animations ne deviennent pas la source des calculs exacts.

## Limites et configuration

App Group/signature compatibles nécessaires sur iPhone. Le widget peut afficher une observation ancienne et ne synchronise pas seul les comptes. Accessibilité améliorée et tests ciblés ne constituent pas une certification VoiceOver/WCAG exhaustive ni une mesure Instruments à 120 fps.

## Vérification

Tests navigateur `privacy.spec.ts` : retrait DOM/texte privé, viewport 390×844, mouvement réduit. `FinancialPresentationTests` et UI XXXL vérifient masquage et présentation ; tests physiques VoiceOver, widget et préférences à compléter selon [livraison iOS](../DELIVERY-IOS.md).

## API et sources

Pas de route HTTP propre à ce module ; les opérations passent par les modules associés ou restent locales.

- [ios/Opale/App/OpaleApp.swift](../../ios/Opale/App/OpaleApp.swift)
- [ios/Opale/Core/WidgetBridge.swift](../../ios/Opale/Core/WidgetBridge.swift)
- [ios/Opale/Features/Settings/SettingsView.swift](../../ios/Opale/Features/Settings/SettingsView.swift)
- [ios/OpaleWidget/OpaleWidget.swift](../../ios/OpaleWidget/OpaleWidget.swift)
- [web/src/lib/components/Amount.svelte](../../web/src/lib/components/Amount.svelte)
- [web/src/lib/components/LineChart.svelte](../../web/src/lib/components/LineChart.svelte)
- [web/src/lib/components/Settings.svelte](../../web/src/lib/components/Settings.svelte)
