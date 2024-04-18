<script lang="ts">
	import Button from '$components/button.svelte';
	import { submitter } from '$lib/form';
	import service from '$lib/resources/jobs';
	import Stack from '$components/stack.svelte';
	import l from '$lib/localization';

	export let id: string;
	export let page: number;

	const { submit, loading } = submitter(
		() =>
			service.delete(id).then(() =>
				service.fetchAll(page, {
					cache: 'no-store' // Force the refresh of jobs list
				})
			),
		{
			confirmation: l.translate('jobs.cancel.confirm')
		}
	);
</script>

<Stack direction="row" justify="flex-end">
	<Button variant="danger" text="jobs.cancel" on:click={submit} loading={$loading} />
</Stack>
