<script lang="ts">
	import { DeploymentStatus, type Deployment } from '$lib/resources/deployments';
	import Loader from '$components/loader.svelte';
	import Stack from '$components/stack.svelte';
	import DeploymentPill from '$components/deployment-pill.svelte';
	import StatusIndicator from './status-indicator.svelte';
	import l from '$lib/localization';
	import select from '$lib/select';
	import type { ComponentProps } from 'svelte';

	export let data: Maybe<Deployment[]> = undefined;
	export let variant: 'env' | 'detail';

	/** Retrieve the metadata information based on the deployment status */
	function metadataFromStatus(depl: Deployment): string {
		switch (depl.state.status) {
			case DeploymentStatus.Succeeded:
				return `${l.datetime(depl.state.finished_at!)} (${l.duration(
					depl.state.started_at!,
					depl.state.finished_at!
				)})`;
			case DeploymentStatus.Failed:
				return `${depl.state.error_code} (${l.duration(
					depl.state.started_at!,
					depl.state.finished_at!
				)})`;
			case DeploymentStatus.Running:
				return `${l.datetime(depl.state.started_at!)} (${l.duration(
					depl.state.started_at!,
					new Date()
				)})`;
			default:
				return l.datetime(depl.requested_at);
		}
	}

	function stateFromStatus(status: DeploymentStatus) {
		return select<DeploymentStatus, ComponentProps<StatusIndicator>['state']>(status, {
			[DeploymentStatus.Succeeded]: 'success',
			[DeploymentStatus.Failed]: 'failed',
			[DeploymentStatus.Running]: 'running',
			[DeploymentStatus.Pending]: 'pending'
		});
	}
</script>

<div class="container" class:loading={!data}>
	{#if data}
		{#if data.length > 0}
			<ul class="deployments">
				{#each data as depl (depl.deployment_number)}
					<Stack as="li" gap={2} justify="space-between">
						<Stack gap={2} class="hide-overflow">
							<StatusIndicator state={stateFromStatus(depl.state.status)} />
							<div class="hide-overflow">
								<div class="title">
									{variant === 'env' ? depl.environment : l.datetime(depl.requested_at)}
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
			<p class="no-data">{l.translate('app.no_deployments')}</p>
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
		overflow: hidden;
		text-overflow: ellipsis;
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

	.hide-overflow {
		overflow: hidden;
	}
</style>
