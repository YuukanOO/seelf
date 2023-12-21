<script lang="ts">
	import Prose from '$components/prose.svelte';
	import l, { type AppTranslationsString } from '$lib/localization';

	export let label: AppTranslationsString;

	let className: string = '';

	/** Additional css classes */
	export { className as class };
</script>

<div class={className}>
	<h3 class="label">{l.translate(label)}</h3>
	<Prose class="value"><slot /></Prose>
</div>

<style module>
	.label {
		font: var(--ty-caption);
	}

	.value {
		color: var(--co-text-5);
		text-overflow: ellipsis;
		overflow: hidden;
	}
</style>
