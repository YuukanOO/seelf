import fetcher, { type FetchOptions, type FetchService } from '$lib/fetcher';
import type { Profile } from '$lib/resources/users';

export interface SessionsService {
	create(email: string, password: string): Promise<Profile>;
	getCurrent(options?: FetchOptions): Promise<Profile>;
	delete(): Promise<void>;
}

export class RemoteSessionsService implements SessionsService {
	constructor(private readonly _fetcher: FetchService) {}

	create(email: string, password: string): Promise<Profile> {
		return this._fetcher.post('/api/v1/sessions', {
			email,
			password
		});
	}

	getCurrent(options?: FetchOptions): Promise<Profile> {
		return this._fetcher.get('/api/v1/profile', options);
	}

	delete(): Promise<void> {
		return this._fetcher.delete('/api/v1/session');
	}
}

const service: SessionsService = new RemoteSessionsService(fetcher);

export default service;
