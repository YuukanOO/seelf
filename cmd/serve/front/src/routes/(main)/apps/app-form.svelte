<script lang="ts">
	import Checkbox from '$components/checkbox.svelte';
	import FormEnvVars from '$components/form-env-vars.svelte';
	import FormErrors from '$components/form-errors.svelte';
	import FormSection from '$components/form-section.svelte';
	import Form from '$components/form.svelte';
	import Panel from '$components/panel.svelte';
	import Stack from '$components/stack.svelte';
	import TextInput from '$components/text-input.svelte';
	import {
		toServiceVariablesRecord,
		type CreateApp,
		type AppDetail,
		fromServiceVariablesRecord,
		type UpdateApp,
		type CreateAppDataEnvironmentConfig
	} from '$lib/resources/apps';
	import type { Target } from '$lib/resources/targets';
	import l from '$lib/localization';
	import type { DropdownOption } from '$components/dropdown.svelte';
	import ServiceUrl from './service-url.svelte';
	import EnvironmentConfigSection from './environment-config-section.svelte';

	export let handler: (data: any) => Promise<unknown>;
	export let targets: Target[];
	export let initialData: Maybe<AppDetail> = undefined;
	export let disabled: Maybe<boolean> = undefined;

	let name = initialData?.name ?? '';
	let production = {
		target: initialData?.production.target.id ?? targets[0]?.id,
		vars: fromServiceVariablesRecord(initialData?.production?.vars)
	};
	let staging = {
		target: initialData?.staging.target.id ?? targets[0]?.id,
		vars: fromServiceVariablesRecord(initialData?.staging?.vars)
	};

	let useVersionControl = !!initialData?.version_control;
	let url = initialData?.version_control?.url ?? '';
	let token = initialData?.version_control?.token;

	let prodScheme = l.translate('app.how.placeholder.scheme');
	let prodUrl = l.translate('app.how.placeholder.url');
	let stagingScheme = prodScheme;
	let stagingUrl = prodUrl;

	const targetsMap = targets.reduce<Record<string, Target>>((acc, value) => {
		acc[value.id] = value;
		return acc;
	}, {});

	$: appName = name || l.translate('app.how.placeholder.name');
	$: {
		try {
			const u = new URL(targetsMap[production.target]?.url!);

			prodScheme = u.protocol + '//';
			prodUrl = u.hostname;
		} catch {
			prodScheme = prodUrl = '';
		}
	}
	$: {
		try {
			const u = new URL(targetsMap[staging.target]?.url!);

			stagingScheme = u.protocol + '//';
			stagingUrl = u.hostname;
		} catch {
			stagingScheme = stagingUrl = '';
		}
	}

	const environmentText = l.translate('app.how.env');
	const defaultServiceText = l.translate('app.how.default');
	const otherServicesText = l.translate('app.how.others');
	const otherServicesTitleText = l.translate('app.how.others.title');

	const targetsOptions = targets.map((target) => ({
		value: target.id,
		label: `${target.url ?? l.translate('target.manual_proxy')} - ${target.name}`
	})) satisfies DropdownOption<string>[];

	// Type $$Props to narrow the handler function based on wether this is an update or a new app
	type $$Props = {
		/** Force the disabled state of the form in case some other actions is processing */
		disabled?: boolean;
		targets: Target[];
	} & (
		| {
				initialData: AppDetail;
				handler: (data: UpdateApp) => Promise<unknown>;
		  }
		| {
				handler: (data: CreateApp) => Promise<unknown>;
		  }
	);

	async function submit() {
		let formData: any;

		const productionData: CreateAppDataEnvironmentConfig = {
			target: production.target,
			vars: production.vars.length ? toServiceVariablesRecord(production.vars) : undefined
		};

		const stagingData: CreateAppDataEnvironmentConfig = {
			target: staging.target,
			vars: staging.vars.length ? toServiceVariablesRecord(staging.vars) : undefined
		};

		if (!initialData) {
			formData = {
				name: name,
				version_control: useVersionControl ? { url, token: token || undefined } : undefined,
				production: productionData,
				staging: stagingData
			} satisfies CreateApp;
		} else {
			formData = {
				version_control: useVersionControl
					? {
							url,
							token: token !== initialData.version_control?.token ? token || null : undefined
					  }
					: null,
				production: productionData,
				staging: stagingData
			} satisfies UpdateApp;
		}

		await handler(formData);
	}
</script>

<Form {disabled} handler={submit} let:submitting let:errors>
	<slot {submitting} />

	<Stack direction="column">
		<FormErrors {errors} />

		{#if targets.length === 0}
			<Panel title="app.no_targets" variant="warning">
				<p>{@html l.translate('app.no_targets.description')}</p>
			</Panel>
		{/if}

		<div>
			{#if !initialData}
				<FormSection title="app.general">
					<Stack direction="column">
						<TextInput
							autofocus={!initialData}
							label="name"
							bind:value={name}
							required
							remoteError={errors?.name}
						>
							<p>{@html l.translate('app.name.help')}</p>
						</TextInput>
						<Panel title="app.how" variant="help" format="collapsable">
							<p>{@html l.translate('app.how.description')}</p>
							<table>
								<thead>
									<tr>
										<th>{environmentText}</th>
										<th>{defaultServiceText}</th>
										<th>{@html otherServicesText}</th>
									</tr>
								</thead>
								<tbody>
									<tr>
										<td data-label={environmentText}><strong>production</strong></td>
										<td data-label={defaultServiceText}>
											<ServiceUrl scheme={prodScheme} host={prodUrl} {appName} />
										</td>
										<td data-label={otherServicesTitleText}>
											<ServiceUrl
												scheme={prodScheme}
												host={prodUrl}
												{appName}
												prefix="dashboard."
											/>
										</td>
									</tr>
									<tr>
										<td data-label={environmentText}><strong>staging</strong></td>
										<td data-label={defaultServiceText}>
											<ServiceUrl
												scheme={stagingScheme}
												host={stagingUrl}
												{appName}
												suffix="-staging"
											/>
										</td>
										<td data-label={otherServicesTitleText}>
											<ServiceUrl
												scheme={stagingScheme}
												host={stagingUrl}
												{appName}
												prefix="dashboard."
												suffix="-staging"
											/>
										</td>
									</tr>
								</tbody>
							</table>
						</Panel>
					</Stack>
				</FormSection>
			{/if}

			<FormSection title="app.vcs">
				<Stack direction="column">
					<Checkbox label="app.vcs.enabled" bind:checked={useVersionControl}>
						<p>{@html l.translate('app.vcs.help')}</p>
					</Checkbox>
					{#if useVersionControl}
						<TextInput label="url" required type="url" bind:value={url} />
						<TextInput
							label="app.vcs.token"
							autocomplete="new-password"
							type="password"
							bind:value={token}
						>
							<p>
								{@html l.translate('app.vcs.token.help')}
							</p>
						</TextInput>
					{/if}
				</Stack>
			</FormSection>

			<EnvironmentConfigSection
				environment="production"
				config={initialData?.production}
				{targetsOptions}
				bind:target={production.target}
				bind:variables={production.vars}
				latestServiceNames={initialData?.latest_deployments.production?.state.services?.map(
					(s) => s.name
				)}
			/>

			<EnvironmentConfigSection
				environment="staging"
				config={initialData?.staging}
				{targetsOptions}
				bind:target={staging.target}
				bind:variables={staging.vars}
				latestServiceNames={initialData?.latest_deployments.staging?.state.services?.map(
					(s) => s.name
				)}
			/>
		</div>
	</Stack>
</Form>
