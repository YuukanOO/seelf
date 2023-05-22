export function isSet<T>(data: Maybe<T>): data is T {
	return !!data;
}
