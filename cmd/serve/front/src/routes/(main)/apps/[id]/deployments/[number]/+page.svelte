<script lang="ts">
	import { goto } from '$app/navigation';
	import BlankSlate from '$components/blank-slate.svelte';
	import Breadcrumb from '$components/breadcrumb.svelte';
	import Button from '$components/button.svelte';
	import CleanupNotice from '$components/cleanup-notice.svelte';
	import Console from '$components/console.svelte';
	import DeploymentCard from '$components/deployment-card.svelte';
	import Stack from '$components/stack.svelte';
	import { submitter } from '$lib/form';
	import routes from '$lib/path';
	import service from '$lib/resources/deployments';

	export let data;

	$: isStale =
		data.app.environments[data.deployment.environment]?.deployment_number !==
		data.deployment.deployment_number;

	$: pollNeeded = !data.deployment.state.finished_at; // Poll or not based on wether the deployment has ended

	$: ({ data: deployment } = service.queryByAppAndNumber(
		data.app.id,
		data.deployment.deployment_number,
		pollNeeded
	));
	$: ({ data: logs } = service.queryLogs(
		data.app.id,
		data.deployment.deployment_number,
		pollNeeded
	));

	$: {
		// Here we want to stop the polling if the deployment match the page's deployment
		// and it has ended.
		if (
			pollNeeded &&
			$deployment?.deployment_number === data.deployment.deployment_number &&
			$deployment?.state.finished_at
		) {
			pollNeeded = false;
		}
	}

	$: ({ loading: redeploying, submit: redeploy } = submitter(
		() =>
			service
				.redeploy(data.app.id, data.deployment.deployment_number)
				.then((d) => goto(routes.deployment(data.app.id, d.deployment_number))),
		{
			confirmation: `The deployment #${data.deployment.deployment_number} will be redeployed. Latest app environment variables will be used. Do you confirm this action?`
		}
	));

	$: ({ loading: promoting, submit: promote } = submitter(
		() =>
			service
				.promote(data.app.id, data.deployment.deployment_number)
				.then((d) => goto(routes.deployment(data.app.id, d.deployment_number))),
		{
			confirmation: `The deployment #${data.deployment.deployment_number} will be promoted to the production environment. Do you confirm this action?`
		}
	));
</script>

<Breadcrumb
	segments={[
		{ path: routes.apps, title: 'Applications' },
		{ path: routes.app(data.app.id), title: data.app.name },
		`Deployment #${data.deployment.deployment_number}`
	]}
>
	{#if data.app.cleanup_requested_at}
		<CleanupNotice data={data.app} />
	{:else}
		{#if data.deployment.environment !== 'production'}
			<Button loading={$promoting} on:click={promote} variant="outlined">Promote</Button>
		{/if}
		<Button loading={$redeploying} on:click={redeploy} variant="outlined">Redeploy</Button>
	{/if}
</Breadcrumb>

<Stack direction="column">
	{#if $deployment}
		<DeploymentCard {isStale} data={$deployment} />
	{/if}

	{#if $logs}
		<Console title="Deployment logs" data={$logs} />
	{:else}
		<BlankSlate>
			<p>Waiting for logs...</p>
		</BlankSlate>
	{/if}
</Stack>
