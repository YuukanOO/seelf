import adapter from '@sveltejs/adapter-static';
import preprocess from 'svelte-preprocess';
import { cssModules, linearPreprocess } from 'svelte-preprocess-cssmodules';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	// Consult https://github.com/sveltejs/svelte-preprocess
	// for more information about preprocessors
	preprocess: linearPreprocess([preprocess(), cssModules()]),

	kit: {
		alias: {
			$components: 'src/components',
			$assets: 'src/assets'
		},
		adapter: adapter({
			fallback: 'fallback.html' // Enable true SPA mode since some pages could not be pregenerated (ex: apps pages)
		})
	}
};

export default config;
