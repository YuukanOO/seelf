import remote, { type FetchOptions, type RemoteService } from '$lib/remote';
import type { Profile } from '$lib/resources/users';

export interface SessionsService {
	create(email: string, password: string): Promise<Profile>;
	getCurrent(options?: FetchOptions): Promise<Profile>;
	delete(): Promise<void>;
}

export class RemoteSessionsService implements SessionsService {
	constructor(private readonly _remote: RemoteService) {}

	create(email: string, password: string): Promise<Profile> {
		return this._remote.post('/api/v1/sessions', {
			email,
			password
		});
	}

	getCurrent(options?: FetchOptions): Promise<Profile> {
		return this._remote.get('/api/v1/profile', options);
	}

	delete(): Promise<void> {
		return this._remote.delete('/api/v1/session');
	}
}

const service: SessionsService = new RemoteSessionsService(remote);

export default service;
