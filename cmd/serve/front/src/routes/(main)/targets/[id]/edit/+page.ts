import service from '$lib/resources/targets';
import { error } from '@sveltejs/kit';

export const load = async ({ params, fetch, depends }) => {
	try {
		const target = await service.fetchById(params.id, { fetch, depends });

		return {
			target
		};
	} catch {
		throw error(404);
	}
};
