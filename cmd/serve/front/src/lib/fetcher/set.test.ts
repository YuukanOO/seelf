import { describe, expect, it } from 'vitest';
import { CacheSet } from './set';
import CacheData from './cache_data';
import { DEFAULT_NOW_FN } from './cache';

const opts = {
	dedupeInterval: 3600000,
	now: DEFAULT_NOW_FN
};

describe('the CacheSet', () => {
	it('should be instantiable', () => {
		const set = new CacheSet(opts);

		expect(set).toBeDefined();
	});

	it('should be instantiable with initial data', () => {
		const cachedData = new CacheData('/api/v1/apps', '/api/v1/apps', opts, 'some data');
		const otherCachedData = new CacheData(
			'/api/v1/apps?some=other&query=params',
			'/api/v1/apps',
			opts,
			'some other data'
		);

		const set = new CacheSet(opts, cachedData, otherCachedData);

		expect(set.getOrCreate('/api/v1/apps')).toEqual(cachedData);
		expect(set.getOrCreate('/api/v1/apps?some=other&query=params')).toEqual(otherCachedData);
		expect(set.getOrCreate('/api/v1/health')).toBeDefined();
	});

	it('should be able to invalid keys', () => {
		const cachedData = new CacheData('/api/v1/apps', '/api/v1/apps', opts, 'some data');
		const otherCachedData = new CacheData(
			'/api/v1/apps?some=other&query=params',
			'/api/v1/apps',
			opts,
			'some other data'
		);

		const set = new CacheSet(opts, cachedData, otherCachedData);

		expect(set.invalidate('/api/v1/apps', '/api/v1/health')).toEqual([
			'/api/v1/apps',
			'/api/v1/apps?some=other&query=params'
		]);
	});
});
