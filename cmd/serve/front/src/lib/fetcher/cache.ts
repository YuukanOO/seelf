import { get, type StartStopNotifier } from 'svelte/store';
import { invalidateAll } from '$app/navigation';
import { browser } from '$app/environment';

import CacheData from './cache_data';
import type { FetchOptions, FetchService, MutateOptions, QueryOptions, QueryResult } from './index';
import { HttpError } from '$lib/error';

export const DEFAULT_DEDUPE_INTERVAL_MS = 2000;
export const DEFAULT_NOW_FN = () => Date.now();

export type CacheFetchServiceOptions = {
	/** Function to determine the current time (exposed here for testing mostly) */
	now?(): number;
	/** Requests in this interval will be deduped */
	dedupeInterval?: number;
};

export default class CacheFetchService implements FetchService {
	private readonly _options: Required<CacheFetchServiceOptions>;
	/** Maps baseKey to computed keys for easier invalidation */
	private readonly _baseKeyMapping: Map<string, string[]> = new Map();
	private readonly _cache: Map<string, CacheData> = new Map();

	public constructor(options?: CacheFetchServiceOptions, ...initialData: CacheData[]) {
		this._options = {
			now: DEFAULT_NOW_FN,
			dedupeInterval: DEFAULT_DEDUPE_INTERVAL_MS,
			...options
		};

		// Build ups the cache based on initial data
		initialData.forEach((data) => {
			this._cache.set(data.key, data);
			this._baseKeyMapping.set(data.baseKey, [
				...(this._baseKeyMapping.get(data.baseKey) || []),
				data.key
			]);
		});
	}

	public post<TOut, TIn>(url: string, data?: TIn, options?: MutateOptions): Promise<TOut> {
		return this.mutate('POST', url, data, options);
	}

	public patch<TOut, TIn>(url: string, data?: TIn, options?: MutateOptions): Promise<TOut> {
		return this.mutate('PATCH', url, data, options);
	}

	public put<TOut, TIn>(url: string, data?: TIn, options?: MutateOptions): Promise<TOut> {
		return this.mutate('PUT', url, data, options);
	}

	public delete(url: string, options?: MutateOptions): Promise<void> {
		return this.mutate('DELETE', url, undefined, options);
	}

	public async get<TOut>(url: string, options?: FetchOptions): Promise<TOut> {
		const cache = this.getOrCreate(url, options?.params);

		if (cache.mustRevalidate()) {
			return this.revalidate(cache, options);
		}

		return get(cache.hit().data) as TOut;
	}

	public query<TOut>(url: string, options?: QueryOptions): QueryResult<TOut> {
		const cache = this.getOrCreate(url, options?.params);

		if (cache.mustRevalidate()) {
			this.revalidate(cache, options).catch(() => {}); // Already catched in the cache.update so the error field will be set
		}

		let listener: StartStopNotifier<TOut> | undefined;

		// Some options may cause a custom listener to be created. This will be used
		// to extend what should be done once subcribing to the data.
		if (options?.refreshInterval) {
			listener = () => {
				let timer: NodeJS.Timeout;

				const deferRevalidation = () => {
					timer = setTimeout(() => {
						if (cache.mustRevalidate()) {
							this.revalidate(cache, options)
								.catch(() => {})
								.then(deferRevalidation);
						} else {
							deferRevalidation();
						}
					}, options.refreshInterval);
				};

				deferRevalidation();

				return () => {
					clearTimeout(timer);
				};
			};
		}

		return cache.hit(listener);
	}

	public async reset(): Promise<void> {
		this._cache.clear();
		this._baseKeyMapping.clear();

		if (browser) {
			await invalidateAll();
		}
	}

	private async revalidate<TOut>(cache: CacheData, options?: FetchOptions): Promise<TOut> {
		return cache.update(() => api<TOut>('GET', cache.key, undefined, options)) as TOut;
	}

	private async mutate<TOut, TIn>(
		method: HttpMethod,
		url: string,
		data?: TIn,
		options?: MutateOptions
	): Promise<TOut> {
		const result = await api<TOut, TIn>(method, url, data, options);

		// Invalidate all the cache entries that matches the base key
		const keys = [url, ...(options?.invalidate ?? [])];
		const computedKeys = keys.flatMap((key) => this._baseKeyMapping.get(key) || []);
		await Promise.all(computedKeys?.map((key) => this._cache.get(key)?.invalidate()));

		return result;
	}

	private getOrCreate(key: string, params?: unknown): CacheData {
		/**
		 * Compute the url with the optional query parameters.
		 * FIXME: UrlSearchParams will let undefined values in the query string so I should
		 * probably take care of that.
		 */
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		const computedKey = params ? `${key}?${new URLSearchParams(params as any)}` : key;

		let cacheData = this._cache.get(computedKey);

		if (!cacheData) {
			cacheData = new CacheData(computedKey, key, this._options);
			this._cache.set(computedKey, cacheData);
			this._baseKeyMapping.set(key, [...(this._baseKeyMapping.get(key) || []), computedKey]);
		}

		return cacheData;
	}
}

type HttpMethod = 'POST' | 'PUT' | 'PATCH' | 'GET' | 'DELETE';

/**
 * Wrap common stuff related to API request.
 */
async function api<TOut = unknown, TIn = unknown>(
	method: HttpMethod,
	url: string,
	data?: TIn,
	options?: Omit<FetchOptions, 'params'>
): Promise<TOut> {
	const additionalHeaders: HeadersInit = {};
	const isFormData = data instanceof FormData;

	if (data && !isFormData) {
		additionalHeaders['Content-Type'] = 'application/json';
	}

	const response = await (options?.fetch ?? fetch)(url, {
		method,
		headers: {
			Accept: 'application/json',
			...additionalHeaders
		},
		cache: options?.cache,
		body: data ? (isFormData ? data : JSON.stringify(data)) : undefined
	});

	if (!response.ok) {
		throw await HttpError.from(response);
	}

	const contentType = response.headers.get('content-type');

	// No content-type header, we don't know how to parse the response so return undefined
	if (!contentType) {
		return undefined as TOut;
	}

	if (contentType.includes('text/plain')) {
		return (await response.text()) as unknown as TOut;
	}

	return await response.json();
}
