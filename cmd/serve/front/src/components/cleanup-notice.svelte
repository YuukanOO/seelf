<script lang="ts">
	import type { AppData } from '$lib/resources/apps';
	import { formatDate } from '$lib/date';
	import Panel from '$components/panel.svelte';

	export let data: Maybe<AppData>;
</script>

{#if data?.cleanup_requested_at}
	<Panel title="Marked for deletion" variant="danger" format="inline">
		<p>
			The application removal has been requested at {formatDate(data.cleanup_requested_at)} and will
			be processed shortly.
		</p>
	</Panel>
{/if}
