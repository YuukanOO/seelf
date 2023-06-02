<script lang="ts">
	import AppCard from '$components/app-card.svelte';
	import BlankSlate from '$components/blank-slate.svelte';
	import Breadcrumb from '$components/breadcrumb.svelte';
	import Button from '$components/button.svelte';
	import Link from '$components/link.svelte';
	import routes from '$lib/path';
	import service from '$lib/resources/apps';

	const { data } = service.queryAll();
</script>

<Breadcrumb segments={['Applications']}>
	<Button href={routes.createApp}>New application</Button>
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
			Looks like you have no application yet. Start by <Link href={routes.createApp}
				>creating one</Link
			>!
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
