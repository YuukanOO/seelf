<script lang="ts">
	import Panel from '$components/panel.svelte';
	import l from '$lib/localization';

	export let requested_at: Maybe<string>;
</script>

{#if requested_at}
	<Panel title="cleanup_requested" variant="danger" format="inline">
		<p>{l.translate('cleanup_requested.description', [requested_at])}</p>
	</Panel>
{/if}
