<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import Stack from '$components/stack.svelte';
	import ButtonBase from '$components/button-base.svelte';
	import type { Paginated } from '$lib/pagination';
	import ArrowRight from '$assets/icons/arrow-right.svelte';
	import ArrowLeft from '$assets/icons/arrow-left.svelte';
	import l from '$lib/localization';

	const dispatcher = createEventDispatcher();

	export let data: Maybe<Paginated<unknown>> = undefined;

	$: numberOfPages = data ? Math.ceil(data.total / data.per_page) : 0;
</script>

{#if data && numberOfPages > 1}
	<Stack justify="space-between">
		<div class="page">{l.translate('page_n_of_m', [data.page, numberOfPages])}</div>
		<Stack class="pagination" gap={0}>
			<ButtonBase
				title="previous"
				disabled={data.first_page}
				variant="outlined"
				on:click={() => dispatcher('previous')}
			>
				<ArrowLeft />
			</ButtonBase>
			<ButtonBase
				title="next"
				disabled={data.last_page}
				variant="outlined"
				on:click={() => dispatcher('next')}
			>
				<ArrowRight />
			</ButtonBase>
		</Stack>
	</Stack>
{/if}

<style module>
	.page {
		font: var(--ty-caption);
	}

	.pagination > *:first-child {
		border-start-end-radius: 0;
		border-end-end-radius: 0;
	}

	.pagination > *:last-child {
		border-start-start-radius: 0;
		border-end-start-radius: 0;
	}
</style>
