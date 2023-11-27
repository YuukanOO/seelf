<script lang="ts">
	import InputHelp from '$components/input-help.svelte';
	import l, { type AppTranslationsString } from '$lib/localization';

	export let label: AppTranslationsString;
	export let error: Maybe<string> = undefined;
	export let remoteError: Maybe<string> = undefined;
	/** Determine wether or not there's a star next to the label */
	export let required: boolean = false;

	$: remoteErrorText =
		remoteError && l.translate(remoteError as AppTranslationsString).toLowerCase();

	/**
	 *  Since there's no way to check for "emptiness", assume the caller gives us
	 * a boolean.
	 */
	export let hasHelp: Maybe<boolean> = undefined;

	const showHelp = hasHelp ?? $$slots.help;
</script>

<div class="adorner">
	<label class="container">
		<span class="label" class:required>{l.translate(label)}</span>
		<slot />
		{#if remoteError}<span class="remote-error">{remoteErrorText}</span>{/if}
		{#if error}<span class="error">{error}</span>{/if}
	</label>

	{#if showHelp}
		<InputHelp class="help">
			<slot name="help" />
		</InputHelp>
	{/if}
</div>

<style module>
	.adorner {
		position: relative;
		width: 100%;
	}

	.container {
		border: 1px solid var(--co-divider-4);
		display: block;
		border-radius: var(--ra-4);
		padding: var(--sp-2);
		position: relative;
	}

	/** What follows the label will be the input, let it span! */
	.label + * {
		display: block;
		width: 100%;
	}

	.label {
		display: block;
		cursor: pointer;
		font: var(--ty-caption);
	}

	.label.required::after {
		content: ' *';
	}

	.help {
		margin-block-start: var(--sp-3);
		padding-inline: var(--sp-2);
	}

	.input {
		color: var(--co-text-5);
		display: block;
		width: 100%;
	}

	.container:focus-within {
		border-color: var(--co-primary-4);
		outline: var(--ou-primary);
	}

	.error,
	.remote-error {
		background-color: var(--co-background-5);
		padding: 0 var(--sp-1);
		margin: 0 calc(var(--sp-1) * -1);
		color: var(--co-error-4);
		font: var(--ty-caption);
		position: absolute;
	}

	/** FIXME: when :user-invalid will be correctly implemented, this will be easier */

	:global(.touched:invalid) ~ .remote-error,
	.error {
		display: none;
	}

	:global(.touched:invalid) ~ .error {
		display: block;
	}
</style>
