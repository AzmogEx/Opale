import test from 'node:test';
import assert from 'node:assert/strict';
import {
	monthBounds,
	dayString,
	independenceDate,
	parseAmount,
	amountInput,
	money,
	query
} from '../.test-build/domain.js';
process.env.TZ = 'Europe/Paris';
test('French month boundaries include every civil day across summer and winter time', () => {
	assert.deepEqual(monthBounds(2026, 10), { from: '2026-10-01', to: '2026-10-31' });
	assert.deepEqual(monthBounds(2026, 3), { from: '2026-03-01', to: '2026-03-31' });
	assert.deepEqual(monthBounds(2028, 2), { from: '2028-02-01', to: '2028-02-29' });
	assert.equal(dayString(new Date(2026, 0, 1)), '2026-01-01');
});
test('FIRE date crosses the actual calendar year, including sub-year horizons', () => {
	assert.equal(independenceDate(6, new Date(2026, 9, 8)).getFullYear(), 2027);
	assert.equal(independenceDate(6, new Date(2026, 9, 8)).getMonth(), 3);
	assert.equal(independenceDate(0, new Date(2026, 9, 8)).getMonth(), 9);
});
test('monetary inputs preserve decimal units exactly without float multiplication', () => {
	assert.equal(parseAmount('1 234,56'), 123456);
	assert.equal(parseAmount('-0,29'), -29);
	assert.equal(parseAmount('123', 0), 123);
	assert.equal(parseAmount('1,234', 3), 1234);
	assert.equal(parseAmount('0.000001', 6), 1);
	assert.throws(() => parseAmount('0.001'));
	assert.throws(() => parseAmount('1.1', 0));
	assert.throws(() => parseAmount('NaN'));
	assert.throws(() => parseAmount('Infinity'));
	assert.throws(() => parseAmount('90071992547409.92'));
});
test('currency display respects yen and dinar minor units', () => {
	assert.match(money(123, 'JPY'), /123/);
	assert.match(money(1234, 'KWD'), /1,234/);
	assert.match(money(12345, 'USD'), /123,45/);
	assert.equal(money(Number.NaN), 'Montant indisponible');
});
test('filters cannot alter query structure', () =>
	assert.equal(
		query({ q: 'A&B + café', offset: 0, missing: '' }),
		'?q=A%26B+%2B+caf%C3%A9&offset=0'
	));
test('maximum safe monetary integer keeps its last minor unit when editing and formatting', () => {
	assert.equal(amountInput(Number.MAX_SAFE_INTEGER, 2), '90071992547409.91');
	assert.match(money(Number.MAX_SAFE_INTEGER, 'EUR'), /,91/);
	assert.equal(amountInput(-1, 3), '-0.001');
	assert.match(money(-1, 'KWD'), /-0,001/);
});
