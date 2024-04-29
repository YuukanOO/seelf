<script lang="ts">
	import { type DeploymentDetail, DeploymentStatus } from '$lib/resources/deployments';
	import type { ComponentProps } from 'svelte';
	import routes from '$lib/path';
	import Card from '$components/card.svelte';
	import Link from '$components/link.svelte';
	import Stack from '$components/stack.svelte';
	import Panel from '$components/panel.svelte';
	import DeploymentPill from '$components/deployment-pill.svelte';
	import Display from '$components/display.svelte';
	import ServiceInfo from '$components/service-info.svelte';
	import l from '$lib/localization';

	export let data: DeploymentDetail;
	export let latestUrl: Maybe<string> = undefined; // If set, show the oudated panel

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

	<dl class="grid">
		<Display label="deployment.started_at">
			{data.state.started_at ? l.datetime(data.state.started_at) : '-'}
		</Display>
		<Display label="deployment.finished_at">
			{data.state.finished_at ? l.datetime(data.state.finished_at) : '-'}
		</Display>
		<Display label="deployment.queued_at">
			{l.datetime(data.requested_at)}
		</Display>
		<Display label="deployment.duration">
			{data.state.started_at
				? l.duration(data.state.started_at, data.state.finished_at ?? new Date())
				: '-'}
		</Display>

		<Display class="large" label="deployment.target">
			{#if data.target.name}
				<Link href={routes.editTarget(data.target.id)}>{data.target.name}</Link>
			{:else}
				{l.translate('deployment.target.deleted')}
			{/if}
		</Display>

		{#if data.source.discriminator === 'git'}
			<Display label="deployment.branch">
				{data.source.data.branch}
			</Display>
			<Display label="deployment.commit">
				<abbr title={data.source.data.hash}>{data.source.data.hash.substring(0, 10)}</abbr>
			</Display>
		{/if}

		{#if data.state.error_code}
			<Display class="large" label="deployment.error_code">
				<Link href={routes.deployment(data.app_id, data.deployment_number)}>
					{data.state.error_code}
				</Link>
			</Display>
		{/if}
		{#if data.state.services && data.state.services?.length > 0}
			<Display class="large" label="deployment.services">
				{#if latestUrl}
					<Panel class="outdated" title="deployment.outdated" format="inline" variant="warning">
						<p>{@html l.translate('deployment.outdated.description', [latestUrl])}</p>
					</Panel>
				{/if}
				<ul class="services">
					{#each data.state.services as service (service.name)}
						<li class="service">
							<ServiceInfo data={service} />
						</li>
					{/each}
				</ul>
			</Display>
		{/if}
	</dl>

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
		grid-template-columns: repeat(2, 1fr);
	}

	.large {
		grid-column: span 2;
	}

	.services {
		margin-block-start: var(--sp-1);
	}

	.service + .service {
		margin-block-start: var(--sp-2);
	}

	.outdated {
		margin-block: var(--sp-2);
		margin-inline: 1px;
	}
</style>
