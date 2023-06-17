<script lang="ts">
	import { afterUpdate } from 'svelte';

	export let title: string;
	export let data: Maybe<string>;

	let container: HTMLDivElement;
	let lastScroll: number = 0;
	let followLogs = true;

	function onScroll(event: Event) {
		const target = event.target as HTMLDivElement;
		const { scrollTop, scrollHeight, clientHeight } = target;

		// If the user has scrolled down, activate the follow only if it has reached the bottom
		if (scrollTop > lastScroll) {
			followLogs = Math.floor(scrollHeight - scrollTop) === clientHeight;
		} else {
			followLogs = false;
		}

		lastScroll = target.scrollTop;
	}

	afterUpdate(() => {
		if (!container || !followLogs) {
			return;
		}

		container.scrollTo({ top: container.scrollHeight });
	});
</script>

<div class="console">
	<h2 class="title">{title}</h2>
	<div class="scrollarea" bind:this={container} on:scroll={onScroll}>
		<pre>{data}</pre>
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
