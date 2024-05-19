<script lang="ts">
	import type { Service } from '$lib/resources/deployments';
	import Stack from '$components/stack.svelte';
	import EntrypointPill from '$components/entrypoint-pill.svelte';

	export let data: Service;
</script>

<Stack as="article" class="container" direction="column" gap={2}>
	<Stack justify="space-between">
		<div>{data.name}</div>
		<div class="image-pill">{data.image}</div>
	</Stack>
	{#if data.entrypoints && data.entrypoints.length > 0}
		<Stack as="ul" class="entrypoints" wrap="wrap" gap={0}>
			{#each data.entrypoints as entrypoint (entrypoint.name)}
				<li class="entrypoint">
					<EntrypointPill data={entrypoint} />
				</li>
			{/each}
		</Stack>
	{/if}
</Stack>

<style module>
	.container {
		background-color: var(--co-background-6);
		border: 1px solid var(--co-divider-4);
		border-radius: var(--ra-4);
		padding: var(--sp-2);
	}

	.entrypoints {
		margin: calc(-1 * var(--ou-size));
	}

	.entrypoint {
		overflow: hidden;
		padding: var(--ou-size);
	}

	.image-pill {
		background-color: var(--co-background-5);
		border-radius: var(--ra-4);
		border: 1px solid var(--co-divider-4);
		font: var(--ty-caption);
		color: var(--co-text-4);
		padding: 0.125rem var(--sp-2);
		white-space: nowrap;
		text-overflow: ellipsis;
		overflow: hidden;
	}
</style>
