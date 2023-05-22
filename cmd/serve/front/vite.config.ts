import { sveltekit } from '@sveltejs/kit/vite';

import { defineConfig } from 'vitest/config';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		proxy: {
			// During development, proxy api requests to the go server to make things transparent
			'/api/v1': 'http://127.0.0.1:8080'
		}
	},
	test: {
		include: ['src/**/*.{test,spec}.{js,ts}']
	}
});
