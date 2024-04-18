<script lang="ts">
	import Prose from '$components/prose.svelte';
	import l, { type AppTranslationsString } from '$lib/localization';

	export let label: AppTranslationsString;

	let className: string = '';

	/** Additional css classes */
	export { className as class };
</script>

<div class={className}>
	<dt class="label">{l.translate(label)}</dt>
	<Prose as="dd" class="value"><slot /></Prose>
</div>

<style module>
	.label {
		font: var(--ty-caption);
		color: var(--co-text-4);
	}

	.value {
		color: var(--co-text-5);
		text-overflow: ellipsis;
		overflow: hidden;
	}
</style>
