import service from '$lib/resources/apps';
import targetsService from '$lib/resources/targets';
import { error } from '@sveltejs/kit';

export const load = async ({ params, fetch, depends }) => {
	try {
		// Retrieve the last version of the app because the one used in the layout load may be outdated.
		const app = await service.fetchById(params.id, { fetch, depends });
		const targets = await targetsService.fetchAll({ active_only: true }, { fetch, depends });

		return {
			app,
			targets
		};
	} catch {
		throw error(404);
	}
};
