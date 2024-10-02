<script lang="ts">
	import Button from '$components/button.svelte';
	import Stack from '$components/stack.svelte';
	import { submitter } from '$lib/form';
	import service from '$lib/resources/jobs';
	import l from '$lib/localization';

	export let id: string;
	export let page: number; // Page number, used to refresh the jobs list
	export let mode: 'dismiss' | 'retry';

	const { submit, loading } = submitter(
		() =>
			service[mode](id).then(() =>
				service.fetchAll(page, {
					cache: 'no-store' // Force the refresh of jobs list
				})
			),
		{
			confirmation: l.translate(`jobs.${mode}.confirm`)
		}
	);
</script>

<Stack direction="row" justify="flex-end">
	<Button
		variant={mode == 'dismiss' ? 'danger' : 'outlined'}
		text={`jobs.${mode}`}
		on:click={submit}
		loading={$loading}
	/>
</Stack>
