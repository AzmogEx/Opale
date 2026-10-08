export interface Field {
	key: string;
	label: string;
	type?:
		| 'text'
		| 'email'
		| 'password'
		| 'date'
		| 'money'
		| 'percent'
		| 'number'
		| 'checkbox'
		| 'select'
		| 'textarea';
	required?: boolean;
	options?: { value: string; label: string }[];
	default?: any;
	min?: number;
	max?: number;
	step?: number;
	readonly?: boolean;
	omitEmpty?: boolean;
	help?: string;
	exponent?: number | ((values: Record<string, any>) => number);
	currency?: (values: Record<string, any>) => string;
}
export const field = (
	key: string,
	label: string,
	type: Field['type'] = 'text',
	extra: Partial<Field> = {}
): Field => ({ key, label, type, ...extra });
export const options = (items: Record<string, string>) =>
	Object.entries(items).map(([value, label]) => ({ value, label }));
