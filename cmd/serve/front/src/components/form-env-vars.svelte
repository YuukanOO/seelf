<script lang="ts">
	import BlankSlate from '$components/blank-slate.svelte';
	import Button from '$components/button.svelte';
	import Stack from '$components/stack.svelte';
	import TextInput from '$components/text-input.svelte';
	import type { ServiceVariables } from '$lib/resources/apps';
	import TextArea from '$components/text-area.svelte';
	import l from '$lib/localization';

	export let title: string;
	export let values: ServiceVariables[] = [];

	function add() {
		values = [...values, { name: '', values: '' }];
	}

	function remove(idx: number) {
		values = values.filter((_, i) => i !== idx);
	}
</script>

<Stack direction="column">
	<Stack justify="space-between">
		<h3 class="title">{title}</h3>
		<Button on:click={add} text="app.environments.service.add" />
	</Stack>
	{#if values.length === 0}
		<BlankSlate>
			<p>{@html l.translate('app.environments.blankslate', [title])}</p>
		</BlankSlate>
	{:else}
		{#each values as value, i}
			<Stack class="container" direction="column">
				<Stack class="service" wrap="wrap" align="flex-start">
					<TextInput label="app.environments.service.name" bind:value={value.name} required />
					<TextArea
						rows={5}
						code
						label="app.environments.service.env"
						placeholder="KEY=value
SOME=value"
						bind:value={value.values}
					/>
				</Stack>
				<Stack justify="flex-end">
					<Button
						variant="outlined"
						on:click={() => remove(i)}
						text="app.environments.service.delete"
					/>
				</Stack>
			</Stack>
		{/each}
	{/if}
</Stack>

<style module>
	.title {
		font: var(--ty-heading-3);
		color: var(--co-text-5);
	}

	.container {
		background-color: var(--co-background-5);
		padding: var(--sp-4);
		border-radius: var(--ra-4);
	}

	.service > * {
		flex: 1 calc(var(--mi-container-width) / 3);
	}

	@media screen and (min-width: 56rem) {
		.container {
			padding: var(--sp-6);
		}
	}
</style>
