import service from '$lib/resources/apps';
import { error } from '@sveltejs/kit';

export const load = async ({ params, fetch }) => {
	try {
		const app = await service.fetchById(params.id, { fetch });

		return {
			app
		};
	} catch {
		throw error(404);
	}
};
