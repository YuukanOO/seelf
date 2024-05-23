import { POLLING_INTERVAL_MS } from '$lib/config';
import fetcher, { type FetchOptions, type FetchService, type QueryResult } from '$lib/fetcher';
import type { ByUserData } from '$lib/resources/users';

export enum TargetStatus {
	Configuring = 0,
	Failed = 1,
	Ready = 2
}

export type TargetState = {
	status: TargetStatus;
	error_code?: string;
	last_ready_version?: string;
};

export type ProviderConfigData = {
	kind: 'docker';
	data: {
		host?: string;
		user?: string;
		port?: number;
		private_key?: string;
	};
};

export type ProviderTypes = ProviderConfigData['kind'];

export type Target = {
	id: string;
	name: string;
	url: string;
	provider: ProviderConfigData;
	state: TargetState;
	cleanup_requested_at?: string;
	created_at: string;
	created_by: ByUserData;
};

export type CreateTarget = {
	name: string;
	url: string;
	docker?: {
		host?: string;
		user?: string;
		port?: number;
		private_key?: string;
	};
};

export type UpdateTarget = {
	name?: string;
	url?: string;
	docker?: {
		host?: string;
		user?: string;
		port?: number;
		private_key: Patch<string>;
	};
};

export type GetTargetsFilters = {
	active_only?: boolean;
};

export interface TargetsService {
	create(payload: CreateTarget): Promise<Target>;
	update(id: string, payload: UpdateTarget): Promise<Target>;
	reconfigure(id: string): Promise<void>;
	fetchAll(filters?: GetTargetsFilters, options?: FetchOptions): Promise<Target[]>;
	fetchById(id: string, options?: FetchOptions): Promise<Target>;
	queryAll(): QueryResult<Target[]>;
	delete(id: string): Promise<void>;
}

type Options = {
	pollingInterval: number;
};

export class RemoteTargetsService implements TargetsService {
	constructor(private readonly _fetcher: FetchService, private readonly _options: Options) {}

	create(payload: CreateTarget): Promise<Target> {
		return this._fetcher.post('/api/v1/targets', payload);
	}

	update(id: string, payload: UpdateTarget): Promise<Target> {
		return this._fetcher.patch(`/api/v1/targets/${id}`, payload, {
			invalidate: ['/api/v1/targets']
		});
	}

	delete(id: string): Promise<void> {
		return this._fetcher.delete(`/api/v1/targets/${id}`, {
			invalidate: ['/api/v1/targets']
		});
	}

	reconfigure(id: string): Promise<void> {
		return this._fetcher.post(`/api/v1/targets/${id}/reconfigure`, undefined, {
			invalidate: [`/api/v1/targets/${id}`, '/api/v1/targets']
		});
	}

	fetchAll(filters?: GetTargetsFilters, options?: FetchOptions): Promise<Target[]> {
		return this._fetcher.get('/api/v1/targets', {
			...options,
			params: filters
		});
	}

	fetchById(id: string, options?: FetchOptions): Promise<Target> {
		return this._fetcher.get(`/api/v1/targets/${id}`, options);
	}

	queryAll(): QueryResult<Target[]> {
		return this._fetcher.query('/api/v1/targets', {
			refreshInterval: this._options.pollingInterval
		});
	}
}

const service: TargetsService = new RemoteTargetsService(fetcher, {
	pollingInterval: POLLING_INTERVAL_MS
});

export default service;
