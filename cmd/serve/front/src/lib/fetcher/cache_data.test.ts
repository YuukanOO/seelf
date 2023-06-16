import { describe, expect, it } from 'vitest';
import CacheData from './cache_data';
import { DEFAULT_NOW_FN } from './cache';

const opts = {
	dedupeInterval: 3600000,
	now: DEFAULT_NOW_FN
};

const updateOperation = new Promise<number>((resolve) => {
	setTimeout(() => resolve(42), 250);
});

describe('the CacheData', () => {
	it('should expose a method to wait for the current update to finish if any', async () => {
		const cache = new CacheData('/api/v1/apps', '/api/v1/apps', opts, 9);

		expect(await cache.wait()).toBe(9);

		cache.update(() => updateOperation);

		expect(await cache.wait()).toBe(42);
	});
});
