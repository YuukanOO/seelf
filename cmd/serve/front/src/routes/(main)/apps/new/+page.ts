import service from '$lib/resources/targets';

export const load = async ({ fetch, depends }) => {
	const targets = await service.fetchAll({ active_only: true }, { fetch, depends });

	return {
		targets
	};
};
