import { POLLING_INTERVAL_MS } from '$lib/config';
import fetcher, { type FetchOptions, type FetchService, type QueryResult } from '$lib/fetcher';
import type { ByUserData } from '$lib/resources/users';

export type Registry = {
	id: string;
	name: string;
	url: string;
	credentials?: Credentials;
	created_at: string;
	created_by: ByUserData;
};

export type Credentials = {
	username: string;
	password: string;
};

export type CreateRegistry = {
	name: string;
	url: string;
	credentials?: Credentials;
};

export type UpdateRegistry = {
	name?: string;
	url?: string;
	credentials: Patch<{
		username: string;
		password?: string;
	}>;
};

export interface RegistriesService {
	create(payload: CreateRegistry): Promise<Registry>;
	update(id: string, payload: UpdateRegistry): Promise<Registry>;
	delete(id: string): Promise<void>;
	fetchAll(options?: FetchOptions): Promise<Registry[]>;
	fetchById(id: string, options?: FetchOptions): Promise<Registry>;
	queryAll(): QueryResult<Registry[]>;
}

type Options = {
	pollingInterval: number;
};

export class RemoteRegistriesService implements RegistriesService {
	constructor(private readonly _fetcher: FetchService, private readonly _options: Options) {}

	create(payload: CreateRegistry): Promise<Registry> {
		return this._fetcher.post('/api/v1/registries', payload);
	}

	update(id: string, payload: UpdateRegistry): Promise<Registry> {
		return this._fetcher.patch(`/api/v1/registries/${id}`, payload);
	}

	delete(id: string): Promise<void> {
		return this._fetcher.delete(`/api/v1/registries/${id}`, {
			invalidate: ['/api/v1/registries'],
			skipUrlInvalidate: true
		});
	}

	queryAll(): QueryResult<Registry[]> {
		return this._fetcher.query('/api/v1/registries', {
			refreshInterval: this._options.pollingInterval
		});
	}

	fetchAll(options?: FetchOptions): Promise<Registry[]> {
		return this._fetcher.get('/api/v1/registries', options);
	}

	fetchById(id: string, options?: FetchOptions): Promise<Registry> {
		return this._fetcher.get(`/api/v1/registries/${id}`, options);
	}
}

const service: RegistriesService = new RemoteRegistriesService(fetcher, {
	pollingInterval: POLLING_INTERVAL_MS
});

export default service;
