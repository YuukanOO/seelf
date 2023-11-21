<script lang="ts">
	import { submitter } from '$lib/form';

	let className: string = '';

	/** Async handler of the form, will be called upon submit. */
	export let handler: () => Promise<unknown>;

	export let disabled: Maybe<boolean> = undefined;
	export let autocomplete: Maybe<HtmlFormAutoComplete> = undefined;

	/** Additional css classes */
	export { className as class };

	const { submit, loading, errors } = submitter(handler);
</script>

<form on:submit|preventDefault={submit} {autocomplete}>
	<fieldset class={className} disabled={$loading || disabled}>
		<slot submitting={$loading} errors={$errors} />
	</fieldset>
</form>
