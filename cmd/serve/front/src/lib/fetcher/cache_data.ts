import {
	writable,
	type StartStopNotifier,
	type Writable,
	type Subscriber,
	type Readable
} from 'svelte/store';

import type { QueryResult } from './index';
import type { CacheFetchServiceOptions } from './cache';

/**
 * Represents a local cached data.
 */
export default class CacheData {
	private readonly _data: Writable<Maybe<unknown>>;
	private readonly _error: Writable<Maybe<Error>>;
	private readonly _loading: Writable<boolean>;

	private _lastRevalidatedAt?: number;

	/**
	 * Builds up a new cache data.
	 * @param key The cache computed key with query params inlined.
	 * @param baseKey The cache base key without query params (to invalidate them easily).
	 * @param _options The cache options.
	 * @param initialValue The initial value of the cache.
	 */
	constructor(
		public readonly key: string,
		public readonly baseKey: string,
		private readonly _options: Required<CacheFetchServiceOptions>,
		initialValue?: unknown
	) {
		this._data = writable();
		this._error = writable();
		this._loading = writable(false);

		if (initialValue != null) {
			this.update(initialValue);
		}
	}

	/**
	 * Update the cache with the given value.
	 * If a Promise is given, it will update the loading state accordingly.
	 */
	public update(value: unknown): void;
	public update(fn: () => Promise<unknown>): Promise<unknown>;
	public update(valueOrFn: unknown | (() => Promise<unknown>)): void | Promise<unknown> {
		const now = this._options.now();
		this._lastRevalidatedAt = now;

		if (typeof valueOrFn !== 'function') {
			this.setData(valueOrFn);
			return;
		}

		this._loading.set(true);

		return valueOrFn()
			.then((data: unknown) => {
				this.setData(data);
				return data;
			})
			.catch((error: Error) => {
				this._error.set(error);
				throw error;
			})
			.finally(() => {
				this._loading.set(false);
			});
	}

	/**
	 * Sets the inner data and clears the error.
	 */
	private setData(value: unknown): void {
		this._data.set(value);
		this._error.set(undefined);
	}

	/**
	 * Checks if this cache should be revalidated.
	 */
	public mustRevalidate(at?: number): boolean {
		return (
			!this._lastRevalidatedAt ||
			(at ?? this._options.now()) - this._lastRevalidatedAt > this._options.dedupeInterval
		);
	}

	/**
	 * Invalidate this cache so the next call to `mustRevalidate` will return true.
	 */
	public invalidate(): void {
		this._lastRevalidatedAt = undefined;
	}

	/**
	 * Retrieve a cache result.
	 * Subscribing to the data will call the custom listener if provided (useful for polling for example).
	 */
	public hit<T>(listener?: StartStopNotifier<T>): QueryResult<T> {
		return {
			data: (listener
				? {
						subscribe: (run: Subscriber<unknown>, invalidate?: (value?: unknown) => void) => {
							const unsubListener = listener(run);
							const unsubStore = this._data.subscribe(run, invalidate);

							return () => {
								if (unsubListener) {
									unsubListener();
								}
								unsubStore();
							};
						}
				  }
				: this._data) as Readable<Maybe<T>>,
			error: this._error,
			loading: this._loading
		};
	}
}
