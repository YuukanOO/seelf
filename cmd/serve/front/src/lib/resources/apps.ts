import cache, { type CacheHit, type CacheService } from '$lib/cache';
import remote, { type FetchOptions, type RemoteService } from '$lib/remote';
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
	pollAll(): CacheHit<AppData[]>;
	pollById(id: string): CacheHit<AppDetailData>;
}

type Options = {
	pollingInterval: number;
};

export class RemoteAppsService implements AppsService {
	constructor(
		private readonly _remote: RemoteService,
		private readonly _cache: CacheService,
		private readonly _options: Options
	) {}

	async create(payload: CreateAppData): Promise<AppDetailData> {
		const data = await this._remote.post<AppDetailData, CreateAppData>('/api/v1/apps', payload);
		await this._cache.invalidate('/api/v1/apps');
		return data;
	}

	async update(id: string, payload: UpdateAppData): Promise<AppDetailData> {
		const url = `/api/v1/apps/${id}`;

		const data = await this._remote.patch<AppDetailData, UpdateAppData>(url, payload);
		await this._cache.invalidate(url);

		return data;
	}

	async delete(id: string): Promise<void> {
		await this._remote.delete(`/api/v1/apps/${id}`);
		await this._cache.invalidate('/api/v1/apps');
	}

	pollAll(): CacheHit<AppData[]> {
		return this._cache.get('/api/v1/apps', { refreshInterval: this._options.pollingInterval });
	}

	pollById(id: string): CacheHit<AppDetailData> {
		return this._cache.get(`/api/v1/apps/${id}`, {
			refreshInterval: this._options.pollingInterval
		});
	}

	fetchAll(options?: FetchOptions): Promise<AppData[]> {
		return this._cache.fetch('/api/v1/apps', options);
	}

	fetchById(id: string, options?: FetchOptions): Promise<AppDetailData> {
		return this._cache.fetch(`/api/v1/apps/${id}`, options);
	}
}

const service: AppsService = new RemoteAppsService(remote, cache, {
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
