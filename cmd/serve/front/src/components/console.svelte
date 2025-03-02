<script lang="ts">
	import l, { type AppTranslationsString } from '$lib/localization';
	import { afterUpdate } from 'svelte';
	import Stack from './stack.svelte';
	import Button from './button.svelte';

	export let id: Maybe<string> = undefined;
	export let title: AppTranslationsString;
	export let data: Maybe<string>;
	export let titleElement = 'h2';

	/** Show the select all button */
	export let selectAllEnabled: boolean = false;

	/** Show the copy to clipboard button */
	export let copyToClipboardEnabled: boolean = false;

	let container: HTMLDivElement;
	let pre: HTMLPreElement;
	let lastScroll: number = 0;
	let followLogs = true;

	$: copyToClipboardAvailable =
		copyToClipboardEnabled && navigator.clipboard && navigator.clipboard.writeText;

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

	function copyToClipboard() {
		if (!data) {
			return;
		}

		navigator.clipboard.writeText(data);
	}

	function selectAll() {
		if (!pre) {
			return;
		}

		const selection = window.getSelection();
		const range = document.createRange();

		range.selectNodeContents(pre);
		selection?.removeAllRanges();
		selection?.addRange(range);
	}

	afterUpdate(() => {
		if (!container || !followLogs) {
			return;
		}

		container.scrollTo({ top: container.scrollHeight });
	});
</script>

<div class="console" {id}>
	<Stack class="header" direction="row" justify="space-between" wrap="wrap">
		<svelte:element this={titleElement}>{l.translate(title)}</svelte:element>
		{#if copyToClipboardAvailable || selectAllEnabled}
			<Stack direction="row">
				{#if copyToClipboardAvailable}
					<Button variant="outlined" text="console.copy_to_clipboard" on:click={copyToClipboard} />
				{/if}
				{#if selectAllEnabled}
					<Button variant="outlined" text="console.select_all" on:click={selectAll} />
				{/if}
			</Stack>
		{/if}
	</Stack>
	<div class="scrollarea" bind:this={container} on:scroll={onScroll}>
		<pre bind:this={pre}>{data}</pre>
	</div>
</div>

<style module>
	.console {
		background-color: var(--co-background-5);
		border: 1px solid var(--co-divider-4);
		border-radius: var(--ra-4);
	}

	.header {
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
