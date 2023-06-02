import fetcher, { type FetchOptions, type FetchService } from '$lib/fetcher';

export type HealthCheckResult = {
	version: string;
	domain: string;
};

export interface HealthCheckService {
	check(options?: FetchOptions): Promise<HealthCheckResult>;
}

export class RemoteHealthCheckService implements HealthCheckService {
	constructor(private readonly _fetcher: FetchService) {}

	check(options?: FetchOptions): Promise<HealthCheckResult> {
		return this._fetcher.get('/api/v1/healthcheck', options);
	}
}

const service: HealthCheckService = new RemoteHealthCheckService(fetcher);

export default service;
