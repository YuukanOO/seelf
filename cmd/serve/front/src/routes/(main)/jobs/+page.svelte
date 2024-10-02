<script lang="ts">
	import Breadcrumb from '$components/breadcrumb.svelte';
	import DataTable from '$components/data-table.svelte';
	import StatusIndicator from '$components/status-indicator.svelte';
	import Display from '$components/display.svelte';
	import Pagination from '$components/pagination.svelte';
	import Stack from '$components/stack.svelte';
	import service from '$lib/resources/jobs';
	import l, { type AppTranslationsString } from '$lib/localization';
	import JobActionButton from './job-action-button.svelte';

	let page = 1;

	function translateMessageName(messageName: string) {
		return l.translate(messageName as AppTranslationsString);
	}

	$: ({ data } = service.queryAll(page));
</script>

<Breadcrumb segments={[l.translate('breadcrumb.jobs')]} />

<DataTable
	data={$data?.data}
	columns={[
		{
			label: 'jobs.status',
			value: 'status'
		},
		{
			label: 'jobs.resource',
			value: 'resource'
		},
		{
			label: 'jobs.dates',
			value: 'dates'
		}
	]}
>
	<svelte:fragment let:value let:item>
		{#if value === 'status'}
			<StatusIndicator
				state={item.retrieved ? 'running' : item.error_code ? 'failed' : 'pending'}
			/>
		{:else if value === 'dates'}
			<div>{l.datetime(item.queued_at)}</div>
			<div class="meta">{l.datetime(item.not_before)}</div>
		{:else if value === 'resource'}
			<!-- @ts-ignore -->
			<div>{translateMessageName(item.message_name)}</div>
			<div class="meta">{item.resource_id}</div>
		{/if}
	</svelte:fragment>

	<svelte:fragment slot="expanded" let:item>
		<Stack direction="column">
			<dl class="grid">
				<Display label="jobs.group">
					{item.group}
				</Display>
				<Display label="jobs.payload">
					<code>{item.message_data}</code>
				</Display>
				<Display label="jobs.error">
					{item.error_code ?? '-'}
				</Display>
			</dl>
			{#if item.error_code}
				<Stack justify="flex-end">
					<JobActionButton id={item.id} mode="dismiss" {page} />
					<JobActionButton id={item.id} mode="retry" {page} />
				</Stack>
			{/if}
		</Stack>
	</svelte:fragment>
</DataTable>

<Pagination data={$data} on:previous={() => page--} on:next={() => page++} />

<style module>
	.meta {
		font: var(--ty-caption);
		color: var(--co-text-4);
	}

	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(20rem, 1fr));
		gap: var(--sp-2);
	}

	.grid code {
		font: var(--ty-caption-mono);
		word-break: break-all;
		display: block;
	}
</style>
