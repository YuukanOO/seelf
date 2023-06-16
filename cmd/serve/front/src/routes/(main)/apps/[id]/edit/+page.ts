import service from '$lib/resources/apps';
import { error } from '@sveltejs/kit';

export const load = async ({ params, fetch, depends }) => {
	try {
		// Retrieve the last version of the app because the one used in the layout load may be outdated.
		const app = await service.fetchById(params.id, { fetch, depends });

		return {
			app
		};
	} catch {
		throw error(404);
	}
};
