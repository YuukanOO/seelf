<script lang="ts">
	import type { ComponentProps } from 'svelte';
	import Card from '$components/card.svelte';
	import Link from '$components/link.svelte';
	import Stack from '$components/stack.svelte';
	import CleanupNotice from '$components/cleanup-notice.svelte';
	import routes from '$lib/path';
	import { type Target, TargetStatus } from '$lib/resources/targets';

	export let data: Target;

	function colorForStatus(status: TargetStatus): ComponentProps<Card>['color'] {
		switch (status) {
			case TargetStatus.Configuring:
				return 'running';
			case TargetStatus.Ready:
				return 'success';
			case TargetStatus.Failed:
				return 'error';
			default:
				return 'pending';
		}
	}
</script>

<Card color={colorForStatus(data.state.status)}>
	<Stack direction="column">
		<div>
			<h2 class="title"><Link href={routes.editTarget(data.id)}>{data.name}</Link></h2>
			<div class="url">{data.url}</div>
		</div>
		<CleanupNotice requested_at={data.cleanup_requested_at} />
	</Stack>
</Card>

<style module>
	.title {
		color: var(--co-text-5);
		font: var(--ty-heading-2);
		white-space: nowrap;
		text-overflow: ellipsis;
		overflow: hidden;
	}

	.url {
		font: var(--ty-caption);
		white-space: nowrap;
		text-overflow: ellipsis;
		overflow: hidden;
	}
</style>
