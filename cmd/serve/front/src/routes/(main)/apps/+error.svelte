<script lang="ts">
	import BlankSlate from '$components/blank-slate.svelte';
	import Breadcrumb from '$components/breadcrumb.svelte';
	import Link from '$components/link.svelte';
	import routes from '$lib/path';
</script>

<Breadcrumb segments={[{ path: routes.apps, title: 'Applications' }, 'Not found']} />

<BlankSlate>
	<p>
		Looks like the application you're looking for does not exist. Head back to the <Link
			href={routes.apps}>homepage</Link
		>.
	</p>
</BlankSlate>
