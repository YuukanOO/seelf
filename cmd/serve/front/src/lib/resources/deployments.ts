import { POLLING_INTERVAL_MS, RUNNING_DEPLOYMENT_POLLING_INTERVAL_MS } from '$lib/config';
import fetcher, { type FetchOptions, type FetchService, type QueryResult } from '$lib/fetcher';
import type { Paginated } from '$lib/pagination';
import type { TargetStatus } from '$lib/resources/targets';
import type { ByUserData } from '$lib/resources/users';

export enum DeploymentStatus {
	Pending = 0,
	Running = 1,
	Failed = 2,
	Succeeded = 3
}

export type SourceData =
	| {
			discriminator: 'git';
			data: { branch: string; hash: string };
	  }
	| {
			discriminator: 'archive' | 'raw';
			data: string;
	  };

export type SourceDataDiscriminator = SourceData['discriminator'];

export type Entrypoint = {
	name: string;
	router: string;
	port: number;
	is_custom: boolean;
	subdomain?: string;
	url?: string;
	published_port?: number;
};

export type Service = {
	name: string;
	image: string;
	entrypoints: Entrypoint[];
};

export type State = {
	status: DeploymentStatus;
	error_code?: string;
	started_at?: string;
	finished_at?: string;
};

export type StateWithServices = State & {
	services: Service[];
};

export type EnvironmentName = 'production' | 'staging';

export type TargetSummary = {
	id: string;
	name?: string;
	url?: string;
	status?: TargetStatus;
};

export type Deployment = {
	app_id: string;
	deployment_number: number;
	environment: EnvironmentName;
	target: TargetSummary;
	source: SourceData;
	state: State;
	requested_at: string;
	requested_by: ByUserData;
};

export type DeploymentDetail = Omit<Deployment, 'state'> & {
	state: StateWithServices;
};

export type QueueDeployment =
	| FormData
	| ({
			environment: EnvironmentName;
	  } & ({ raw: string } | { git: { branch: string; hash?: string } }));

export type QueryDeploymentsFilters = {
	page?: number;
	environment?: EnvironmentName;
};

export interface DeploymentsService {
	queue(appid: string, data: QueueDeployment): Promise<Deployment>;
	redeploy(appid: string, number: number): Promise<Deployment>;
	promote(appid: string, number: number): Promise<Deployment>;
	queryAllByApp(id: string, filters?: QueryDeploymentsFilters): QueryResult<Paginated<Deployment>>;
	queryLogs(appid: string, number: number, poll?: boolean): QueryResult<string>;
	queryByAppAndNumber(appid: string, number: number, poll?: boolean): QueryResult<DeploymentDetail>;
	fetchByAppAndNumber(
		appid: string,
		number: number,
		options?: FetchOptions
	): Promise<DeploymentDetail>;
}

type Options = {
	pollingInterval: number;
	runningDeploymentsPollingInterval: number;
};

class RemoteDeploymentsService implements DeploymentsService {
	constructor(private readonly _fetcher: FetchService, private readonly _options: Options) {}

	fetchByAppAndNumber(
		appid: string,
		number: number,
		options?: FetchOptions
	): Promise<DeploymentDetail> {
		return this._fetcher.get(`/api/v1/apps/${appid}/deployments/${number}`, options);
	}

	queryLogs(appid: string, number: number, poll?: boolean): QueryResult<string> {
		return this._fetcher.query(`/api/v1/apps/${appid}/deployments/${number}/logs`, {
			refreshInterval: poll ? this._options.runningDeploymentsPollingInterval : undefined,
			cache: 'no-store' // Don't know why, but sometimes in my tests, the server responds with 304 with outdated logs...
		});
	}

	queryByAppAndNumber(
		appid: string,
		number: number,
		poll?: boolean
	): QueryResult<DeploymentDetail> {
		return this._fetcher.query(`/api/v1/apps/${appid}/deployments/${number}`, {
			refreshInterval: poll ? this._options.runningDeploymentsPollingInterval : undefined
		});
	}

	queue(appid: string, data: QueueDeployment): Promise<Deployment> {
		return this._fetcher.post(`/api/v1/apps/${appid}/deployments`, data, {
			invalidate: [`/api/v1/apps/${appid}`, '/api/v1/apps']
		});
	}

	redeploy(appid: string, number: number): Promise<Deployment> {
		return this._fetcher.post(`/api/v1/apps/${appid}/deployments/${number}/redeploy`, undefined, {
			invalidate: [`/api/v1/apps/${appid}`, `/api/v1/apps/${appid}/deployments`, '/api/v1/apps']
		});
	}

	promote(appid: string, number: number): Promise<Deployment> {
		return this._fetcher.post(`/api/v1/apps/${appid}/deployments/${number}/promote`, undefined, {
			invalidate: [`/api/v1/apps/${appid}`, `/api/v1/apps/${appid}/deployments`, '/api/v1/apps']
		});
	}

	queryAllByApp(id: string, filters?: QueryDeploymentsFilters): QueryResult<Paginated<Deployment>> {
		return this._fetcher.query(`/api/v1/apps/${id}/deployments`, {
			params: filters
		});
	}
}

const service: DeploymentsService = new RemoteDeploymentsService(fetcher, {
	pollingInterval: POLLING_INTERVAL_MS,
	runningDeploymentsPollingInterval: RUNNING_DEPLOYMENT_POLLING_INTERVAL_MS
});

export default service;
