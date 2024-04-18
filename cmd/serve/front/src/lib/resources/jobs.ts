import { POLLING_INTERVAL_MS } from '$lib/config';
import fetcher, { type FetchOptions, type FetchService, type QueryResult } from '$lib/fetcher';
import type { Paginated } from '$lib/pagination';

export type Job = {
	id: string;
	resource_id: string;
	group: string;
	message_name: string;
	message_data: string;
	queued_at: string;
	not_before: string;
	error_code?: string;
	policy: number;
	retrieved: boolean;
};

export enum JobPolicy {
	PreserveOrder = 1,
	WaitForOthersResourceID = 2,
	Cancellable = 4
}

export interface JobsService {
	delete(id: string): Promise<void>;
	fetchAll(page: number, options?: FetchOptions): Promise<Paginated<Job>>;
	queryAll(page: number): QueryResult<Paginated<Job>>;
}

type Options = {
	pollingInterval: number;
};

export class RemoteJobsService implements JobsService {
	constructor(private readonly _fetcher: FetchService, private readonly _options: Options) {}

	delete(id: string): Promise<void> {
		return this._fetcher.delete(`/api/v1/jobs/${id}`, {
			invalidate: ['/api/v1/jobs']
		});
	}

	queryAll(page: number): QueryResult<Paginated<Job>> {
		return this._fetcher.query('/api/v1/jobs', {
			refreshInterval: this._options.pollingInterval,
			params: {
				page
			}
		});
	}

	fetchAll(page: number, options?: FetchOptions): Promise<Paginated<Job>> {
		return this._fetcher.get('/api/v1/jobs', {
			...options,
			params: {
				page
			}
		});
	}
}

const service: JobsService = new RemoteJobsService(fetcher, {
	pollingInterval: POLLING_INTERVAL_MS
});

export default service;
