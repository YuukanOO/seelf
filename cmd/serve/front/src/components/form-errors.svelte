<script lang="ts">
	import { GLOBAL_ERROR_NAME, type SubmitterErrors } from '$lib/form';
	import Panel from '$components/panel.svelte';

	export let title = 'Error';
	export let errors: SubmitterErrors;

	let className = '';

	export { className as class };
</script>

{#if errors?.[GLOBAL_ERROR_NAME]}
	<Panel {title} class={className} format="inline" variant="danger">
		{errors[GLOBAL_ERROR_NAME]}
	</Panel>
{/if}
