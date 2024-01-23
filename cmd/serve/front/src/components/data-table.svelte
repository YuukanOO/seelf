<script lang="ts">
	import BlankSlate from '$components/blank-slate.svelte';
	import ArrowRight from '$assets/icons/arrow-right.svelte';
	import type { AppTranslationsString } from '$lib/localization';
	import l from '$lib/localization';

	type T = $$Generic<{ id: string }>;

	type Column<T> = {
		label: AppTranslationsString;
		/** If set to a raw string, will be pass down to the slot with the provided value */
		value: string | ((data: T) => string);
	};

	export let data: Maybe<T[]>;
	export let columns: Column<T>[];
</script>

<table class="datatable">
	<thead>
		<tr>
			{#each columns as { label } (label)}
				<th>{l.translate(label)}</th>
			{/each}
			{#if $$slots.expanded}
				<th />
			{/if}
		</tr>
	</thead>
	<tbody>
		{#if !data || data.length === 0}
			<tr>
				<td colspan={columns.length}>
					<BlankSlate>
						<p>{l.translate('datatable.no_data')}</p>
					</BlankSlate>
				</td>
			</tr>
		{:else}
			{#each data as item (item.id)}
				<tr class="datarow">
					{#each columns as { label, value } (label)}
						<td data-label={l.translate(label)}>
							{#if typeof value === 'string'}
								<slot {value} {item} />
							{:else}
								{value(item)}
							{/if}
						</td>
					{/each}
					{#if $$slots.expanded}
						<td class="expand-cell">
							<label class="expand-toggle">
								<input type="checkbox" />
								<span>{l.translate('datatable.toggle')}</span>
								<ArrowRight />
							</label>
						</td>
					{/if}
				</tr>
				{#if $$slots.expanded}
					<tr class="expanded">
						<td colspan={columns.length + 1}>
							<slot name="expanded" {item} />
						</td>
					</tr>
				{/if}
			{/each}
		{/if}
	</tbody>
</table>

<style module>
	.expand-toggle {
		align-items: center;
		cursor: pointer;
		display: flex;
	}

	.expand-toggle span {
		font: var(--ty-caption);
	}

	.expand-toggle svg {
		display: block;
		width: 1rem;
		height: 1rem;
	}

	.expanded {
		display: none;
	}

	.expanded td {
		background-color: var(--co-background-5);
		padding: var(--sp-4);
	}

	.datarow {
		color: var(--co-text-5);
	}

	.datarow:has(input[type='checkbox']:checked) .expand-toggle svg {
		transform: rotate(90deg);
	}

	.datarow:has(input[type='checkbox']:checked) + .expanded {
		display: table-row;
	}

	.datarow td {
		vertical-align: middle;
	}

	@media screen and (min-width: 56rem) {
		.expand-toggle {
			justify-content: flex-end;
			padding: var(--sp-2);
		}

		.expand-toggle span {
			position: absolute;
			width: 1px;
			height: 1px;
			padding: 0;
			margin: -1px;
			overflow: hidden;
			clip: rect(0, 0, 0, 0);
			white-space: nowrap;
			border-width: 0;
		}
	}
</style>
