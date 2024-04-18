import service from '$lib/resources/targets';

export const load = async ({ fetch, depends }) => {
	const targets = await service.fetchAll(undefined, { fetch, depends });

	return {
		targets
	};
};
