<script lang="ts">
	import InputAdorner from '$components/input-adorner.svelte';
	import { messageFromAttributes } from '$lib/form';
	import type { AppTranslationsString } from '$lib/localization';

	let touched = false;

	export let label: AppTranslationsString;
	export let rows: Maybe<number> = undefined;
	export let placeholder: Maybe<string> = undefined;
	export let value: Maybe<string> = undefined;
	export let required = false;
	export let readonly = false;
	export let code = false;
	export let remoteError: Maybe<string> = undefined;

	let className: string = '';

	/** Additional css classes */
	export { className as class };
</script>

<InputAdorner
	{label}
	{required}
	hasHelp={!!$$slots.default}
	{remoteError}
	error={messageFromAttributes({ required })}
>
	<textarea
		{required}
		class="input {className}"
		class:touched
		class:code
		{rows}
		{placeholder}
		{readonly}
		bind:value
		on:blur={() => (touched = true)}
	/>
	<slot slot="help" />
</InputAdorner>

<style module>
	.input {
		color: var(--co-text-5);
	}

	.input.code {
		font-family: var(--fo-mono);
	}
</style>
