<script lang="ts">
	import BlankSlate from '$components/blank-slate.svelte';
	import Breadcrumb from '$components/breadcrumb.svelte';
	import Button from '$components/button.svelte';
	import CleanupNotice from '$components/cleanup-notice.svelte';
	import EnvironmentCard from '$components/environment-card.svelte';
	import Link from '$components/link.svelte';
	import routes from '$lib/path';
	import service from '$lib/resources/apps';
	import l from '$lib/localization';

	export let data;

	const { data: app } = service.queryById(data.app.id);

	$: ({ production, staging } = $app?.latest_deployments ?? {});
</script>

<Breadcrumb
	segments={[{ path: routes.apps, title: l.translate('breadcrumb.applications') }, data.app.name]}
>
	{#if $app?.cleanup_requested_at}
		<CleanupNotice requested_at={$app.cleanup_requested_at} />
	{:else}
		<Button href={routes.editApp(data.app.id)} variant="outlined" text="app.edit" />
		<Button href={routes.createDeployment(data.app.id)} text="deployment.new" />
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
			{l.translate('deployment.blankslate.title')}
			<Link href={routes.createDeployment(data.app.id)}>
				{l.translate('deployment.blankslate.cta')}
			</Link>
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
