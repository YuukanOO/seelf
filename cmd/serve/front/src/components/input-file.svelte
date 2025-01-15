<script lang="ts">
	import InputAdorner from '$components/input-adorner.svelte';
	import { messageFromAttributes } from '$lib/form';
	import type { AppTranslationsString } from '$lib/localization';

	let touched = false;

	export let label: AppTranslationsString;
	export let accept: Maybe<string> = undefined;
	export let required = false;
	export let remoteError: Maybe<string> = undefined;
	export let files: Maybe<FileList> = undefined;
</script>

<InputAdorner
	{label}
	{required}
	{remoteError}
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
		on:blur={() => {
			touched = true;
			remoteError = undefined;
		}}
	/>
	<slot slot="help" />
</InputAdorner>

<style module>
	.input {
		color: var(--co-text-5);
		cursor: pointer;
	}
</style>
