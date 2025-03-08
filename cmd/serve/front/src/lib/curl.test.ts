import { describe, expect, test } from 'vitest';
import { buildCommand } from './curl';

describe('the buildCommand function', () => {
	test('should returns an empty string if the kind is not supported', () => {
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		expect(buildCommand({ payload: { kind: 'not-supported' } } as any)).toEqual('');
	});

	test('should return a valid command for raw payload', () => {
		const cmd = buildCommand({
			appId: 'some-id',
			environment: 'production',
			apiKey: 'some-api-key',
			origin: 'http://localhost:1234',
			payload: {
				kind: 'raw',
				raw: 'some content'
			}
		});

		expect(cmd).toEqual(
			`curl -i -X POST -H "Authorization: Bearer some-api-key" -H "Content-Type: application/json" -d "{ \\"environment\\":\\"production\\", \\"raw\\":\\"some content\\"}" http://localhost:1234/api/v1/apps/some-id/deployments`
		);
	});

	test('should return a valid command for archive payload', () => {
		const cmd = buildCommand({
			appId: 'some-id',
			environment: 'production',
			apiKey: 'some-api-key',
			origin: 'http://localhost:1234',
			payload: {
				kind: 'archive',
				filename: 'multiple-ports.tar.gz'
			}
		});

		expect(cmd).toEqual(
			`curl -i -X POST -H "Authorization: Bearer some-api-key" -F environment=production -F archive=@multiple-ports.tar.gz http://localhost:1234/api/v1/apps/some-id/deployments`
		);
	});

	test('should return a valid command for git payload without hash', () => {
		const cmd = buildCommand({
			appId: 'some-id',
			environment: 'production',
			apiKey: 'some-api-key',
			origin: 'http://localhost:1234',
			payload: {
				kind: 'git',
				branch: 'main'
			}
		});

		expect(cmd).toEqual(
			`curl -i -X POST -H "Authorization: Bearer some-api-key" -H "Content-Type: application/json" -d "{ \\"environment\\":\\"production\\",\\"git\\":{ \\"branch\\": \\"main\\" } }"  http://localhost:1234/api/v1/apps/some-id/deployments`
		);
	});

	test('should return a valid command for git payload with hash', () => {
		const cmd = buildCommand({
			appId: 'some-id',
			environment: 'production',
			apiKey: 'some-api-key',
			origin: 'http://localhost:1234',
			payload: {
				kind: 'git',
				branch: 'main',
				hash: 'some-hash'
			}
		});

		expect(cmd).toEqual(
			`curl -i -X POST -H "Authorization: Bearer some-api-key" -H "Content-Type: application/json" -d "{ \\"environment\\":\\"production\\",\\"git\\":{ \\"branch\\": \\"main\\", \\"hash\\": \\"some-hash\\" } }"  http://localhost:1234/api/v1/apps/some-id/deployments`
		);
	});
});
