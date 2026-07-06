// Formatage des montants — les CENTIMES restent des entiers jusqu'à
// l'affichage (ENF-007) ; la division ne sert qu'au rendu.

const whole = new Intl.NumberFormat('fr-FR', {
	style: 'currency',
	currency: 'EUR',
	maximumFractionDigits: 0
});

const full = new Intl.NumberFormat('fr-FR', {
	style: 'currency',
	currency: 'EUR',
	minimumFractionDigits: 2
});

/** « 48 300 € » — euros entiers. */
export const euros = (cents: number) => whole.format(Math.trunc(cents / 100));

/** « −42,90 € » — au centime. */
export const eurosFull = (cents: number) => full.format(cents / 100);

/** « +2 140 € » — écart signé. */
export const signedEuros = (cents: number) => (cents >= 0 ? '+' : '') + euros(cents);

/** « 7,09 % » depuis des points de base. */
export const percent = (bps: number) =>
	`${(bps / 100).toLocaleString('fr-FR', { maximumFractionDigits: 2 })} %`;

/** « juillet 2026 » pour un Date. */
export const monthLabel = (d: Date) =>
	d.toLocaleDateString('fr-FR', { month: 'long', year: 'numeric' });

/** yyyy-MM-dd attendu par le backend. */
export const dayString = (d: Date) => d.toISOString().slice(0, 10);
