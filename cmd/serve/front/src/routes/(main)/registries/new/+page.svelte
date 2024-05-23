<script lang="ts">
	import { goto } from '$app/navigation';
	import Breadcrumb from '$components/breadcrumb.svelte';
	import Button from '$components/button.svelte';
	import routes from '$lib/path';
	import RegistryForm from '../registry-form.svelte';
	import l from '$lib/localization';
	import service, { type CreateRegistry } from '$lib/resources/registries';

	const submit = (data: CreateRegistry) => service.create(data).then(() => goto(routes.registries));
</script>

<RegistryForm handler={submit}>
	<Breadcrumb
		slot="default"
		let:submitting
		segments={[
			{ path: routes.registries, title: l.translate('breadcrumb.registries') },
			l.translate('breadcrumb.registry.new')
		]}
	>
		<Button type="submit" loading={submitting} text="create" />
	</Breadcrumb>
</RegistryForm>
