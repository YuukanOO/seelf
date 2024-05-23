import type { Readable } from 'svelte/store';
import CacheFetchService from './cache';
import { SvelteInvalidator } from './set';

export type FetchOptions = {
	/** Query parameters to add to the url. */
	params?: unknown;
	/** Browser cache strategy to use */
	cache?: RequestInit['cache'];
	/**
	 * Fetch function to use instead of the window.fetch. Usefull in prefetching
	 * from page.ts files.
	 */
	fetch?: typeof globalThis.fetch;
	/**
	 * Depends function as given by sveltekit to mark a dependency if the data as been
	 * cached. It make sure the subsequent invalidates will work as expected because when you
	 * call `.get` it may returns cached data.
	 */
	depends?: (...deps: string[]) => void;
};

export type QueryOptions = FetchOptions & {
	/** Revalidate every given milliseconds */
	refreshInterval?: number;
};

export type MutateOptions = Pick<FetchOptions, 'fetch'> & {
	/**
	 * By default a mutating verb such as POST will invalidate the data at the
	 * url but you can specify other urls to invalidate here.
	 */
	invalidate?: string[];
	/**
	 * By default the url at which the mutate is done is invalidated but you
	 * can override this behavior by setting this flag to true.
	 */
	skipUrlInvalidate?: boolean;
};

export type QueryResult<T> = {
	/** Error if any during the last retrieval */
	error: Readable<Maybe<Error>>;
	/** Is the query currently loading */
	loading: Readable<boolean>;
	/** Data or undefined is not fetched yet */
	data: Readable<Maybe<T>>;
};

/**
 * Represents the service aiming at retrieving server data. The default implementation
 * uses a local cache to avoid multiple requests for the same data. It also handles
 * the invalidation when doing a mutation on the data.
 */
export interface FetchService {
	post<TOut, TIn>(url: string, data?: TIn, options?: MutateOptions): Promise<TOut>;
	patch<TOut, TIn>(url: string, data?: TIn, options?: MutateOptions): Promise<TOut>;
	put<TOut, TIn>(url: string, data?: TIn, options?: MutateOptions): Promise<TOut>;
	delete(url: string, options?: MutateOptions): Promise<void>;
	get<TOut>(url: string, options?: FetchOptions): Promise<TOut>;
	query<TOut>(url: string, options?: QueryOptions): QueryResult<TOut>;
	reset(): Promise<void>;
}

const service: FetchService = new CacheFetchService(new SvelteInvalidator());

export default service;
