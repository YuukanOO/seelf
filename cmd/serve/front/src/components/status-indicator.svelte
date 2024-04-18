<script lang="ts">
	export let state: 'success' | 'failed' | 'running' | 'pending' = 'pending';
</script>

<span
	class="status-indicator"
	class:success={state === 'success'}
	class:failed={state === 'failed'}
	class:running={state === 'running'}
/>

<style module>
	.status-indicator {
		box-shadow: inset 0 0 0 2px var(--co-background-4);
		background-color: var(--co-pending-4);
		border-color: var(--co-pending-4);
		border-width: 2px;
		border-style: solid;
		display: block;
		flex-shrink: 0;
		height: var(--sp-4);
		width: var(--sp-4);
		border-radius: 50%;
		position: relative;
		z-index: 1;
	}

	.status-indicator.success {
		background-color: var(--co-success-4);
		border-color: var(--co-success-4);
	}

	.status-indicator.failed {
		background-color: var(--co-error-4);
		border-color: var(--co-error-4);
	}

	.status-indicator.running {
		background-color: var(--co-running-4);
		border-color: var(--co-running-4) var(--co-running-4) var(--co-background-4);
		animation: rotation 1s linear infinite;
	}

	@keyframes rotation {
		0% {
			transform: rotate(0deg);
		}
		100% {
			transform: rotate(360deg);
		}
	}
</style>
