import service from '$lib/resources/registries';

export const load = async ({ fetch, depends }) => {
	const registries = await service.fetchAll({ fetch, depends });

	return {
		registries
	};
};
