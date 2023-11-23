<script lang="ts">
	import { GLOBAL_ERROR_NAME, type SubmitterErrors } from '$lib/form';
	import Panel from '$components/panel.svelte';
	import l, { type AppTranslationsString } from '$lib/localization';

	export let title: AppTranslationsString = 'error';
	export let errors: SubmitterErrors;

	let className = '';

	$: globalErrorText =
		errors?.[GLOBAL_ERROR_NAME] && l.translate(errors[GLOBAL_ERROR_NAME] as AppTranslationsString);

	export { className as class };
</script>

{#if globalErrorText}
	<Panel {title} class={className} format="inline" variant="danger">
		{globalErrorText}
	</Panel>
{/if}
