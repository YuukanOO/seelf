<script lang="ts">
	export let visible = false;
</script>

{#if visible}
	<div class="progress-bar" />
{/if}

<style module>
	.progress-bar {
		height: 0.1rem;
		background: linear-gradient(
			to right,
			transparent 0%,
			var(--co-primary-4) 50%,
			transparent 100%
		);
		position: fixed;
		inset-block-start: 0;
		animation: progress 2s linear infinite alternate;
		width: 10%;
		z-index: 20;
	}

	@keyframes progress {
		0% {
			inset-inline-start: -10%;
		}

		100% {
			inset-inline-start: 100%;
		}
	}
</style>
