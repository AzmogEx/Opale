# Conventions financières et données

Version de livraison du 8 octobre 2026. Ces conventions décrivent les calculs implémentés et les limites de leurs données d’entrée.

## Montants et devises

Les champs historiques `*_cents` représentent des entiers d’unités monétaires **mineures natives**. `currency_exponent` donne l’échelle : EUR et USD 2, JPY 0, KWD 3. Aucun montant persisté ne passe par un flottant. Les imports refusent les décimales excédentaires ; les conversions de change utilisent PostgreSQL NUMERIC, avec arrondi au centime EUR le plus proche. Les dépassements de capacité sont des erreurs, jamais des valeurs tronquées.

Les agrégats, enveloppes, objectifs, trésorerie, projections et comparaisons générales sont en EUR. Les valorisations, transactions et détails d’un actif restent dans sa devise d’origine. Les taux valent des micro-euros pour une unité de devise. Un taux manuel appartient à son profil et comporte une date civile ; il prévaut sur une référence publique. Les références BCE conservent la date de publication du fournisseur. Les anciennes références partagées restent identifiées comme telles.

Une transaction utilise le dernier taux daté au plus tard à sa date. Un point de patrimoine utilise celui disponible à la date du point. Un total courant utilise le dernier taux connu à ce jour. Aucun taux disponible à cette date signifie une erreur explicite422, jamais une parité1:1. L’interface permet de saisir une référence historique manquante. Les actifs/dettes sans valorisation rendent le total incomplet ; `missing_valuations` le signale.

Les devises héritées à une échelle potentiellement ambiguë bloquent la migration0011. La procédure de revue et de restauration figure dans [EXPLOITATION.md](EXPLOITATION.md). Il ne faut pas renommer la devise pour contourner ce contrôle.

## Valorisation, mouvements et dates

Une valorisation représente une **clôture de journée civile**, comprenant les mouvements de cette date. Pour un compte courant ou un livret : dernier solde valorisé + mouvements comptabilisés strictement postérieurs à la valorisation, jusqu’à aujourd’hui inclus. Une transaction le jour de la valorisation n’est pas ajoutée deux fois. Les dates futures et les opérations provisoires sont exclues du solde courant. Sans solde initial, les mouvements partent de zéro, avec total signalé incomplet.

Pour les autres actifs, la valeur est la dernière valorisation connue à la date considérée. Un apport d’investissement ne devient pas automatiquement une performance de marché ; les flux du module investissement servent à analyser les valorisations observées.

Le capital restant dû est la dernière valorisation de dette moins les remboursements de principal comptabilisés et explicitement liés, selon la même frontière de clôture. Une liaison exige le même propriétaire et la même devise que le compte débité. Un remboursement supérieur au capital enregistré est refusé sans débit partiel. La suppression/recréation corrige un remboursement lié ; ses champs comptables ne sont pas modifiables indépendamment.

Les dates civiles sont des chaînes YYYY-MM-DD. Le fuseau de référence des bornes métier est Europe/Paris. Les instants de session, création et mise à jour sont distincts. Les règles mensuelles du calendrier gardent leur jour d’ancrage : une échéance au31 revient au dernier jour du mois court puis au31 du mois suivant.

Un actif ayant des transactions ne peut pas être supprimé : il faut l’archiver ou supprimer explicitement ses mouvements, en conservant les règles de suppression groupée des virements. Les corrections de valorisation d’une dette refusent un capital devenu négatif à une date passée ou future après les remboursements liés, sans modification partielle.

L’archivage ferme un actif ou une dette à sa date de clôture et l’exclut des totaux courants. Les valorisations et points antérieurs restent consultables ; il ne supprime pas l’historique. Une valeur finale nulle ou une contrepartie de cession doit être enregistrée pour représenter économiquement une vente ou un remboursement complet.

## Revenus, dépenses et transferts

`expense_income`, `interest` et `fee` alimentent revenus/dépenses. `internal_transfer`, `investment_contribution`, `investment_withdrawal` et `loan_principal` en sont exclus. Les mouvements restent présents dans le solde de leur compte. La catégorie globale historique « Virements » est exclue des flux ordinaires pour compatibilité ; les frais explicites restent des dépenses.

Le formulaire de virement crée atomiquement les deux côtés et des frais facultatifs. Même devise : les deux principaux doivent être identiques. Devises différentes : l’utilisateur renseigne les deux montants réellement débités/crédités ; aucun taux de transfert n’est inventé. `request_id` rend la création idempotente. Une suppression vise le virement entier ; la ventilation et l’édition comptable d’un seul côté sont refusées.

La balance d’un espace commun convertit chaque dépense en EUR et répartit le reste de division, centime par centime, entre membres dans un ordre stable. La somme des positions est nulle. Seules les opérations explicitement marquées communes sont partagées ; un retrait de membre détache ses opérations et révoque son accès.

## Imports et banque

Un import est atomique et borné : maximum 5 Mio de texte et 10 000 mouvements. Une ligne invalide refuse le fichier complet avec une erreur ; elle n’est pas ignorée silencieusement. CSV UTF-8 et Windows-1252, OFX SGML/XML ; le FITID OFX est conservé comme identifiant source lorsqu’il existe.

L’identité importée vit dans un registre distinct des lignes visibles. La ventilation ou la suppression d’une ligne ne supprime pas cette identité. Les imports concurrents sont sérialisés par profil. Les opérations bancaires avec identifiant stable peuvent évoluer de provisoire à comptabilisé et être corrigées sans doublon. Une correction fournisseur incompatible avec une ventilation demande une réconciliation explicite.

Sans identifiant fournisseur, l’identité est l’empreinte compte/date/montant/libellé brut et le rang de l’occurrence identique dans le fichier. Deux fichiers partiels contenant des opérations réellement identiques sont intrinsèquement ambigus ; privilégier OFX/FITID ou la banque. Une suppression conserve l’identité pour éviter une recréation involontaire au réimport. Des parts héritées sans registre, avec même date/libellé et somme égale à la ligne source, provoquent un refus de réconciliation plutôt qu’une recréation silencieuse. Un FITID fournisseur ne doit pas être inventé pour contourner ce contrôle.

Les comptes bancaires ont des associations distinctes et durables, même après déconnexion. Les soldes provisoires restent séparés. Les conventions de soldes fournisseur, les reprises et les limites de validation réelle sont détaillées dans [DELIVERY-BACKEND.md](DELIVERY-BACKEND.md).

## Projections, performance, fiscalité

La projection utilise un rendement nominal annuel réparti sur 12 mois, puis versement en fin de mois. Le taux lui-même n’est pas tronqué à des points de base mensuels entiers ; l’intérêt est ramené aux centimes à chaque pas. Le rendement réel de la projection générale est l’approximation nominal moins inflation, explicitement affichée. L’épargne saisie est constante dans l’unité affichée. Les comparateurs de décisions affichent séparément résultat nominal et résultat déflaté, avec hypothèses communes entre alternatives. Ce sont des scénarios, pas des prévisions certaines.

Le comparateur détaille les jalons à 0, 5 et 10 ans aux hypothèses centrales, en plus de l’horizon choisi. Au départ, seules les opérations initiales et les frais sont inclus, sans rendement ni mensualité écoulée ; la valeur nette de liquidation inclut les frais de revente hypothétiques. Les scénarios prudent, normal et ambitieux font varier uniquement le rendement des liquidités placées de −200, 0 et +200 points de base, borné entre −50 % et +20 %. Ce ne sont pas des probabilités. L’orientation déterministe privilégie une alternative finançable seulement si elle reste préférable aux trois rendements testés ; une conclusion instable, une égalité ou deux alternatives non finançables ne produisent aucune préférence.

L’indépendance est le premier mois où le capital atteint dépenses mensuelles×12/taux de retrait. Le calcul s’arrête dès l’atteinte et refuse les débordements. Sans rythme suffisant dans l’horizon maximal de 100 ans, aucune date certaine n’est affichée. Les objectifs utilisent leur épargne affectée ; le total des allocations ne doit pas dépasser la capacité observée.

La performance d’investissement distingue capital initial, apports, retraits, distributions et frais externes. Un portefeuille de 1 000 recevant 500 et valant 1 500 produit un gain de marché nul. La performance est cumulée et non annualisée, conditionnée à une couverture des flux confirmée. Les frais déjà contenus dans la valeur ne doivent pas être saisis comme frais externes supplémentaires.

Une valorisation de société est **la valeur de la part détenue**. Le pourcentage de détention est informatif et n’est pas appliqué une seconde fois. Le compte courant d’associé est une créance explicite, liée à un actif existant ou créé, incluse une seule fois dans le patrimoine.

Le fiscal 2026 fournit une estimation brute au barème sur revenus 2025, avec limites et sources affichées. Le plafond PER provient de l’avis fiscal saisi ; il n’est pas deviné. Les sources et exclusions figurent dans [DELIVERY-BACKEND.md](DELIVERY-BACKEND.md).

## Export

Le ZIP `opale-export/2` provient d’un instantané SQL cohérent, toutes opérations incluses sans limite d’interface. Il contient données natives, archives, règles, enveloppes, objectifs, domaines, droits détenus, métadonnées et documents déchiffrés. Les tables exportées sont explicitement autorisées ; les tables futures ne sont pas ajoutées automatiquement.

`manifest.json` contient `complete:true`, les nombres de lignes et les SHA-256. Il n’est produit qu’après vérification des documents. Le serveur termine un fichier temporaire privé avant de renvoyer HTTP 200 ; une interruption ou un coffre indisponible ne doit pas produire une réussite partielle. PIN, sessions, jetons push et secrets fournisseurs sont exclus.

L’export demandé explicitement contient les vrais montants et documents, y compris en mode discret. Le masquage d’écran n’est pas un chiffrement de l’archive. La sauvegarde d’exploitation est différente : dump PostgreSQL chiffré au niveau des documents et clé du coffre conservée séparément.

Les instantanés mensuels sont des observations datées (`recorded_at`), regroupées par mois. Ils sont pris à la première exécution disposant de valorisations complètes et ne représentent pas forcément la clôture du mois. Les observations anciennes restent celles enregistrées ; la courbe de l’accueil recalcule l’historique à partir des données corrigées.
