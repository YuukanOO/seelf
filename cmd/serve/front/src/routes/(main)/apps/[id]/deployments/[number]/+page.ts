import service from '$lib/resources/deployments';
import appService from '$lib/resources/apps';
import { error } from '@sveltejs/kit';

export const load = async ({ fetch, params: { id, number }, depends }) => {
	try {
		const deployment = await service.fetchByAppAndNumber(id, parseInt(number), { fetch, depends });
		const app = await appService.fetchById(id, { fetch, depends });

		return {
			app,
			deployment
		};
	} catch {
		throw error(404);
	}
};
