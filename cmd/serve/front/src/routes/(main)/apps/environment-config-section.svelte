<script lang="ts">
	import FormEnvVars from '$components/form-env-vars.svelte';
	import FormSection from '$components/form-section.svelte';
	import Panel from '$components/panel.svelte';
	import Stack from '$components/stack.svelte';
	import Dropdown, { type DropdownOption } from '$components/dropdown.svelte';
	import l from '$lib/localization';
	import type { Environment, ServiceVariables } from '$lib/resources/apps';

	export let target: string;
	export let variables: ServiceVariables[];

	export let environment: 'production' | 'staging';
	export let latestServiceNames: Maybe<string[]> = undefined;
	export let config: Maybe<Environment>;
	export let targetsOptions: DropdownOption<string>[];
	export let errors: Maybe<Record<string, string>> = undefined;

	const isMigrating = !!config?.migration?.id;
</script>

<FormSection title={`app.environment.${environment}`}>
	<Stack direction="column">
		<Dropdown
			label="app.environment.target"
			options={targetsOptions}
			bind:value={target}
			disabled={isMigrating}
			remoteError={errors?.[`${environment}.target`]}
		/>

		{#if isMigrating}
			<Panel title="app.environment.target.migrating" variant="help">
				<p>
					{@html l.translate('app.environment.target.migrating.description', [
						config?.migration?.name ?? ''
					])}
				</p>
			</Panel>
		{/if}

		{#if config && config.target.id !== target}
			<Panel title="app.environment.target.changed" variant="warning">
				<p>
					{@html l.translate('app.environment.target.changed.description', [config.target.name])}
				</p>
			</Panel>
		{/if}
	</Stack>
</FormSection>

<FormEnvVars class="env-vars-container" {latestServiceNames} bind:values={variables} />

<style module>
	.env-vars-container {
		margin-block-start: var(--sp-4);
		margin-block-end: var(--sp-6);
	}
</style>
