<script lang="ts">
	let className: string = '';

	/** Additional css classes */
	export { className as class };

	export let as = 'div';
</script>

<svelte:element this={as} class="prose {className}">
	<slot />
</svelte:element>

<style module>
	.prose a {
		color: var(--co-primary-4);
	}

	.prose strong {
		color: var(--co-text-5);
		font-weight: 600;
	}

	.prose code {
		font-family: var(--fo-mono);
	}

	.prose > * + * {
		margin-block-start: var(--sp-4);
	}

	table {
		width: 100%;
	}

	table thead {
		display: none;
	}

	table thead th {
		border-block-end: 1px solid var(--co-divider-4);
		padding-inline: var(--sp-1);
		padding-block-end: var(--sp-1);
		font-style: italic;
	}

	table tbody td {
		display: block;
		padding-block: var(--sp-1);
	}

	table tbody td::before {
		content: attr(data-label);
		display: block;
		font-style: italic;
	}

	table tbody tr {
		display: block;
	}

	table tbody tr + tr {
		border-block-start: 1px solid var(--co-divider-4);
		margin-block-start: var(--sp-2);
		padding-block-start: var(--sp-2);
	}

	@media screen and (min-width: 56rem) {
		table thead {
			display: table-header-group;
		}

		table tbody tr {
			display: table-row;
		}

		table tbody tr + tr {
			margin: 0;
			padding: 0;
		}

		table tbody td {
			display: table-cell;
			padding: var(--sp-2) var(--sp-1);
		}

		table tbody td::before {
			display: none;
		}
	}
</style>
