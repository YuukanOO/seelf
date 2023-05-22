import { HttpError } from '$lib/error';

/**
 * Additional fetch options when dealing with remote data.
 */
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
};

/**
 * Service used to communicate with the outside world.
 */
export interface RemoteService {
	post<TOut, TIn>(url: string, data?: TIn): Promise<TOut>;
	patch<TOut, TIn>(url: string, data?: TIn): Promise<TOut>;
	put<TOut, TIn>(url: string, data?: TIn): Promise<TOut>;
	delete(url: string): Promise<void>;
	get<TOut>(url: string, options?: FetchOptions): Promise<TOut>;
}

export class HttpService implements RemoteService {
	post<TOut, TIn>(url: string, data?: TIn): Promise<TOut> {
		return api('POST', url, data);
	}

	patch<TOut, TIn>(url: string, data?: TIn): Promise<TOut> {
		return api('PATCH', url, data);
	}

	put<TOut, TIn>(url: string, data?: TIn): Promise<TOut> {
		return api('PUT', url, data);
	}

	delete(url: string): Promise<void> {
		return api('DELETE', url);
	}

	get<TOut>(url: string, options?: FetchOptions): Promise<TOut> {
		return api('GET', url, undefined, options);
	}
}

const service: RemoteService = new HttpService();

export default service;

type HttpMethod = 'POST' | 'PUT' | 'PATCH' | 'GET' | 'DELETE';

/**
 * Wrap common stuff related to API request.
 */
async function api<TOut = unknown, TIn = unknown>(
	method: HttpMethod,
	url: string,
	data?: TIn,
	options?: FetchOptions
): Promise<TOut> {
	const additionalHeaders: HeadersInit = {};
	const isFormData = data instanceof FormData;
	const computedUrl = computeUrl(url, options?.params);

	if (data && !isFormData) {
		additionalHeaders['Content-Type'] = 'application/json';
	}

	const response = await (options?.fetch ?? fetch)(computedUrl, {
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

	if (!contentType) {
		return undefined as TOut;
	}

	if (contentType.includes('text/plain')) {
		return (await response.text()) as unknown as TOut;
	}

	return await response.json();
}

/**
 * Compute the url with the optional query parameters.
 * FIXME: UrlSearchParams will let undefined values in the query string so I should
 * probably take care of that.
 */
export const computeUrl = (url: string, params?: unknown) =>
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	params ? `${url}?${new URLSearchParams(params as any)}` : url;
