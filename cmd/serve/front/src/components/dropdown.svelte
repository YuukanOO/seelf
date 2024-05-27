<script lang="ts" context="module">
	export type DropdownOption<T extends string> =
		| {
				label: string;
				value: T;
		  }
		| T;
</script>

<script lang="ts">
	import ArrowDown from '$assets/icons/arrow-down.svelte';
	import InputAdorner from '$components/input-adorner.svelte';
	import type { AppTranslationsString } from '$lib/localization';

	type T = $$Generic<string>;

	let touched = false;

	export let label: AppTranslationsString;
	export let disabled = false;
	export let options: DropdownOption<T>[] = [];
	export let value: Maybe<string> = undefined;
	export let remoteError: Maybe<string> = undefined;
</script>

<InputAdorner hasHelp={!!$$slots.default} {remoteError} {label}>
	<select
		class="input"
		{disabled}
		class:touched
		bind:value
		on:blur={() => {
			touched = true;
			remoteError = undefined;
		}}
	>
		{#each options as option}
			{#if typeof option === 'string'}
				<option value={option}>{option}</option>
			{:else}
				<option value={option.value}>{option.label}</option>
			{/if}
		{/each}
	</select>
	{#if !disabled}
		<ArrowDown />
	{/if}
	<slot slot="help" />
</InputAdorner>

<style module>
	.input {
		color: var(--co-text-5);
		cursor: pointer;
	}

	/** Since select::after is still not working in 2023... */
	.input + * {
		height: 1rem;
		width: 1rem;
		position: absolute;
		pointer-events: none;
		inset-inline-end: var(--sp-2);
		inset-block-start: 50%;
		transform: translateY(-50%);
	}

	.input:not([multiple]) option {
		background-color: var(--co-background-5);
		color: var(--co-text-5);
	}
</style>
