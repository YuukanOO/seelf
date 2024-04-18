<script lang="ts">
	import { goto } from '$app/navigation';
	import Breadcrumb from '$components/breadcrumb.svelte';
	import Button from '$components/button.svelte';
	import routes from '$lib/path';
	import TargetForm from '../../target-form.svelte';
	import l from '$lib/localization';
	import service, { TargetStatus, type UpdateTarget } from '$lib/resources/targets';
	import { submitter } from '$lib/form';
	import FormErrors from '$components/form-errors.svelte';
	import Stack from '$components/stack.svelte';
	import CleanupNotice from '$components/cleanup-notice.svelte';

	export let data;

	const submit = (d: UpdateTarget) =>
		service.update(data.target.id, d).then((t) => goto(routes.targets));

	const {
		loading: deleting,
		errors: deleteErr,
		submit: deleteTarget
	} = submitter(() => service.delete(data.target.id).then(() => goto(routes.targets)), {
		confirmation: l.translate(
			data.target.state.status === TargetStatus.Failed && data.target.state.last_ready_version
				? 'target.delete.confirm_failed_status'
				: 'target.delete.confirm',
			[data.target.name]
		)
	});

	const {
		loading: reconfiguring,
		errors: reconfigureErr,
		submit: reconfigureTarget
	} = submitter(() => service.reconfigure(data.target.id).then(() => goto(routes.targets)), {
		confirmation: l.translate('target.reconfigure.confirm')
	});
</script>

<TargetForm disabled={$deleting} initialData={data.target} handler={submit}>
	<svelte:fragment slot="default" let:submitting>
		<Breadcrumb
			title={l.translate('breadcrumb.target.settings', [data.target.name])}
			segments={[
				{ path: routes.targets, title: l.translate('breadcrumb.targets') },
				data.target.name,
				l.translate('breadcrumb.settings')
			]}
		>
			{#if data.target.cleanup_requested_at}
				<CleanupNotice requested_at={data.target.cleanup_requested_at} />
			{:else}
				<Button
					loading={$deleting}
					on:click={deleteTarget}
					disabled={data.target.state.status === TargetStatus.Configuring}
					variant="danger"
					text="target.delete"
				/>
				<Button
					loading={$reconfiguring}
					disabled={data.target.state.status === TargetStatus.Configuring}
					on:click={reconfigureTarget}
					variant="outlined"
					text="target.reconfigure"
				/>
				<Button type="submit" loading={submitting} text="save" />
			{/if}
		</Breadcrumb>
		<Stack direction="column" class="delete-form-errors">
			<FormErrors title="target.delete.failed" errors={$deleteErr} />
			<FormErrors title="target.reconfigure.failed" errors={$reconfigureErr} />
		</Stack>
	</svelte:fragment>
</TargetForm>

<style module>
	.delete-form-errors {
		margin-block-end: var(--sp-4);
	}
</style>
