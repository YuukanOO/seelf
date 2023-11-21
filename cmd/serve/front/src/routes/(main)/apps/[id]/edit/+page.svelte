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

	export let data;

	const submit = (form: UpdateAppData) =>
		service.update(data.app.id, form).then(() => goto(routes.app(data.app.id)));

	const {
		loading: deleting,
		errors,
		submit: deleteApp
	} = submitter(() => service.delete(data.app.id).then(() => goto(routes.apps)), {
		confirmation: `Are you sure you want to delete the application ${data.app.name}?

This action is IRREVERSIBLE and will DELETE ALL DATA associated with this application: containers, images, volumes, logs and networks.`
	});
</script>

<AppForm disabled={$deleting} handler={submit} initialData={data.app} domain={data.health.domain}>
	<svelte:fragment slot="default" let:submitting>
		<Breadcrumb
			title={`${data.app.name} settings`}
			segments={[
				{ path: routes.apps, title: 'Applications' },
				{ path: routes.app(data.app.id), title: data.app.name },
				'Settings'
			]}
		>
			{#if data.app.cleanup_requested_at}
				<CleanupNotice data={data.app} />
			{:else}
				<Button loading={$deleting} on:click={deleteApp} variant="danger">Delete application</Button
				>
				<Button type="submit" loading={submitting}>Save</Button>
			{/if}
		</Breadcrumb>
		<FormErrors title="Deletion failed" class="delete-form-errors" errors={$errors} />
	</svelte:fragment>
</AppForm>

<style module>
	.delete-form-errors {
		margin-block-end: var(--sp-4);
	}
</style>
