import cache, { type CacheHit, type CacheService } from '$lib/cache';
import { LOGS_POLLING_INTERVAL_MS, POLLING_INTERVAL_MS } from '$lib/config';
import type { Paginated } from '$lib/pagination';
import remote, { type FetchOptions, type RemoteService } from '$lib/remote';
import type { ByUserData } from '$lib/resources/users';

export enum DeploymentStatus {
	Pending = 0,
	Running = 1,
	Failed = 2,
	Succeeded = 3
}

export type Kind = 'archive' | 'git' | 'raw';

export type MetaData = {
	kind: Kind;
	data: string;
};

export type Service = {
	name: string;
	image: string;
	url?: string;
};

export type StateData = {
	status: DeploymentStatus;
	error_code?: string;
	started_at?: string;
	finished_at?: string;
	services?: Service[];
};

export type Environment = 'production' | 'staging';

export type DeploymentData = {
	app_id: string;
	deployment_number: number;
	environment: Environment;
	meta: MetaData;
	state: StateData;
	requested_at: string;
	requested_by: ByUserData;
};

export type QueueDeploymentData =
	| FormData
	| ({
			environment: Environment;
	  } & ({ raw: string } | { git: { branch: string; hash?: string } }));

export type QueryDeploymentsFilters = {
	page?: number;
	environment?: Environment;
};

export interface DeploymentsService {
	queue(appid: string, data: QueueDeploymentData): Promise<DeploymentData>;
	redeploy(appid: string, number: number): Promise<DeploymentData>;
	promote(appid: string, number: number): Promise<DeploymentData>;
	queryAllByApp(id: string, filters?: QueryDeploymentsFilters): CacheHit<Paginated<DeploymentData>>;
	pollLogs(appid: string, number: number): CacheHit<string>;
	pollByAppAndNumber(appid: string, number: number): CacheHit<DeploymentData>;
	fetchByAppAndNumber(
		appid: string,
		number: number,
		options?: FetchOptions
	): Promise<DeploymentData>;
}

type Options = {
	pollingInterval: number;
	logsPollingInterval: number;
};

class RemoteDeploymentsService implements DeploymentsService {
	constructor(
		private readonly _remote: RemoteService,
		private readonly _cache: CacheService,
		private readonly _options: Options
	) {}

	fetchByAppAndNumber(
		appid: string,
		number: number,
		options?: FetchOptions
	): Promise<DeploymentData> {
		return this._cache.fetch(`/api/v1/apps/${appid}/deployments/${number}`, options);
	}

	pollLogs(appid: string, number: number): CacheHit<string> {
		return this._cache.get(`/api/v1/apps/${appid}/deployments/${number}/logs`, {
			refreshInterval: this._options.logsPollingInterval,
			cache: 'no-store' // Don't know why, but sometimes in my tests, the server responds with 304 with outdated logs...
		});
	}

	pollByAppAndNumber(appid: string, number: number): CacheHit<DeploymentData> {
		return this._cache.get(`/api/v1/apps/${appid}/deployments/${number}`, {
			refreshInterval: this._options.pollingInterval
		});
	}

	async queue(appid: string, data: QueueDeploymentData): Promise<DeploymentData> {
		const result = await this._remote.post<DeploymentData, QueueDeploymentData>(
			`/api/v1/apps/${appid}/deployments`,
			data
		);

		await this.invalidateAppRelatedCache(appid);

		return result;
	}

	async redeploy(appid: string, number: number): Promise<DeploymentData> {
		const result = await this._remote.post<DeploymentData, unknown>(
			`/api/v1/apps/${appid}/deployments/${number}/redeploy`
		);

		await this.invalidateAppRelatedCache(appid);

		return result;
	}

	async promote(appid: string, number: number): Promise<DeploymentData> {
		const result = await this._remote.post<DeploymentData, unknown>(
			`/api/v1/apps/${appid}/deployments/${number}/promote`
		);

		await this.invalidateAppRelatedCache(appid);

		return result;
	}

	queryAllByApp(
		id: string,
		filters?: QueryDeploymentsFilters
	): CacheHit<Paginated<DeploymentData>> {
		return this._cache.get(`/api/v1/apps/${id}/deployments`, {
			params: filters
		});
	}

	private async invalidateAppRelatedCache(appid: string): Promise<void> {
		await this._cache.invalidate(`/api/v1/apps/${appid}`);
		await this._cache.invalidate('/api/v1/apps');
	}
}

const service: DeploymentsService = new RemoteDeploymentsService(remote, cache, {
	pollingInterval: POLLING_INTERVAL_MS,
	logsPollingInterval: LOGS_POLLING_INTERVAL_MS
});

export default service;
