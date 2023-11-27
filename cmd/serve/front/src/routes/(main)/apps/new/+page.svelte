<script lang="ts">
	import { goto } from '$app/navigation';
	import Breadcrumb from '$components/breadcrumb.svelte';
	import Button from '$components/button.svelte';
	import routes from '$lib/path';
	import service, { type CreateAppData } from '$lib/resources/apps';
	import AppForm from '../app-form.svelte';
	import l from '$lib/localization';

	export let data;

	const submit = (data: CreateAppData) => service.create(data).then((a) => goto(routes.app(a.id)));
</script>

<AppForm handler={submit} domain={data.health.domain}>
	<Breadcrumb
		slot="default"
		let:submitting
		segments={[
			{ path: routes.apps, title: l.translate('breadcrumb.applications') },
			l.translate('breadcrumb.application.new')
		]}
	>
		<Button type="submit" loading={submitting} text="create" />
	</Breadcrumb>
</AppForm>
