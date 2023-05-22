import remote, { type FetchOptions, type RemoteService } from '$lib/remote';

export type HealthCheckResult = {
	version: string;
	domain: string;
};

export interface HealthCheckService {
	check(options?: FetchOptions): Promise<HealthCheckResult>;
}

export class RemoteHealthCheckService implements HealthCheckService {
	constructor(private readonly _remote: RemoteService) {}

	check(options?: FetchOptions): Promise<HealthCheckResult> {
		return this._remote.get('/api/v1/healthcheck', options);
	}
}

const service: HealthCheckService = new RemoteHealthCheckService(remote);

export default service;
