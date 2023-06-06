<script lang="ts">
	import BlankSlate from '$components/blank-slate.svelte';
	import Breadcrumb from '$components/breadcrumb.svelte';
	import Button from '$components/button.svelte';
	import CleanupNotice from '$components/cleanup-notice.svelte';
	import EnvironmentCard from '$components/environment-card.svelte';
	import Link from '$components/link.svelte';
	import routes from '$lib/path';
	import service from '$lib/resources/apps';

	export let data;

	const { data: app } = service.queryById(data.app.id);

	$: ({ production, staging } = $app?.environments ?? {});
</script>

<Breadcrumb segments={[{ path: routes.apps, title: 'Applications' }, data.app.name]}>
	{#if $app?.cleanup_requested_at}
		<CleanupNotice data={$app} />
	{:else}
		<Button href={routes.editApp(data.app.id)} variant="outlined">Edit application</Button>
		<Button href={routes.createDeployment(data.app.id)}>New deployment</Button>
	{/if}
</Breadcrumb>

{#if production || staging}
	<div class="grid">
		{#if production}
			<EnvironmentCard data={production} />
		{/if}
		{#if staging}
			<EnvironmentCard data={staging} />
		{/if}
	</div>
{:else}
	<BlankSlate>
		<p>
			No deployment to show. Go ahead and <Link href={routes.createDeployment(data.app.id)}
				>create the first one</Link
			>!
		</p>
	</BlankSlate>
{/if}

<style module>
	.grid {
		align-items: flex-start;
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(calc(var(--mi-container-width) / 3), 1fr));
		gap: var(--sp-4);
	}
</style>
