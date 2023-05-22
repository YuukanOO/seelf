/**
 * Expose application routes.
 */
const routes = {
	signin: (redirectTo?: string) =>
		redirectTo ? `/signin?redirectTo=${encodeURIComponent(redirectTo)}` : '/signin',
	profile: '/profile',
	apps: '/',
	createApp: '/apps/new',
	editApp: (id: string) => `/apps/${id}/edit`,
	app: (id: string) => `/apps/${id}`,
	createDeployment: (id: string) => `/apps/${id}/deployments/new`,
	deployment: (id: string, number: number) => `/apps/${id}/deployments/${number}`
} as const;

export default routes;
