<script lang="ts">
	import { messageFromAttributes } from '$lib/form';
	import InputAdorner from '$components/input-adorner.svelte';

	let touched = false;

	export let label: string;
	export let accept: Maybe<string> = undefined;
	export let required = false;
	export let files: Maybe<FileList> = undefined;
</script>

<InputAdorner
	{label}
	{required}
	hasHelp={!!$$slots.default}
	error={messageFromAttributes({ required })}
>
	<input
		type="file"
		class="input"
		{required}
		{accept}
		class:touched
		bind:files
		on:blur={() => (touched = true)}
	/>
	<slot slot="help" />
</InputAdorner>

<style module>
	.input {
		color: var(--co-text-5);
		cursor: pointer;
	}
</style>
