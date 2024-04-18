<script lang="ts">
	import AppCard from '$components/app-card.svelte';
	import BlankSlate from '$components/blank-slate.svelte';
	import Breadcrumb from '$components/breadcrumb.svelte';
	import CardsGrid from '$components/cards-grid.svelte';
	import Button from '$components/button.svelte';
	import Link from '$components/link.svelte';
	import routes from '$lib/path';
	import service from '$lib/resources/apps';
	import l from '$lib/localization';

	const { data } = service.queryAll();
</script>

<Breadcrumb segments={[l.translate('breadcrumb.applications')]}>
	<Button href={routes.createApp} text="app.new" />
</Breadcrumb>

{#if $data && $data.length > 0}
	<CardsGrid>
		{#each $data as app (app.id)}
			<AppCard data={app} />
		{/each}
	</CardsGrid>
{:else}
	<BlankSlate>
		<p>
			{l.translate('app.blankslate.title')}
			<Link href={routes.createApp}>{l.translate('app.blankslate.cta')}</Link>
		</p>
	</BlankSlate>
{/if}
