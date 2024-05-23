<script lang="ts">
	import BlankSlate from '$components/blank-slate.svelte';
	import Breadcrumb from '$components/breadcrumb.svelte';
	import Button from '$components/button.svelte';
	import CardsGrid from '$components/cards-grid.svelte';
	import RegistryCard from '$components/registry-card.svelte';
	import routes from '$lib/path';
	import service from '$lib/resources/registries';
	import l from '$lib/localization';

	const { data } = service.queryAll();
</script>

<Breadcrumb segments={[l.translate('breadcrumb.registries')]}>
	<Button href={routes.createRegistry} text="registry.new" />
</Breadcrumb>

{#if $data && $data.length > 0}
	<CardsGrid>
		{#each $data as registry (registry.id)}
			<RegistryCard data={registry} />
		{/each}
	</CardsGrid>
{:else}
	<BlankSlate>
		<p>
			{@html l.translate('registry.blankslate')}
		</p>
	</BlankSlate>
{/if}
