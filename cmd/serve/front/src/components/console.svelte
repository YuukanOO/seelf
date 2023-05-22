<script lang="ts">
	import { afterUpdate } from 'svelte';
	import type { Readable } from 'svelte/store';

	export let title: string;
	export let data: Readable<Maybe<string>>;

	let container: HTMLDivElement;

	afterUpdate(() => {
		// FIXME: maybe add a button "follow logs" to actually enable the scrollTo
		if (!container) {
			return;
		}

		container.scrollTo({ top: container.scrollHeight });
	});
</script>

<div class="console">
	<h2 class="title">{title}</h2>
	<div class="scrollarea" bind:this={container}>
		<pre>{$data}</pre>
	</div>
</div>

<style module>
	.console {
		background-color: var(--co-background-5);
		border: 1px solid var(--co-divider-4);
		border-radius: var(--ra-4);
	}

	.title {
		background-color: var(--co-background-4);
		border-block-end: 1px solid var(--co-divider-4);
		box-shadow: 0 10px 10px var(--co-background-4);
		border-start-start-radius: var(--ra-4);
		border-start-end-radius: var(--ra-4);
		font: var(--ty-caption);
		padding: var(--sp-2) var(--sp-4);
		position: relative;
	}

	.scrollarea {
		overflow: auto;
		padding: var(--sp-4);
		max-height: 40vh;
	}

	.console pre {
		font-family: var(--fo-mono);
		white-space: pre-wrap;
		word-break: break-word;
		margin: 0;
	}
</style>
