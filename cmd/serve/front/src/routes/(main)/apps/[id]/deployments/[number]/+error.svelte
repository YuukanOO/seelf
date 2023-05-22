<script lang="ts">
	import BlankSlate from '$components/blank-slate.svelte';
	import Breadcrumb from '$components/breadcrumb.svelte';
	import Link from '$components/link.svelte';
	import routes from '$lib/path';
	import type { PageData } from './$types';

	export let data: PageData; // FIXME: don't know why Svelte does not found them automatically
</script>

<Breadcrumb
	segments={[
		{ path: routes.apps, title: 'Applications' },
		{ path: routes.app(data.app.id), title: data.app.name },
		'Not found'
	]}
/>

<BlankSlate>
	<p>
		Looks like the deployment you're looking for does not exist! Head back to <Link
			href={routes.app(data.app.id)}>the application page</Link
		>.
	</p>
</BlankSlate>
