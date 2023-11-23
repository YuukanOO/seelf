<script lang="ts">
	import Stack from '$components/stack.svelte';
	import l, { type AppTranslationsString } from '$lib/localization';

	export let title: AppTranslationsString;

	/** Render the section title but does not wrap the inner slot in a specific container style */
	export let transparent: boolean = false;
</script>

<fieldset class="section">
	<Stack justify="space-between">
		<legend class="title">{l.translate(title)}</legend>
		{#if $$slots.actions}
			<Stack>
				<slot name="actions" />
			</Stack>
		{/if}
	</Stack>
	<div class="content" class:transparent>
		<slot />
	</div>
</fieldset>

<style module>
	.section + .section {
		margin-block-start: var(--sp-6);
	}

	.title {
		color: var(--co-primary-4);
		font: var(--ty-heading-2);
	}

	.content {
		background-color: var(--co-background-5);
		border-radius: var(--ra-4);
		padding: var(--sp-4);
		margin-block-start: var(--sp-2);
	}

	.content.transparent {
		background-color: transparent;
		padding: 0;
	}

	@media screen and (min-width: 56rem) {
		.content {
			padding: var(--sp-6);
		}
	}
</style>
