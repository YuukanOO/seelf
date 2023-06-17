import { describe, expect, it } from 'vitest';
import CacheData from './data';

const updateOperation = new Promise<number>((resolve) => {
	setTimeout(() => resolve(42), 250);
});

describe('the CacheData', () => {
	it('should compute the baseKey if not given explicitly', () => {
		const cache = new CacheData('/api/v1/apps?page=2');

		expect(cache.baseKey).toBe('/api/v1/apps');
	});

	it('should provide a method to check for invalidation', () => {
		const cache = new CacheData('/api/v1/apps', undefined, 9);

		expect(cache.mustRevalidate(5000, Date.now())).toBe(false);

		cache.update(42);

		expect(cache.mustRevalidate(-10, Date.now())).toBe(true);
	});

	it('should expose a method to wait for the current update to finish if any', async () => {
		const cache = new CacheData('/api/v1/apps', undefined, 9);

		expect(await cache.wait()).toBe(9);

		cache.update(() => updateOperation);

		expect(await cache.wait()).toBe(42);
	});

	it('should provide a method to update the cache data', async () => {
		const cache = new CacheData('/api/v1/apps');

		expect(cache.mustRevalidate(5000)).toBe(true);

		cache.update(42);

		expect(cache.mustRevalidate(5000)).toBe(false);
		expect(await cache.wait()).toBe(42);
	});
});
