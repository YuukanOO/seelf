<script lang="ts">
	import { goto } from '$app/navigation';
	import Breadcrumb from '$components/breadcrumb.svelte';
	import Button from '$components/button.svelte';
	import routes from '$lib/path';
	import RegistryForm from '../../registry-form.svelte';
	import l from '$lib/localization';
	import service, { type UpdateRegistry } from '$lib/resources/registries';
	import { submitter } from '$lib/form';
	import FormErrors from '$components/form-errors.svelte';
	import Stack from '$components/stack.svelte';

	export let data;

	const submit = (d: UpdateRegistry) =>
		service.update(data.registry.id, d).then((t) => goto(routes.registries));

	const {
		loading: deleting,
		errors: deleteErr,
		submit: deleteRegistry
	} = submitter(() => service.delete(data.registry.id).then(() => goto(routes.registries)), {
		confirmation: l.translate('registry.delete.confirm', [data.registry.name])
	});
</script>

<RegistryForm disabled={$deleting} initialData={data.registry} handler={submit}>
	<svelte:fragment slot="default" let:submitting>
		<Breadcrumb
			title={l.translate('breadcrumb.registry.settings', [data.registry.name])}
			segments={[
				{ path: routes.registries, title: l.translate('breadcrumb.registries') },
				data.registry.name,
				l.translate('breadcrumb.settings')
			]}
		>
			<Button
				loading={$deleting}
				on:click={deleteRegistry}
				variant="danger"
				text="registry.delete"
			/>
			<Button type="submit" loading={submitting} text="save" />
		</Breadcrumb>
		<Stack direction="column" class="delete-form-errors">
			<FormErrors title="registry.delete.failed" errors={$deleteErr} />
		</Stack>
	</svelte:fragment>
</RegistryForm>

<style module>
	.delete-form-errors {
		margin-block-end: var(--sp-4);
	}
</style>
