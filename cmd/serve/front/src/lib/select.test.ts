import { describe, expect, test } from 'vitest';
import select from './select';

describe('the select function', () => {
	test('should returns a default value if not found', () => {
		expect(select('foo', { default: 'default' })).toEqual('default');
	});
});
