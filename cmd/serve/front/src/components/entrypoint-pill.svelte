<script lang="ts">
	import type { Entrypoint } from '$lib/resources/deployments';

	export let data: Entrypoint;
</script>

{#if data.url}
	<a class="entry" href={data.url} rel="noopener noreferrer" target="_blank">
		<span class="port">{data.port}</span>
		<span class="content">{data.url}</span>
	</a>
{:else}
	<div class="entry">
		<span class="port">{data.port}</span>
		<span class="content">{data.router}</span>
	</div>
{/if}

<style module>
	.entry {
		border: 1px solid var(--co-primary-4);
		background-color: var(--co-background-5);
		border-radius: var(--ra-4);
		border: 1px solid var(--co-divider-4);
		color: var(--co-text-4) !important;
		display: flex;
		align-items: center;
		font: var(--ty-caption);
	}

	a.entry:hover,
	a.entry:focus {
		border-color: var(--co-primary-4);
		color: var(--co-primary-4) !important;
		text-decoration: none;
	}

	a.entry:focus {
		outline: var(--ou-primary);
	}

	.port {
		background-color: var(--co-warning-1);
		border-inline-end: 1px solid var(--co-divider-4);
		border-start-start-radius: var(--ra-4);
		border-end-start-radius: var(--ra-4);
		color: var(--co-warning-4);
		padding: var(--sp-1);
		flex: 0;
	}

	a.entry .port {
		background-color: var(--co-primary-1);
		color: var(--co-primary-4);
	}

	.content {
		padding: var(--sp-1) var(--sp-2);
		flex: 1;
		white-space: nowrap;
		text-overflow: ellipsis;
		overflow: hidden;
	}
</style>
