<script lang="ts">
	import { formatDatetime, formatDuration } from '$lib/date';
	import { DeploymentStatus, type DeploymentData } from '$lib/resources/deployments';
	import Loader from '$components/loader.svelte';
	import Stack from '$components/stack.svelte';
	import DeploymentPill from '$components/deployment-pill.svelte';

	export let data: Maybe<DeploymentData[]> = undefined;
	export let variant: 'env' | 'detail';

	/** Retrieve the metadata information based on the deployment status */
	function metadataFromStatus(depl: DeploymentData): string {
		switch (depl.state.status) {
			case DeploymentStatus.Succeeded:
				return `${formatDatetime(depl.state.finished_at!)} (${formatDuration(
					depl.state.started_at!,
					depl.state.finished_at!
				)})`;
			case DeploymentStatus.Failed:
				return `${depl.state.error_code} (${formatDuration(
					depl.state.started_at!,
					depl.state.finished_at!
				)})`;
			case DeploymentStatus.Running:
				return `${formatDatetime(depl.state.started_at!)} (${formatDuration(
					depl.state.started_at!,
					new Date()
				)})`;
			default:
				return `${formatDatetime(depl.requested_at)}`;
		}
	}
</script>

<div class="container" class:loading={!data}>
	{#if data}
		{#if data.length > 0}
			<ul class="deployments">
				{#each data as depl}
					<Stack as="li" gap={2} justify="space-between">
						<Stack gap={2}>
							<span
								class="status"
								class:success={depl.state.status === DeploymentStatus.Succeeded}
								class:failed={depl.state.status === DeploymentStatus.Failed}
								class:running={depl.state.status === DeploymentStatus.Running}
							/>
							<div>
								<div class="title">
									{variant === 'env' ? depl.environment : formatDatetime(depl.requested_at)}
								</div>
								<div class="metadata">
									{metadataFromStatus(depl)}
								</div>
							</div>
						</Stack>
						<DeploymentPill data={depl} />
					</Stack>
				{/each}
			</ul>
		{:else}
			<p class="no-data">No deployment yet</p>
		{/if}
	{:else}
		<Loader />
	{/if}
</div>

<style module>
	.no-data {
		font-style: italic;
		text-align: center;
	}

	.container {
		position: relative;
	}

	.container.loading {
		display: flex;
		place-content: center;
	}

	.metadata {
		font: var(--ty-caption);
	}

	.title {
		font-weight: 600;
	}

	.deployments li + li {
		margin-block-start: var(--sp-2);
	}

	.deployments::before {
		content: '';
		display: block;
		position: absolute;
		inset-inline-start: calc(var(--sp-2) - 1px);
		inset-block: 0;
		border-inline-start: 2px dotted var(--co-divider-4);
		z-index: 0;
	}

	.status {
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

	.status.success {
		background-color: var(--co-success-4);
		border-color: var(--co-success-4);
	}

	.status.failed {
		background-color: var(--co-error-4);
		border-color: var(--co-error-4);
	}

	.status.running {
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
