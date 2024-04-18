<script lang="ts">
	import { goto } from '$app/navigation';
	import Breadcrumb from '$components/breadcrumb.svelte';
	import Button from '$components/button.svelte';
	import routes from '$lib/path';
	import AppForm from '../target-form.svelte';
	import l from '$lib/localization';
	import service, { type CreateTarget } from '$lib/resources/targets';

	const submit = (data: CreateTarget) => service.create(data).then(() => goto(routes.targets));
</script>

<AppForm handler={submit}>
	<Breadcrumb
		slot="default"
		let:submitting
		segments={[
			{ path: routes.targets, title: l.translate('breadcrumb.targets') },
			l.translate('breadcrumb.target.new')
		]}
	>
		<Button type="submit" loading={submitting} text="create" />
	</Breadcrumb>
</AppForm>
