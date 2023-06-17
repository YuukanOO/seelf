import { describe, expect, it } from 'vitest';
import CacheFetchService, { DEFAULT_NOW_FN } from './cache';
import CacheData from './data';
import { get } from 'svelte/store';
import { HttpError } from '$lib/error';

const opts = {
	dedupeInterval: 3600000, // For tests, make a really long dedupe interval
	now: DEFAULT_NOW_FN
};

describe('the CacheFetchService', () => {
	it('should fetch data if not cached', async () => {
		const service = new CacheFetchService();
		const mockFetch = new MockFetch('some response');

		const data = await service.get('/api/v1/apps', { fetch: mockFetch.fetch });

		expect(mockFetch.input).toBe('/api/v1/apps');
		expect(mockFetch.init?.method).toBe('GET');
		expect(data).toBe(mockFetch.data);
	});

	it('should return cached data if fetched recently', async () => {
		const cachedData = new CacheData('/api/v1/apps', undefined, 'cached response data');
		const service = new CacheFetchService(undefined, opts, cachedData);
		const mockFetch = new MockFetch('some response');

		const data = await service.get('/api/v1/apps', { fetch: mockFetch.fetch });

		expect(mockFetch.input).toBeUndefined();
		expect(mockFetch.init).toBeUndefined();
		expect(get(cachedData.hit().data)).toBe(data);
	});

	it('should revalidate data if stale', async () => {
		const cachedData = new CacheData('/api/v1/apps');
		const service = new CacheFetchService(undefined, opts, cachedData);
		const mockFetch = new MockFetch('some response');

		const data = await service.get('/api/v1/apps', { fetch: mockFetch.fetch });

		expect(mockFetch.input).toBe('/api/v1/apps');
		expect(mockFetch.init?.method).toBe('GET');
		expect(data).toBe(mockFetch.data);
		expect(get(cachedData.hit().data)).toBe(mockFetch.data);
	});

	it('should throw the error back if the request failed', async () => {
		const service = new CacheFetchService();
		const err = new HttpError(422);
		const mockFetch = new MockFetch(err);

		await expect(service.get('/api/v1/apps', { fetch: mockFetch.fetch })).rejects.toThrow(err);
	});

	it('should set the query result error if the request failed instead of throwing it', async () => {
		const service = new CacheFetchService();
		const err = new HttpError(422);
		const mockFetch = new MockFetch(err);

		const { error } = await service.query<string>('/api/v1/apps', { fetch: mockFetch.fetch });

		await waitTill(() => {
			const e = get(error);
			expect(e).toBeInstanceOf(HttpError);
			expect((e as HttpError).status).toBe(err.status);
		}, 3000);
	});

	it('should automatically invalidate url when doing a mutating request', async () => {
		const cachedData = new CacheData('/api/v1/apps', undefined, 'some data');
		const otherCachedData = new CacheData('/api/v1/app/1', undefined, 'some other data');
		const service = new CacheFetchService(undefined, opts, cachedData, otherCachedData);
		const mockFetch = new MockFetch('some response');

		await service.post('/api/v1/apps', undefined, { fetch: mockFetch.fetch });

		expect(mockFetch.input).toBe('/api/v1/apps');
		expect(mockFetch.init?.method).toBe('POST');
		expect(cachedData.mustRevalidate(opts.dedupeInterval)).toBe(true);
		expect(otherCachedData.mustRevalidate(opts.dedupeInterval)).toBe(false);
	});

	it('should invalidate given keys when doing a mutating request', async () => {
		const cachedData = new CacheData('/api/v1/app/1', undefined, 'some data');
		const page1Data = new CacheData(
			'/api/v1/app/1/deployments?env=production&page=1',
			'/api/v1/app/1/deployments',
			'some other data'
		);
		const page2Data = new CacheData(
			'/api/v1/app/1/deployments?env=production&page=2',
			'/api/v1/app/1/deployments',
			'some other data'
		);
		const notRelatedData = new CacheData('/api/v1/health', undefined, 'some data');
		const service = new CacheFetchService(
			undefined,
			opts,
			cachedData,
			page1Data,
			page2Data,
			notRelatedData
		);
		const mockFetch = new MockFetch('some response');

		await service.put('/api/v1/app/1', undefined, {
			fetch: mockFetch.fetch,
			invalidate: ['/api/v1/app/1/deployments']
		});

		expect(mockFetch.input).toBe('/api/v1/app/1');
		expect(mockFetch.init?.method).toBe('PUT');
		expect(cachedData.mustRevalidate(opts.dedupeInterval)).toBe(true);
		expect(page1Data.mustRevalidate(opts.dedupeInterval)).toBe(true);
		expect(page2Data.mustRevalidate(opts.dedupeInterval)).toBe(true);
		expect(notRelatedData.mustRevalidate(opts.dedupeInterval)).toBe(false);
	});
});

/**
 * Wait till the fn doesn't throw an error or the timeout is reached.
 * This is used when testing the cache service to wait for the cache to be updated.
 */
function waitTill(fn: () => void, timeout: number): Promise<void> {
	return new Promise((resolve, reject) => {
		const start = Date.now();
		const interval = setInterval(() => {
			try {
				fn();
				clearInterval(interval);
				resolve();
			} catch (err) {
				if (Date.now() - start > timeout) {
					clearInterval(interval);
					reject();
				}
			}
		}, 100);
	});
}

class MockFetch {
	public input?: RequestInfo | URL;
	public init?: RequestInit;

	constructor(public readonly data: any) {
		this.fetch = this.fetch.bind(this);
	}

	fetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
		this.input = input;
		this.init = init;

		return Promise.resolve(
			this.data instanceof HttpError
				? new Response(undefined, { status: this.data.status })
				: new Response(JSON.stringify(this.data), {
						status: 200,
						headers: {
							'content-type': 'application/json'
						}
				  })
		);
	}
}
