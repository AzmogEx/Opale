/** Calendar days are local civil dates, never instants converted through UTC. */
export const dayString = (date: Date) =>
	`${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;
export const monthBounds = (year: number, month: number) => ({
	from: dayString(new Date(year, month - 1, 1)),
	to: dayString(new Date(year, month, 0))
});
export const independenceDate = (months: number, now = new Date()) =>
	new Date(now.getFullYear(), now.getMonth() + months, 1);
/** Parse user decimal input exactly into the API's integer centimes contract. */
export function parseAmount(input: string, exponent = 2): number {
	const value = input
		.trim()
		.replace(/[\s\u00a0\u202f]/g, '')
		.replace(',', '.');
	if (
		!new RegExp(`^-?\\d+(?:\\.\\d{1,${Math.max(1, exponent)}})?$`).test(value) ||
		(exponent === 0 && value.includes('.'))
	)
		throw new Error(`Saisis un montant avec au plus ${exponent} décimales.`);
	const negative = value.startsWith('-');
	const [whole, fraction = ''] = value.replace('-', '').split('.');
	const cents =
		BigInt(whole) * 10n ** BigInt(exponent) + BigInt(fraction.padEnd(exponent, '0') || '0');
	if (cents > BigInt(Number.MAX_SAFE_INTEGER))
		throw new Error('Ce montant dépasse la précision prise en charge.');
	return Number(negative ? -cents : cents);
}
export const currencyExponent = (currency = 'EUR') =>
	new Intl.NumberFormat('fr-FR', { style: 'currency', currency }).resolvedOptions()
		.maximumFractionDigits ?? 2;
export function amountInput(cents: number | null | undefined, exponent = 2): string {
	if (cents == null) return '';
	if (!Number.isSafeInteger(cents)) throw new Error('Montant hors précision.');
	const value = BigInt(cents),
		absolute = value < 0n ? -value : value,
		scale = 10n ** BigInt(exponent);
	return `${value < 0n ? '-' : ''}${absolute / scale}${exponent ? '.' + String(absolute % scale).padStart(exponent, '0') : ''}`;
}
export function money(cents: number, currency = 'EUR', full = true) {
	if (!Number.isSafeInteger(cents)) return 'Montant indisponible';
	const exponent = currencyExponent(currency),
		scale = 10n ** BigInt(exponent),
		value = BigInt(cents),
		absolute = value < 0n ? -value : value;
	const formatter = new Intl.NumberFormat('fr-FR', {
		style: 'currency',
		currency,
		minimumFractionDigits: full ? exponent : 0,
		maximumFractionDigits: full ? exponent : 0
	});
	if (!full) {
		const rounded = (absolute + scale / 2n) / scale;
		return formatter.format(value < 0n ? -rounded : rounded);
	}
	const whole = absolute / scale,
		formatted = formatter.formatToParts(value < 0n ? (whole === 0n ? -0 : -whole) : whole);
	const fraction = String(absolute % scale).padStart(exponent, '0');
	return formatted.map((part) => (part.type === 'fraction' ? fraction : part.value)).join('');
}
export function query(
	params: Record<string, string | number | boolean | null | undefined>
): string {
	const q = new URLSearchParams();
	for (const [key, value] of Object.entries(params))
		if (value !== undefined && value !== null && value !== '') q.set(key, String(value));
	return q.size ? `?${q}` : '';
}
export const messageOf = (error: unknown) =>
	error instanceof Error ? error.message : String(error);
