import { describe, expect, it } from 'vitest';
import CacheData from './data';
import {
	DEFAULT_NOW_FN,
	type CacheFetchServiceOptions,
	DEFAULT_INVALIDATE,
	DEFAULT_INVALIDATE_ALL
} from './cache';

const opts = {
	dedupeInterval: 3600000,
	now: DEFAULT_NOW_FN,
	invalidate: DEFAULT_INVALIDATE,
	invalidateAll: DEFAULT_INVALIDATE_ALL
} satisfies CacheFetchServiceOptions;

const updateOperation = new Promise<number>((resolve) => {
	setTimeout(() => resolve(42), 250);
});

describe('the CacheData', () => {
	it('should compute the baseKey if not given explicitly', () => {
		const cache = new CacheData(opts, '/api/v1/apps?page=2');

		expect(cache.baseKey).toBe('/api/v1/apps');
	});

	it('should expose a method to wait for the current update to finish if any', async () => {
		const cache = new CacheData(opts, '/api/v1/apps', '/api/v1/apps', 9);

		expect(await cache.wait()).toBe(9);

		cache.update(() => updateOperation);

		expect(await cache.wait()).toBe(42);
	});
});
