import fetcher, { type FetchService } from '$lib/fetcher';

export type Profile = {
	id: string;
	email: string;
	api_key: string;
	registered_at: string;
};

export type ByUserData = {
	id: string;
	email: string;
};

export type UpdateProfileData = {
	email?: string;
	password?: string;
};

export interface UsersService {
	update(payload: UpdateProfileData): Promise<Profile>;
	refreshAPIKey(): Promise<Pick<Profile, 'api_key'>>;
}

export class RemoteUsersService implements UsersService {
	constructor(private readonly _fetcher: FetchService) {}

	update(payload: UpdateProfileData): Promise<Profile> {
		return this._fetcher.patch('/api/v1/profile', payload);
	}

	refreshAPIKey(): Promise<Pick<Profile, 'api_key'>> {
		return this._fetcher.put('/api/v1/profile/key');
	}
}

const service: UsersService = new RemoteUsersService(fetcher);

export default service;
