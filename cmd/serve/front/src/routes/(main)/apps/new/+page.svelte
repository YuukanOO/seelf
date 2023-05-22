<script lang="ts">
	import { goto } from '$app/navigation';
	import Breadcrumb from '$components/breadcrumb.svelte';
	import Button from '$components/button.svelte';
	import routes from '$lib/path';
	import service, { type CreateAppData } from '$lib/resources/apps';
	import AppForm from '../app-form.svelte';

	export let data;

	const submit = (data: CreateAppData) => service.create(data).then((a) => goto(routes.app(a.id)));
</script>

<AppForm handler={submit} domain={data.health.domain}>
	<Breadcrumb
		slot="default"
		let:submitting
		segments={[{ path: routes.apps, title: 'Applications' }, 'New application']}
	>
		<Button type="submit" loading={submitting}>Create</Button>
	</Breadcrumb>
</AppForm>
