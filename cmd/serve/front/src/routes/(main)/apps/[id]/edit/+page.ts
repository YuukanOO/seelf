import service from '$lib/resources/apps';
import { error } from '@sveltejs/kit';

export const load = async ({ params, fetch, parent }) => {
	// Make sure the layout app fetching has happened before
	await parent();

	try {
		// Retrieve the last version of the app because the one used in the layout load may be outdated.
		const app = await service.fetchById(params.id, { fetch });

		return {
			app
		};
	} catch {
		throw error(404);
	}
};
