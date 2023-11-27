<script lang="ts">
	import BlankSlate from '$components/blank-slate.svelte';
	import Breadcrumb from '$components/breadcrumb.svelte';
	import Link from '$components/link.svelte';
	import routes from '$lib/path';
	import type { PageData } from './$types';
	import l from '$lib/localization';

	export let data: PageData; // FIXME: don't know why Svelte does not found them automatically
</script>

<Breadcrumb
	segments={[
		{ path: routes.apps, title: l.translate('breadcrumb.applications') },
		{ path: routes.app(data.app.id), title: data.app.name },
		l.translate('breadcrumb.not_found')
	]}
/>

<BlankSlate>
	<p>
		{l.translate('deployment.not_found')}
		<Link href={routes.app(data.app.id)}>{l.translate('deployment.not_found.cta')}</Link>.
	</p>
</BlankSlate>
