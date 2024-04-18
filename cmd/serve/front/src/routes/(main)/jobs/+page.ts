import service from '$lib/resources/jobs';

export const load = async ({ fetch, depends }) => {
	const targets = await service.fetchAll(1, { fetch, depends });

	return {
		targets
	};
};
