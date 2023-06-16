import type { CacheFetchServiceOptions } from './cache';
import CacheData from './cache_data';

/**
 * Represents a set of cached data with base / computed mapping.
 */
export class CacheSet {
	/** Maps baseKey to computed keys for easier invalidation */
	private readonly _baseKeyMapping: Map<string, string[]> = new Map();
	private readonly _data: Map<string, CacheData> = new Map();

	public constructor(
		private readonly _options: Required<CacheFetchServiceOptions>,
		...initialData: CacheData[]
	) {
		initialData.forEach((data) => {
			this._data.set(data.key, data);
			this._baseKeyMapping.set(data.baseKey, [
				...(this._baseKeyMapping.get(data.baseKey) ?? []),
				data.key
			]);
		});
	}

	/**
	 * Retrieve or create a cache entry. The `params` argument represents an eventual
	 * query string that will be used to compute the final key.
	 */
	public getOrCreate(key: string, params?: unknown): CacheData {
		/**
		 * Compute the url with the optional query parameters.
		 * FIXME: UrlSearchParams will let undefined values in the query string so I should
		 * probably take care of that.
		 */
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		const computedKey = params ? `${key}?${new URLSearchParams(params as any)}` : key;

		let cacheData = this._data.get(computedKey);

		if (!cacheData) {
			cacheData = new CacheData(computedKey, key, this._options);
			this._data.set(computedKey, cacheData);
			this._baseKeyMapping.set(key, [...(this._baseKeyMapping.get(key) ?? []), computedKey]);
		}

		return cacheData;
	}

	/**
	 * Invalidate all the cache entries that matches the base keys and returns computed keys
	 * that have actually been invalidated.
	 */
	public invalidate(...keys: string[]): string[] {
		const computedKeys = keys.flatMap((key) => this._baseKeyMapping.get(key) ?? []);
		computedKeys.forEach((key) => this._data.get(key)?.invalidate());
		return computedKeys;
	}

	/**
	 * Reset the cache set data.
	 */
	public clear(): void {
		this._data.clear();
		this._baseKeyMapping.clear();
	}
}
