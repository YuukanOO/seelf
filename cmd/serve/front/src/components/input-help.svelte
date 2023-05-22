<script lang="ts">
	import Prose from '$components/prose.svelte';

	let className: string = '';

	/** Additional css classes */
	export { className as class };
</script>

<Prose class="help {className}">
	<slot />
</Prose>

<style module>
	.help {
		font: var(--ty-caption);
	}
</style>
