<script lang="ts">
	import BlankSlate from '$components/blank-slate.svelte';
	import Breadcrumb from '$components/breadcrumb.svelte';
	import Link from '$components/link.svelte';
	import routes from '$lib/path';
	import l from '$lib/localization';
</script>

<Breadcrumb
	segments={[
		{ path: routes.targets, title: l.translate('breadcrumb.targets') },
		l.translate('breadcrumb.not_found')
	]}
/>

<BlankSlate>
	<p>
		{l.translate('target.not_found')}
		<Link href={routes.targets}>{l.translate('target.not_found.cta')}</Link>.
	</p>
</BlankSlate>
