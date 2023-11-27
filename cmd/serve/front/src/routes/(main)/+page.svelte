<script lang="ts">
	import AppCard from '$components/app-card.svelte';
	import BlankSlate from '$components/blank-slate.svelte';
	import Breadcrumb from '$components/breadcrumb.svelte';
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
	<div class="grid">
		{#each $data as app}
			<AppCard data={app} />
		{/each}
	</div>
{:else}
	<BlankSlate>
		<p>
			{l.translate('app.blankslate.title')}
			<Link href={routes.createApp}>{l.translate('app.blankslate.cta')}</Link>
		</p>
	</BlankSlate>
{/if}

<style module>
	.grid {
		align-items: flex-start;
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(15rem, 1fr));
		gap: var(--sp-4);
	}
</style>
