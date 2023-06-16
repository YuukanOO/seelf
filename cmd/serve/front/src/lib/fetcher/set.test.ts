import { describe, expect, it } from 'vitest';
import CacheSet from './set';
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

describe('the CacheSet', () => {
	it('should be instantiable', () => {
		const set = new CacheSet(opts);

		expect(set).toBeDefined();
	});

	it('should be instantiable with initial data', () => {
		const cachedData = new CacheData(opts, '/api/v1/apps', '/api/v1/apps', 'some data');
		const otherCachedData = new CacheData(
			opts,
			'/api/v1/apps?some=other&query=params',
			'/api/v1/apps',
			'some other data'
		);

		const set = new CacheSet(opts, cachedData, otherCachedData);

		expect(set.getOrCreate('/api/v1/apps')).toEqual(cachedData);
		expect(set.getOrCreate('/api/v1/apps?some=other&query=params')).toEqual(otherCachedData);
		expect(set.getOrCreate('/api/v1/health')).toBeDefined();
	});

	it('should be able to invalidate keys', async () => {
		const cachedData = new CacheData(opts, '/api/v1/apps', '/api/v1/apps', 'some data');
		const otherCachedData = new CacheData(
			opts,
			'/api/v1/apps?some=other&query=params',
			'/api/v1/apps',
			'some other data'
		);

		const set = new CacheSet(opts, cachedData, otherCachedData);

		await set.invalidate('/api/v1/apps', '/api/v1/health');

		expect(cachedData.mustRevalidate()).toBe(true);
		expect(otherCachedData.mustRevalidate()).toBe(true);
	});
});
