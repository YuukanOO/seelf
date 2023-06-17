import { describe, expect, it } from 'vitest';
import CacheSet from './set';
import CacheData from './data';

describe('the CacheSet', () => {
	it('should be instantiable', () => {
		const set = new CacheSet();

		expect(set).toBeDefined();
	});

	it('should be instantiable with initial data', () => {
		const cachedData = new CacheData('/api/v1/apps', undefined, 'some data');
		const otherCachedData = new CacheData(
			'/api/v1/apps?some=other&query=params',
			'/api/v1/apps',
			'some other data'
		);

		const set = new CacheSet(undefined, cachedData, otherCachedData);

		expect(set.getOrCreate('/api/v1/apps')).toEqual(cachedData);
		expect(set.getOrCreate('/api/v1/apps?some=other&query=params')).toEqual(otherCachedData);
		expect(set.getOrCreate('/api/v1/health')).toBeDefined();
	});

	it('should be able to invalidate keys', async () => {
		const cachedData = new CacheData('/api/v1/apps', undefined, 'some data');
		const otherCachedData = new CacheData(
			'/api/v1/apps?some=other&query=params',
			'/api/v1/apps',
			'some other data'
		);

		const set = new CacheSet(undefined, cachedData, otherCachedData);

		await set.invalidate('/api/v1/apps', '/api/v1/health');

		expect(cachedData.mustRevalidate(5000)).toBe(true);
		expect(otherCachedData.mustRevalidate(5000)).toBe(true);
	});
});
