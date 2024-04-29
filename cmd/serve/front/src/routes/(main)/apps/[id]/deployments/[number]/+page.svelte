<script lang="ts">
	import { goto } from '$app/navigation';
	import BlankSlate from '$components/blank-slate.svelte';
	import Breadcrumb from '$components/breadcrumb.svelte';
	import Button from '$components/button.svelte';
	import CleanupNotice from '$components/cleanup-notice.svelte';
	import Console from '$components/console.svelte';
	import DeploymentCard from '$components/deployment-card.svelte';
	import FormErrors from '$components/form-errors.svelte';
	import Stack from '$components/stack.svelte';
	import { submitter } from '$lib/form';
	import routes from '$lib/path';
	import service from '$lib/resources/deployments';
	import { TargetStatus } from '$lib/resources/targets';
	import l from '$lib/localization';

	export let data;

	$: isStale =
		data.app.latest_deployments[data.deployment.environment]?.deployment_number !==
		data.deployment.deployment_number;

	$: latestUrl = isStale
		? routes.deployment(
				data.app.id,
				data.app.latest_deployments[data.deployment.environment]!.deployment_number
		  )
		: undefined;

	$: pollNeeded =
		!data.deployment.state.finished_at ||
		data.deployment.target.status === TargetStatus.Configuring; // Poll or not based on wether the deployment has ended or the target is configuring

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
			$deployment?.state.finished_at &&
			$deployment?.target.status !== TargetStatus.Configuring
		) {
			pollNeeded = false;
		}
	}

	$: ({
		loading: redeploying,
		errors: redeployErr,
		submit: redeploy
	} = submitter(
		() =>
			service
				.redeploy(data.app.id, data.deployment.deployment_number)
				.then((d) => goto(routes.deployment(data.app.id, d.deployment_number))),
		{
			confirmation: l.translate('deployment.redeploy.confirm', [data.deployment.deployment_number])
		}
	));

	$: ({
		loading: promoting,
		errors: promoteErr,
		submit: promote
	} = submitter(
		() =>
			service
				.promote(data.app.id, data.deployment.deployment_number)
				.then((d) => goto(routes.deployment(data.app.id, d.deployment_number))),
		{
			confirmation: l.translate('deployment.promote.confirm', [data.deployment.deployment_number])
		}
	));
</script>

<Breadcrumb
	segments={[
		{ path: routes.apps, title: l.translate('breadcrumb.applications') },
		{ path: routes.app(data.app.id), title: data.app.name },
		l.translate('breadcrumb.deployment.name', [data.deployment.deployment_number])
	]}
>
	{#if data.app.cleanup_requested_at}
		<CleanupNotice requested_at={data.app.cleanup_requested_at} />
	{:else}
		{#if data.deployment.environment !== 'production'}
			<Button
				loading={$promoting}
				on:click={promote}
				variant="outlined"
				text="deployment.promote"
			/>
		{/if}
		<Button
			loading={$redeploying}
			on:click={redeploy}
			variant="outlined"
			text="deployment.redeploy"
		/>
	{/if}
</Breadcrumb>

<Stack direction="column">
	<FormErrors errors={$redeployErr} title="deployment.redeploy.failed" />
	<FormErrors errors={$promoteErr} title="deployment.promote.failed" />

	{#if $deployment}
		<DeploymentCard {latestUrl} data={$deployment} />
	{/if}

	{#if $logs}
		<Console title="deployment.logs" data={$logs} />
	{:else}
		<BlankSlate>
			<p>{l.translate('deployment.waiting_for_logs')}</p>
		</BlankSlate>
	{/if}
</Stack>
