<script lang="ts">
	let className: string = '';

	/** Additional css classes */
	export { className as class };

	export let as = 'div';
</script>

<svelte:element this={as} class="prose {className}">
	<slot />
</svelte:element>

<style module>
	.prose a {
		color: var(--co-primary-4);
	}

	.prose strong {
		color: var(--co-text-5);
		font-weight: 600;
	}

	.prose code {
		font-family: var(--fo-mono);
	}

	.prose > * + * {
		margin-block-start: var(--sp-4);
	}
</style>
