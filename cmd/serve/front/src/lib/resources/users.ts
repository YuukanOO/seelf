import cache, { type CacheService } from '$lib/cache';
import remote, { type RemoteService } from '$lib/remote';

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
}

export class RemoteUsersService implements UsersService {
	constructor(private readonly _remote: RemoteService, private readonly _cache: CacheService) {}

	async update(payload: UpdateProfileData): Promise<Profile> {
		const result = await this._remote.patch<Profile, UpdateProfileData>('/api/v1/profile', payload);

		this._cache.invalidate('/api/v1/profile');

		return result;
	}
}

const service: UsersService = new RemoteUsersService(remote, cache);

export default service;
