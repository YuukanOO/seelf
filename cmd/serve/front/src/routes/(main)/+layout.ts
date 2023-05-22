import auth from '$lib/auth';
import service from '$lib/resources/healthcheck';

export const ssr = false;

export const load = async ({ fetch }) => {
	const user = await auth.checkSession({ fetch, cache: 'no-cache' }); // Don't know why, the cache option is mandatory here
	const health = await service.check({ fetch });

	return { user, health };
};
