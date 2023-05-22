<script lang="ts">
	import { isSet } from '$lib/collections';
	import routes from '$lib/path';
	import type { AppData } from '$lib/resources/apps';
	import Link from '$components/link.svelte';
	import Card from '$components/card.svelte';
	import DeploymentsList from '$components/deployments-list.svelte';
	import Stack from '$components/stack.svelte';
	import CleanupNotice from '$components/cleanup-notice.svelte';

	export let data: AppData;

	$: ({ production, staging } = data.environments);
</script>

<Card>
	<Stack direction="column">
		<h2 class="title"><Link href={routes.app(data.id)}>{data.name}</Link></h2>
		<CleanupNotice {data} />
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
</style>
