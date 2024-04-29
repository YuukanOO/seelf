<script lang="ts">
	import service, { type DeploymentDetail } from '$lib/resources/deployments';
	import DeploymentsList from '$components/deployments-list.svelte';
	import DeploymentCard from '$components/deployment-card.svelte';
	import Stack from '$components/stack.svelte';
	import Pagination from '$components/pagination.svelte';

	export let data: DeploymentDetail;

	let page = 1;

	$: ({ data: deployments } = service.queryAllByApp(data.app_id, {
		page,
		environment: data.environment
	}));
</script>

<DeploymentCard {data}>
	<Stack direction="column">
		<DeploymentsList variant="detail" data={$deployments?.data} />
		<Pagination data={$deployments} on:previous={() => page--} on:next={() => page++} />
	</Stack>
</DeploymentCard>
