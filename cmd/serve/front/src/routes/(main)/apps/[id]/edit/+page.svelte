<script lang="ts">
	import Button from '$components/button.svelte';
	import Breadcrumb from '$components/breadcrumb.svelte';
	import type { UpdateAppData } from '$lib/resources/apps';
	import AppForm from '../../app-form.svelte';
	import service from '$lib/resources/apps';
	import { goto } from '$app/navigation';
	import routes from '$lib/path';
	import CleanupNotice from '$components/cleanup-notice.svelte';
	import { submitter } from '$lib/form';
	import FormErrors from '$components/form-errors.svelte';
	import l from '$lib/localization';

	export let data;

	const submit = (form: UpdateAppData) =>
		service.update(data.app.id, form).then(() => goto(routes.app(data.app.id)));

	const {
		loading: deleting,
		errors,
		submit: deleteApp
	} = submitter(() => service.delete(data.app.id).then(() => goto(routes.apps)), {
		confirmation: l.translate('app.delete.confirm', [data.app.name])
	});
</script>

<AppForm disabled={$deleting} handler={submit} initialData={data.app} domain={data.health.domain}>
	<svelte:fragment slot="default" let:submitting>
		<Breadcrumb
			title={l.translate('breadcrumb.application.settings', [data.app.name])}
			segments={[
				{ path: routes.apps, title: l.translate('breadcrumb.applications') },
				{ path: routes.app(data.app.id), title: data.app.name },
				l.translate('breadcrumb.settings')
			]}
		>
			{#if data.app.cleanup_requested_at}
				<CleanupNotice data={data.app} />
			{:else}
				<Button loading={$deleting} on:click={deleteApp} variant="danger" text="app.delete" />
				<Button type="submit" loading={submitting} text="save" />
			{/if}
		</Breadcrumb>
		<FormErrors title="app.delete.failed" class="delete-form-errors" errors={$errors} />
	</svelte:fragment>
</AppForm>

<style module>
	.delete-form-errors {
		margin-block-end: var(--sp-4);
	}
</style>
