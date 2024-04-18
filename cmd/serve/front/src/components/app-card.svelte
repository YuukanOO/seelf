<script lang="ts">
	import { isSet } from '$lib/collections';
	import routes from '$lib/path';
	import type { App } from '$lib/resources/apps';
	import Link from '$components/link.svelte';
	import Card from '$components/card.svelte';
	import DeploymentsList from '$components/deployments-list.svelte';
	import Stack from '$components/stack.svelte';
	import CleanupNotice from '$components/cleanup-notice.svelte';

	export let data: App;

	$: ({ production, staging } = data.latest_deployments);
</script>

<Card>
	<Stack direction="column">
		<div>
			<h2 class="title"><Link href={routes.app(data.id)}>{data.name}</Link></h2>
			<div class="targets">
				{#if data.production_target.id === data.staging_target.id}
					<Link href={routes.editTarget(data.production_target.id)}
						>{data.production_target.name}</Link
					>
				{:else}
					<Link href={routes.editTarget(data.production_target.id)}>
						{data.production_target.name}
					</Link>
					/
					<Link href={routes.editTarget(data.staging_target.id)}>{data.staging_target.name}</Link>
				{/if}
			</div>
		</div>
		<CleanupNotice requested_at={data.cleanup_requested_at} />
	</Stack>
	<DeploymentsList variant="env" slot="footer" data={[production, staging].filter(isSet)} />
</Card>

<style module>
	.title {
		color: var(--co-text-5);
		font: var(--ty-heading-2);
		white-space: nowrap;
		text-overflow: ellipsis;
		overflow: hidden;
	}

	.targets {
		font: var(--ty-caption);
	}
</style>
