<script lang="ts">
	let className: string = '';

	/** Additional css classes */
	export { className as class };

	/** Till there's no way to pass down empty slot without messing with the $$slots, accept a boolean */
	export let hasFooter: Maybe<boolean> = undefined;
	export let color: 'divider' | 'pending' | 'running' | 'success' | 'error' = 'divider';

	const showFooter = hasFooter ?? $$slots.footer;
</script>

<div
	class="card ${className}"
	class:no-footer={!showFooter}
	style="--card-color: var(--co-{color}-4)"
>
	<div class="body">
		<slot />
	</div>

	{#if showFooter}
		<div class="footer">
			<slot name="footer" />
		</div>
	{/if}
</div>

<style module>
	.card {
		background-color: var(--co-background-5);
		border: 1px solid var(--card-color);
		border-radius: var(--ra-4);
	}

	.body {
		background-image: linear-gradient(to top, var(--co-background-5), 94%, transparent),
			repeating-linear-gradient(
				-45deg,
				transparent,
				transparent 8px,
				var(--card-color) 8px,
				var(--card-color) 10px
			);
		border-start-start-radius: var(--ra-4);
		border-start-end-radius: var(--ra-4);
		padding: var(--sp-4);
	}

	.card.no-footer .body {
		border-end-start-radius: var(--ra-4);
		border-end-end-radius: var(--ra-4);
	}

	.footer {
		background-color: var(--co-background-4);
		border-radius: var(--ra-4);
		padding: var(--sp-4);
	}
</style>
