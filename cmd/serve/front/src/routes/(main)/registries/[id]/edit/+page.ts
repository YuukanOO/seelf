import service from '$lib/resources/registries';
import { error } from '@sveltejs/kit';

export const load = async ({ params, fetch, depends }) => {
	try {
		const registry = await service.fetchById(params.id, { fetch, depends });

		return {
			registry
		};
	} catch {
		throw error(404);
	}
};
