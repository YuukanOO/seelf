import service from '$lib/resources/apps';

export const load = async ({ fetch }) => {
	const apps = await service.fetchAll({ fetch });

	return {
		apps
	};
};
