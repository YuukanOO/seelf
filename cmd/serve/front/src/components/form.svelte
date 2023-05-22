<script lang="ts">
	import { ValidationError } from '$lib/error';

	let submitting = false;
	let errors: Record<string, Maybe<string>> = {};
	let className: string = '';

	/** Async handler of the form, will be called upon submit. */
	export let handler: () => Promise<unknown>;

	export let disabled: Maybe<boolean> = undefined;
	export let autocomplete: Maybe<HtmlFormAutoComplete> = undefined;

	/** Additional css classes */
	export { className as class };

	async function onSubmit() {
		try {
			submitting = true;
			await handler();
		} catch (ex) {
			if (ex instanceof ValidationError) {
				errors = ex.fields;
			} else {
				console.error(ex);
			}
		} finally {
			submitting = false;
		}
	}
</script>

<form on:submit|preventDefault={onSubmit} {autocomplete}>
	<fieldset class={className} disabled={submitting || disabled}>
		<slot {submitting} {errors} />
	</fieldset>
</form>
