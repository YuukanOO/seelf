import service from '$lib/resources/deployments';
import { error } from '@sveltejs/kit';

export const load = async ({ fetch, params: { id, number }, depends }) => {
	try {
		const deployment = await service.fetchByAppAndNumber(id, parseInt(number), { fetch, depends });

		return {
			deployment
		};
	} catch {
		throw error(404);
	}
};
