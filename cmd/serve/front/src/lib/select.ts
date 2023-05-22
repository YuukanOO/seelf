/** Select an option based on the provided value. */
export default function select<TValue extends string | number | symbol, TOutput>(
	value: TValue,
	options: Partial<Record<TValue, TOutput>> & { default: TOutput }
): TOutput;
export default function select<TValue extends string | number | symbol, TOutput>(
	value: TValue,
	options: Partial<Record<TValue, TOutput>> & { default?: TOutput }
): Maybe<TOutput>;
export default function select<TValue extends string | number | symbol, TOutput>(
	value: TValue,
	options: Partial<Record<TValue, TOutput>> & { default?: TOutput }
): Maybe<TOutput> {
	const output = options[value];

	if (!output) {
		return options.default;
	}

	return output;
}
