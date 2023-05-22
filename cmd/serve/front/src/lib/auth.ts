import { goto } from '$app/navigation';
import sessions, { type SessionsService } from '$lib/resources/sessions';
import type { Profile } from '$lib/resources/users';
import type { FetchOptions } from '$lib/remote';
import routes from '$lib/path';
import cache, { type CacheService } from '$lib/cache';

type AuthOptions = {
	onLoginSucceeded?(): Promise<void>;
	onLogoutSucceeded?(redirectTo?: string): Promise<void>;
};

const anonymous: Profile = { id: '', email: '', api_key: '', registered_at: '' };

export class Auth {
	constructor(
		private readonly _sessions: SessionsService,
		private readonly _cache: CacheService,
		private readonly _options: AuthOptions
	) {}

	/**
	 * Try to log the user in.
	 */
	public async login(email: string, password: string) {
		await this._sessions.create(email, password);
		await this._cache.reset();
		await this._options.onLoginSucceeded?.();
	}

	/**
	 * Logs the user out.
	 */
	public async logout(redirectTo?: string) {
		try {
			await this._sessions.delete();
		} catch {
			/* empty */
		}

		await this._options.onLogoutSucceeded?.(redirectTo);
	}

	/**
	 * Try to get the user session and its profile to update the user store.
	 * If it fails, it will log the current user out.
	 */
	async checkSession(options?: FetchOptions): Promise<Profile> {
		try {
			const data = await this._sessions.getCurrent(options);
			return data;
		} catch {
			await this.logout(window.location.pathname);
		}
		return anonymous;
	}
}

export default new Auth(sessions, cache, {
	onLoginSucceeded: () => {
		const params = new URLSearchParams(window.location.search);
		return goto(params.get('redirectTo') ?? routes.apps);
	},
	onLogoutSucceeded: (redirectTo) => goto(routes.signin(redirectTo))
});
