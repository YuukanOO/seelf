import {
	get,
	writable,
	type Readable,
	type Writable,
	type StartStopNotifier,
	type Subscriber
} from 'svelte/store';
import remote, { computeUrl, type FetchOptions, type RemoteService } from '$lib/remote';
import { invalidate, invalidateAll } from '$app/navigation';

export type CacheHit<T> = {
	/** Error thrown by the last revalidation */
	error: Readable<Maybe<Error>>;
	/** Is a revalidation occuring now? */
	loading: Readable<boolean>;
	/** Cached data or undefined if not fetched yet */
	data: Readable<Maybe<T>>;
};

export type CacheFetchOptions = FetchOptions & {
	/** Revalidate every given milliseconds, only one poller will be active at a time */
	refreshInterval?: number;
};

export interface CacheService {
	/**
	 * Reset the cache, clearing all the data.
	 */
	reset(): Promise<void>;
	/**
	 * Invalidate the given key which will force a revalidation as soon as someone
	 * ask for it.
	 */
	invalidate(key: string, options?: Pick<FetchOptions, 'params'>): Promise<void>;
	/**
	 * Retrieve the value at the given key and revalidate as needed.
	 */
	get<T>(key: string, options?: CacheFetchOptions): CacheHit<T>;
	/**
	 * Fetch the data at the given key and returns it, revalidating in the same
	 * time if needed.
	 *
	 * This is especialy usefull when prefetching data in a `page.ts` file.
	 */
	fetch<T>(key: string, options?: FetchOptions): Promise<T>;
}

/**
 * Represents a local cached data.
 */
class CacheData {
	private readonly _data: Writable<Maybe<unknown>>;
	private readonly _error: Writable<Maybe<Error>>;
	private readonly _loading: Writable<boolean>;

	private _lastRevalidatedAt?: number;

	constructor(public readonly key: string, private readonly _options: Required<SWROptions>) {
		this._data = writable();
		this._error = writable();
		this._loading = writable(false);
	}

	/**
	 * Update the cache with the given value.
	 * If a Promise is given, it will update the loading state accordingly.
	 */
	public update(value: unknown): void;
	public update(fn: () => Promise<unknown>): void;
	public update(valueOrFn: unknown | ((value: unknown) => Promise<unknown>)): void {
		const now = this._options.now();
		this._lastRevalidatedAt = now;

		if (typeof valueOrFn !== 'function') {
			this.setData(valueOrFn);
			return;
		}

		this._loading.set(true);

		valueOrFn()
			.then((data: unknown) => {
				this.setData(data);
			})
			.catch((error: Error) => {
				this._error.set(error);
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
	public mustRevalidate(at: number): boolean {
		return !this._lastRevalidatedAt || at - this._lastRevalidatedAt > this._options.dedupeInterval;
	}

	/**
	 * Invalidate this cache so the next call to `mustRevalidate` will return true.
	 */
	public invalidate() {
		this._lastRevalidatedAt = undefined;
	}

	/**
	 * Retrieve a cache result.
	 * Subscribing to the data will call the custom listener if provided (useful for polling for example).
	 */
	public hit<T>(listener?: StartStopNotifier<T>): CacheHit<T> {
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

type RevalidateResult<T> = { data?: T; error?: Error };

const DEFAULT_DEDUPE_INTERVAL_MS = 2000;
const DEFAULT_NOW_FN = () => new Date().getTime();

export type SWROptions = {
	/** Function to determine the current time (exposed here for testing mostly) */
	now?(): number;
	/** Requests in this interval will be deduped */
	dedupeInterval?: number;
};

/**
 * Implements the Stale While Revalidate pattern in an easy way.
 */
export class SWR implements CacheService {
	private readonly _cache: Map<string, CacheData>;
	private readonly _options: Required<SWROptions>;

	public constructor(private readonly _remote: RemoteService, options?: SWROptions) {
		this._options = {
			now: DEFAULT_NOW_FN,
			dedupeInterval: DEFAULT_DEDUPE_INTERVAL_MS,
			...options
		};
		this._cache = new Map();
	}

	public reset(): Promise<void> {
		this._cache.clear();
		return invalidateAll();
	}

	public invalidate(key: string, options?: Pick<FetchOptions, 'params'>): Promise<void> {
		const cache = this.getOrCreate(key, options?.params);
		cache.invalidate();
		return invalidate(cache.key); // Also invalidate the page load dependency so it will rerun
	}

	public get<T>(key: string, options?: CacheFetchOptions): CacheHit<T> {
		const cache = this.getOrCreate(key, options?.params);

		// Revalidate upon calling it
		this.refresh(cache, options);

		let listener: StartStopNotifier<T> | undefined;

		// Some options may cause a custom listener to be created. This will be used
		// to extend what should be done once subcribing to the data.
		if (options?.refreshInterval) {
			listener = () => {
				let timer: NodeJS.Timeout;

				const deferRevalidation = () => {
					timer = setTimeout(() => {
						this.refresh(cache, options).then(deferRevalidation);
					}, options.refreshInterval);
				};

				deferRevalidation();

				return () => {
					clearTimeout(timer);
				};
			};
		}

		return cache.hit<T>(listener);
	}

	public async fetch<T>(key: string, options?: FetchOptions): Promise<T> {
		const cache = this.getOrCreate(key, options?.params);
		const freshResult = await this.refresh<T>(cache, options);

		// No revalidation has happened, return the cached value instead
		if (!freshResult) {
			// eslint-disable-next-line @typescript-eslint/no-non-null-assertion
			return get(cache.hit<T>().data)!;
		}

		const { data, error } = freshResult;

		if (error) {
			throw error;
		}

		// eslint-disable-next-line @typescript-eslint/no-non-null-assertion
		return data!;
	}

	private revalidate<T>(cache: CacheData, options?: FetchOptions): Promise<RevalidateResult<T>> {
		// Remove params from options to avoid passing them to the remote service since
		// they are already part of the key.
		// eslint-disable-next-line @typescript-eslint/no-unused-vars
		const { params, ...fetchOptions } = options ?? {};

		return new Promise<RevalidateResult<T>>((resolve) => {
			cache.update(async () => {
				try {
					const data = await this._remote.get<T>(cache.key, fetchOptions);
					resolve({ data });
					return data;
				} catch (err) {
					resolve({ error: err as Error });
					throw err;
				}
			});
		});
	}

	/**
	 * Refresh the given CacheData if stale.
	 */
	private async refresh<T>(
		data: CacheData,
		options?: FetchOptions
	): Promise<RevalidateResult<T> | undefined> {
		const now = this._options.now();

		if (data.mustRevalidate(now)) {
			return this.revalidate(data, options);
		}
	}

	/**
	 * Retrieve or create the given CacheData at the given key.
	 */
	private getOrCreate(key: string, params?: unknown): CacheData {
		const computedKey = computeUrl(key, params);
		let cacheData = this._cache.get(computedKey);

		if (!cacheData) {
			cacheData = new CacheData(computedKey, this._options);
			this._cache.set(computedKey, cacheData);
		}

		return cacheData;
	}
}

const service: CacheService = new SWR(remote);

export default service;
