<script lang="ts">
	import BlankSlate from '$components/blank-slate.svelte';
	import Breadcrumb from '$components/breadcrumb.svelte';
	import Button from '$components/button.svelte';
	import CardsGrid from '$components/cards-grid.svelte';
	import routes from '$lib/path';
	import service from '$lib/resources/targets';
	import l from '$lib/localization';
	import TargetCard from '$components/target-card.svelte';

	const { data } = service.queryAll();
</script>

<Breadcrumb segments={[l.translate('breadcrumb.targets')]}>
	<Button href={routes.createTarget} text="target.new" />
</Breadcrumb>

{#if $data && $data.length > 0}
	<CardsGrid>
		{#each $data as target (target.id)}
			<TargetCard data={target} />
		{/each}
	</CardsGrid>
{:else}
	<BlankSlate>
		<p>
			{@html l.translate('target.blankslate')}
		</p>
	</BlankSlate>
{/if}
