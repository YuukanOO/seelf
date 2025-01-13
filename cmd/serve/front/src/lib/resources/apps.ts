import { POLLING_INTERVAL_MS } from '$lib/config';
import fetcher, { type FetchOptions, type FetchService, type QueryResult } from '$lib/fetcher';
import type { Deployment, DeploymentDetail } from '$lib/resources/deployments';
import type { ByUserData } from '$lib/resources/users';

export type App = {
	id: string;
	name: string;
	cleanup_requested_at?: string;
	created_at: string;
	created_by: ByUserData;
	latest_deployments: LatestDeployments<Deployment>;
	production_target: TargetSummary;
	staging_target: TargetSummary;
};

export type LatestDeployments<T> = {
	production?: T;
	staging?: T;
};

export type EnvironmentVariablesPerService = Record<string, Record<string, string>>;
export type VersionControl = { url: string; token?: string };

export type TargetSummary = {
	id: string;
	name: string;
	url?: string;
};

export type AppDetail = {
	id: string;
	name: string;
	cleanup_requested_at?: string;
	created_at: string;
	created_by: ByUserData;
	latest_deployments: LatestDeployments<DeploymentDetail>;
	version_control?: VersionControl;
	production: Environment;
	staging: Environment;
};

export type Environment = {
	target: TargetSummary;
	migration?: TargetSummary;
	vars?: EnvironmentVariablesPerService;
};

export type CreateAppDataEnvironmentConfig = {
	target: string;
	vars?: EnvironmentVariablesPerService;
};

export type CreateApp = {
	name: string;
	version_control?: VersionControl;
	production: CreateAppDataEnvironmentConfig;
	staging: CreateAppDataEnvironmentConfig;
};

export type UpdateApp = {
	version_control: Patch<{
		url: string;
		token: Patch<string>;
	}>;
	production: Maybe<CreateAppDataEnvironmentConfig>;
	staging: Maybe<CreateAppDataEnvironmentConfig>;
};

export interface AppsService {
	create(payload: CreateApp): Promise<AppDetail>;
	update(id: string, payload: UpdateApp): Promise<AppDetail>;
	delete(id: string): Promise<void>;
	fetchAll(options?: FetchOptions): Promise<App[]>;
	fetchById(id: string, options?: FetchOptions): Promise<AppDetail>;
	queryAll(): QueryResult<App[]>;
	queryById(id: string): QueryResult<AppDetail>;
}

type Options = {
	pollingInterval: number;
};

export class RemoteAppsService implements AppsService {
	constructor(private readonly _fetcher: FetchService, private readonly _options: Options) {}

	create(payload: CreateApp): Promise<AppDetail> {
		return this._fetcher.post('/api/v1/apps', payload);
	}

	update(id: string, payload: UpdateApp): Promise<AppDetail> {
		return this._fetcher.patch(`/api/v1/apps/${id}`, payload);
	}

	delete(id: string): Promise<void> {
		return this._fetcher.delete(`/api/v1/apps/${id}`, {
			invalidate: ['/api/v1/apps']
		});
	}

	queryAll(): QueryResult<App[]> {
		return this._fetcher.query('/api/v1/apps', { refreshInterval: this._options.pollingInterval });
	}

	queryById(id: string): QueryResult<AppDetail> {
		return this._fetcher.query(`/api/v1/apps/${id}`, {
			refreshInterval: this._options.pollingInterval
		});
	}

	fetchAll(options?: FetchOptions): Promise<App[]> {
		return this._fetcher.get('/api/v1/apps', options);
	}

	fetchById(id: string, options?: FetchOptions): Promise<AppDetail> {
		return this._fetcher.get(`/api/v1/apps/${id}`, options);
	}
}

const service: AppsService = new RemoteAppsService(fetcher, {
	pollingInterval: POLLING_INTERVAL_MS
});

export default service;

export type ServiceVariables = {
	name: string;
	values: string;
};

const regexEnvKeyValue = /^\s*(?<key>[^=]+)\s*=\s*(?<value>.*)\s*$/gm;

/**
 * Transform a list of service variables into a record.
 */
export function toServiceVariablesRecord(
	services: ServiceVariables[]
): EnvironmentVariablesPerService {
	return services.reduce<EnvironmentVariablesPerService>(
		(result, service) => ({
			...result,
			[service.name]: parseEnv(service.values)
		}),
		{}
	);
}

export function fromServiceVariablesRecord(
	values?: EnvironmentVariablesPerService
): ServiceVariables[] {
	return Object.entries(values ?? {}).map(([name, values]) => ({
		name,
		values: Object.entries(values)
			.map(([key, value]) => `${key}=${value}`)
			.join('\n')
	}));
}

function parseEnv(str: string): Record<string, string> {
	const result: Record<string, string> = {};
	let match: RegExpExecArray | null;

	while ((match = regexEnvKeyValue.exec(str)) !== null) {
		// eslint-disable-next-line @typescript-eslint/no-non-null-assertion
		const { key, value } = match.groups!;

		result[key] = value;
	}

	return result;
}
