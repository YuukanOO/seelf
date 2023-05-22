<script lang="ts">
	import { type DeploymentData, DeploymentStatus } from '$lib/resources/deployments';
	import type { ComponentProps } from 'svelte';
	import { formatDatetime, formatDuration } from '$lib/date';
	import Card from '$components/card.svelte';
	import Link from '$components/link.svelte';
	import Stack from '$components/stack.svelte';
	import DeploymentPill from '$components/deployment-pill.svelte';
	import Display from '$components/display.svelte';
	import ExternalLaunch from '$assets/icons/external-launch.svelte';
	import routes from '$lib/path';

	export let data: DeploymentData;

	function colorForStatus(status: DeploymentStatus): ComponentProps<Card>['color'] {
		switch (status) {
			case DeploymentStatus.Running:
				return 'running';
			case DeploymentStatus.Succeeded:
				return 'success';
			case DeploymentStatus.Failed:
				return 'error';
			default:
				return 'pending';
		}
	}
</script>

<Card hasFooter={!!$$slots.default} color={colorForStatus(data.state.status)}>
	<Stack justify="space-between">
		<h2 class="title">{data.environment}</h2>
		<DeploymentPill {data} />
	</Stack>

	<div class="grid">
		<Display label="started at">
			{data.state.started_at ? formatDatetime(data.state.started_at) : '-'}
		</Display>
		<Display label="finished at">
			{data.state.finished_at ? formatDatetime(data.state.finished_at) : '-'}
		</Display>
		<Display label="queued at">
			{formatDatetime(data.requested_at)}
		</Display>
		<Display label="duration">
			{data.state.started_at
				? formatDuration(data.state.started_at, data.state.finished_at ?? new Date())
				: '-'}
		</Display>

		{#if data.meta.kind === 'git'}
			{@const [branch, commit] = data.meta.data.split('@', 2)}
			<Display label="branch">
				{branch}
			</Display>
			<Display label="commit">
				<abbr title={commit}>{commit.substring(0, 10)}</abbr>
			</Display>
		{/if}

		{#if data.state.error_code}
			<Display class="large" label="error code">
				<Link href={routes.deployment(data.app_id, data.deployment_number)}
					>{data.state.error_code}</Link
				>
			</Display>
		{/if}
		{#if data.state.services && data.state.services?.length > 0}
			<Display class="large" label="deployed services">
				<ul>
					{#each data.state.services as service}
						<Stack as="li" justify="space-between">
							{#if service.url}
								<Link class="service-url" href={service.url} external newWindow
									>{service.name} <ExternalLaunch /></Link
								>
							{:else}
								<div>{service.name}</div>
							{/if}
							<div class="service-image">{service.image}</div>
						</Stack>
					{/each}
				</ul>
			</Display>
		{/if}
	</div>

	<slot slot="footer" />
</Card>

<style module>
	.title {
		font: var(--ty-heading-2);
		color: var(--co-text-5);
	}

	.grid {
		display: grid;
		margin-block-start: var(--sp-4);
		gap: var(--sp-2);
		grid-template-columns: 1fr 1fr;
	}

	.large {
		grid-column: span 2;
	}

	.service-url {
		display: flex;
		align-items: center;
		gap: var(--sp-1);
	}

	.service-url svg {
		height: 1rem;
		width: 1rem;
	}

	.service-image {
		background-color: var(--co-background-5);
		border-radius: var(--ra-4);
		border: 1px solid var(--co-divider-4);
		font: var(--ty-caption);
		color: var(--co-text-4);
		padding: 0.125rem var(--sp-2);
		white-space: nowrap;
		text-overflow: ellipsis;
		overflow: hidden;
	}
</style>
