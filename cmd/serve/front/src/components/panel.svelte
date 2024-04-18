<script lang="ts">
	import Prose from '$components/prose.svelte';
	import Stack from '$components/stack.svelte';
	import l, { type AppTranslationsString } from '$lib/localization';

	export let title: AppTranslationsString;
	export let variant: 'help' | 'danger' | 'warning' | 'success' = 'help';
	export let format: 'default' | 'collapsable' | 'inline' = 'default';
	let className: string = '';

	/** Additional css classes */
	export { className as class };
</script>

{#if format === 'collapsable'}
	<details
		class="container {className}"
		style="--panel-border-color: var(--co-{variant}-3);--panel-background-color: var(--co-{variant}-1);--panel-title-color: var(--co-{variant}-4);"
	>
		<summary class="title collapsable">
			<Stack gap={1} justify="space-between">
				{l.translate(title)}
				<span class="hint">{l.translate('panel.hint')}</span>
			</Stack>
		</summary>
		<Prose class="content">
			<slot />
		</Prose>
	</details>
{:else}
	<Stack
		style="--panel-border-color: var(--co-{variant}-3);--panel-background-color: var(--co-{variant}-1);--panel-title-color: var(--co-{variant}-4);"
		class="container {className}"
		wrap={format === 'inline' ? 'wrap' : undefined}
		direction={format === 'inline' ? 'row' : 'column'}
		gap={2}
	>
		<div class="title">{l.translate(title)}</div>
		<Prose>
			<slot />
		</Prose>
	</Stack>
{/if}

<style module>
	.container {
		outline: 1px solid var(--panel-border-color);
		background-color: var(--panel-background-color);
		border-radius: var(--ra-4);
		color: var(--co-text-4);
		padding: var(--sp-2);
		font: var(--ty-caption);
	}

	details.container {
		padding: 0;
	}

	details .content {
		padding: var(--sp-2);
		padding-block-start: 0;
	}

	.title {
		color: var(--panel-title-color);
		font-weight: 600;
	}

	.title .hint {
		font: var(--ty-caption);
		font-style: italic;
		text-align: end;
		color: var(--co-text-4);
	}

	summary.title {
		cursor: pointer;
		display: block;
		padding: var(--sp-2);
	}

	summary.title::marker {
		content: '';
	}
</style>
