import type { StartStopNotifier } from 'svelte/store';

import { HttpError } from '$lib/error';
import type { FetchOptions, FetchService, MutateOptions, QueryOptions, QueryResult } from './index';
import CacheSet, { type Invalidator } from './set';
import type CacheData from './data';

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
	private readonly _cache: CacheSet;

	public constructor(
		invalidator?: Invalidator,
		options?: CacheFetchServiceOptions,
		...initialData: CacheData[]
	) {
		this._options = {
			now: DEFAULT_NOW_FN,
			dedupeInterval: DEFAULT_DEDUPE_INTERVAL_MS,
			...options
		};

		this._cache = new CacheSet(invalidator, ...initialData);
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
		const cache = this._cache.getOrCreate(url, options?.params);
		const now = this._options.now();

		if (options?.cache === 'no-store' || cache.mustRevalidate(this._options.dedupeInterval, now)) {
			return this.revalidate(cache, options, now);
		}

		// Still mark the dependency with the given key to make sure it will be invalidated correctly
		options?.depends?.(cache.key);

		return cache.wait();
	}

	public query<TOut>(url: string, options?: QueryOptions): QueryResult<TOut> {
		const cache = this._cache.getOrCreate(url, options?.params);

		this.tryRevalidate(cache, options, this._options.now());

		let listener: StartStopNotifier<TOut> | undefined;

		// Some options may cause a custom listener to be created. This will be used
		// to extend what should be done once subcribing to the data.
		if (options?.refreshInterval) {
			listener = () => {
				let timer: NodeJS.Timeout;
				let mustStop = false; // Sets to true to handle the case when the fetch has been started and the timeout cleared but the promise is still pending

				const deferRevalidation = () => {
					if (mustStop) {
						return;
					}

					timer = setTimeout(() => {
						this.tryRevalidate(cache, options, this._options.now()).then(deferRevalidation);
					}, options.refreshInterval);
				};

				deferRevalidation();

				return () => {
					clearTimeout(timer);
					mustStop = true;
				};
			};
		}

		return cache.hit(listener);
	}

	public async reset(): Promise<void> {
		this._cache.clear();
	}

	private async revalidate<TOut>(
		cache: CacheData,
		options?: FetchOptions,
		at?: number
	): Promise<TOut> {
		return cache.update(() => api<TOut>('GET', cache.key, undefined, options), at) as TOut;
	}

	private async tryRevalidate(
		cache: CacheData,
		options?: FetchOptions,
		at?: number
	): Promise<void> {
		if (options?.cache !== 'no-store' && !cache.mustRevalidate(this._options.dedupeInterval, at)) {
			return;
		}

		// eslint-disable-next-line @typescript-eslint/no-empty-function
		await this.revalidate(cache, options, at).catch(() => {});
	}

	private async mutate<TOut, TIn>(
		method: HttpMethod,
		url: string,
		body?: TIn,
		options?: MutateOptions
	): Promise<TOut> {
		const result = await api<TOut, TIn>(method, url, body, options);

		// Invalidate all the cache entries that matches the base key
		const keys = [url, ...(options?.invalidate ?? [])];
		await this._cache.invalidate(...keys);

		return result;
	}
}

type HttpMethod = 'POST' | 'PUT' | 'PATCH' | 'GET' | 'DELETE';

/**
 * Wrap common stuff related to API request.
 */
async function api<TOut = unknown, TIn = unknown>(
	method: HttpMethod,
	url: string,
	body?: TIn,
	options?: Omit<FetchOptions, 'params' | 'depends'>
): Promise<TOut> {
	const additionalHeaders: HeadersInit = {};
	const isFormData = body instanceof FormData;

	if (body && !isFormData) {
		additionalHeaders['Content-Type'] = 'application/json';
	}

	const response = await (options?.fetch ?? fetch)(url, {
		method,
		headers: {
			Accept: 'application/json',
			...additionalHeaders
		},
		cache: options?.cache,
		body: body ? (isFormData ? body : JSON.stringify(body)) : undefined
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
