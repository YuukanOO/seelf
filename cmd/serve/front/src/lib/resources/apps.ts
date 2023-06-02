import fetcher, { type FetchOptions, type FetchService, type QueryResult } from '$lib/fetcher';
import { POLLING_INTERVAL_MS } from '$lib/config';
import type { ByUserData } from '$lib/resources/users';
import type { DeploymentData } from '$lib/resources/deployments';

export type AppData = {
	id: string;
	name: string;
	cleanup_requested_at?: string;
	created_at: string;
	created_by: ByUserData;
	environments: Record<string, Maybe<DeploymentData>>;
};

type EnvironmentPerEnvPerService = Record<string, Record<string, Record<string, string>>>;
type VCSConfig = { url: string; token?: string };

export type AppDetailData = AppData & {
	vcs?: VCSConfig;
	env?: EnvironmentPerEnvPerService;
};

export type CreateAppData = {
	name: string;
	env?: EnvironmentPerEnvPerService;
	vcs?: VCSConfig;
};

export type UpdateAppData = {
	vcs: Patch<{
		url?: string;
		token: Patch<string>;
	}>;
	env: Patch<EnvironmentPerEnvPerService>;
};

export interface AppsService {
	create(payload: CreateAppData): Promise<AppDetailData>;
	update(id: string, payload: UpdateAppData): Promise<AppDetailData>;
	delete(id: string): Promise<void>;
	fetchAll(options?: FetchOptions): Promise<AppData[]>;
	fetchById(id: string, options?: FetchOptions): Promise<AppDetailData>;
	queryAll(): QueryResult<AppData[]>;
	queryById(id: string): QueryResult<AppDetailData>;
}

type Options = {
	pollingInterval: number;
};

export class RemoteAppsService implements AppsService {
	constructor(private readonly _fetcher: FetchService, private readonly _options: Options) {}

	create(payload: CreateAppData): Promise<AppDetailData> {
		return this._fetcher.post('/api/v1/apps', payload);
	}

	update(id: string, payload: UpdateAppData): Promise<AppDetailData> {
		return this._fetcher.patch(`/api/v1/apps/${id}`, payload);
	}

	delete(id: string): Promise<void> {
		return this._fetcher.delete(`/api/v1/apps/${id}`, {
			invalidate: ['/api/v1/apps']
		});
	}

	queryAll(): QueryResult<AppData[]> {
		return this._fetcher.query('/api/v1/apps', { refreshInterval: this._options.pollingInterval });
	}

	queryById(id: string): QueryResult<AppDetailData> {
		return this._fetcher.query(`/api/v1/apps/${id}`, {
			refreshInterval: this._options.pollingInterval
		});
	}

	fetchAll(options?: FetchOptions): Promise<AppData[]> {
		return this._fetcher.get('/api/v1/apps', options);
	}

	fetchById(id: string, options?: FetchOptions): Promise<AppDetailData> {
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
): Record<string, Record<string, string>> {
	return services.reduce<Record<string, Record<string, string>>>(
		(result, service) => ({
			...result,
			[service.name]: parseEnv(service.values)
		}),
		{}
	);
}

export function fromServiceVariablesRecord(
	values?: Record<string, Record<string, string>>
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
