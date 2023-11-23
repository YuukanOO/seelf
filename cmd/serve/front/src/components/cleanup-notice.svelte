<script lang="ts">
	import type { AppData } from '$lib/resources/apps';
	import Panel from '$components/panel.svelte';
	import l from '$lib/localization';

	export let data: Maybe<AppData>;
</script>

{#if data?.cleanup_requested_at}
	<Panel title="app.cleanup_requested" variant="danger" format="inline">
		<p>{l.translate('app.cleanup_requested.description', [data.cleanup_requested_at])}</p>
	</Panel>
{/if}
