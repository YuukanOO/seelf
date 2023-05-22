<script lang="ts">
	import { messageFromAttributes } from '$lib/form';
	import InputAdorner from '$components/input-adorner.svelte';

	export let label: string;
	export let type: HtmlInputType = 'text';
	export let value: Maybe<string> = undefined;
	export let autofocus: Maybe<boolean> = false;
	export let autocomplete: Maybe<HtmlInputAutoComplete> = undefined;
	export let required = false;
	export let readonly = false;
	export let placeholder: Maybe<string> = undefined;
	export let remoteError: Maybe<string> = undefined;

	let touched = false;

	// Needed or svelte will complain about a dynamic type with two-way binding.
	const refType = (node: HTMLInputElement) => {
		node.type = type;
	};
</script>

<InputAdorner
	{label}
	{required}
	hasHelp={!!$$slots.default}
	{remoteError}
	error={messageFromAttributes({ required, type })}
>
	<!-- svelte-ignore a11y-autofocus -->
	<input
		{required}
		{readonly}
		{placeholder}
		{autofocus}
		{autocomplete}
		on:blur={() => {
			touched = true;
			remoteError = undefined;
		}}
		class="input"
		class:touched
		use:refType
		bind:value
	/>
	<slot slot="help" />
</InputAdorner>

<style module>
	.input {
		color: var(--co-text-5);
	}
</style>
