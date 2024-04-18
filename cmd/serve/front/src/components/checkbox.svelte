<script lang="ts">
	import Stack from '$components/stack.svelte';
	import InputHelp from '$components/input-help.svelte';
	import l, { type AppTranslationsString } from '$lib/localization';

	export let label: AppTranslationsString;
	export let disabled: boolean = false;
	export let checked: boolean = false;
</script>

<Stack as="label" class="container" gap={2}>
	<input type="checkbox" {disabled} class="checkbox" bind:checked />
	<Stack direction="column" gap={0}>
		<span>{l.translate(label)}</span>
		{#if $$slots.default}
			<InputHelp>
				<slot />
			</InputHelp>
		{/if}
	</Stack>
</Stack>

<style module>
	.container {
		cursor: pointer;
	}

	.checkbox {
		background-color: transparent;
		border: 1px solid var(--co-divider-4);
		border-radius: var(--ra-2);
		display: block;
		height: var(--sp-4);
		width: var(--sp-4);
		flex-shrink: 0;
	}

	.checkbox:checked {
		background-color: var(--co-primary-4);
		border-color: var(--co-primary-4);
		box-shadow: inset 0 0 0 2px var(--co-background-5);
	}

	.checkbox:focus,
	.checkbox:active {
		border-color: var(--co-primary-4);
		outline: var(--ou-primary);
	}
</style>
