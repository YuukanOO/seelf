<script lang="ts">
	import Checkbox from '$components/checkbox.svelte';
	import FormEnvVars from '$components/form-env-vars.svelte';
	import FormErrors from '$components/form-errors.svelte';
	import FormSection from '$components/form-section.svelte';
	import Form from '$components/form.svelte';
	import Link from '$components/link.svelte';
	import Panel from '$components/panel.svelte';
	import Stack from '$components/stack.svelte';
	import TextInput from '$components/text-input.svelte';
	import {
		toServiceVariablesRecord,
		type CreateAppData,
		type AppDetailData,
		fromServiceVariablesRecord,
		type UpdateAppData
	} from '$lib/resources/apps';
	import l from '$lib/localization';

	export let handler: (data: any) => Promise<unknown>;
	export let initialData: Maybe<AppDetailData> = undefined;
	export let domain: string;
	export let disabled: Maybe<boolean> = undefined;

	const domainUrl = new URL(domain);

	let name = initialData?.name ?? '';
	let production = fromServiceVariablesRecord(initialData?.env?.production);
	let staging = fromServiceVariablesRecord(initialData?.env?.staging);
	let useVCS = !!initialData?.vcs;
	let url = initialData?.vcs?.url ?? '';
	let token = initialData?.vcs?.token;

	$: appName = name || l.translate('app.how.placeholder');

	const environmentText = l.translate('app.how.env');
	const defaultServiceText = l.translate('app.how.default');
	const otherServicesText = l.translate('app.how.others');
	const otherServicesTitleText = l.translate('app.how.others.title');

	// Type $$Props to narrow the handler function based on wether this is an update or a new app
	type $$Props = {
		domain: string;
		/** Force the disabled state of the form in case some other actions is processing */
		disabled?: boolean;
	} & (
		| {
				initialData: AppDetailData;
				handler: (data: UpdateAppData) => Promise<unknown>;
		  }
		| {
				handler: (data: CreateAppData) => Promise<unknown>;
		  }
	);

	async function submit() {
		let env: CreateAppData['env'] = production.length > 0 || staging.length > 0 ? {} : undefined;

		if (production.length > 0) {
			env!['production'] = toServiceVariablesRecord(production);
		}

		if (staging.length > 0) {
			env!['staging'] = toServiceVariablesRecord(staging);
		}

		let formData: any;

		if (!initialData) {
			formData = {
				name: name,
				vcs: useVCS ? { url, token: token ? token : undefined } : undefined,
				env
			} satisfies CreateAppData;
		} else {
			formData = {
				vcs: useVCS
					? {
							url,
							token: token !== initialData.vcs?.token ? (token ? token : null) : undefined
					  }
					: null,
				env: env ?? null
			} satisfies UpdateAppData;
		}

		await handler(formData);
	}
</script>

<Form {disabled} handler={submit} let:submitting let:errors>
	<slot {submitting} />

	<Stack direction="column">
		<FormErrors {errors} />

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
											{domainUrl.protocol}//{appName}.{domainUrl.host}
										</td>
										<td data-label={otherServicesTitleText}>
											{domainUrl.protocol}//dashboard.{appName}.{domainUrl.host}
										</td>
									</tr>
									<tr>
										<td data-label={environmentText}><strong>staging</strong></td>
										<td data-label={defaultServiceText}>
											{domainUrl.protocol}//{appName}-staging.{domainUrl.host}
										</td>
										<td data-label={otherServicesTitleText}>
											{domainUrl.protocol}//dashboard.{appName}-staging.{domainUrl.host}
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
					<Checkbox label="app.vcs.enabled" bind:checked={useVCS}>
						<p>{@html l.translate('app.vcs.help')}</p>
					</Checkbox>
					{#if useVCS}
						<TextInput label="url" required type="url" bind:value={url} />
						<TextInput
							label="app.vcs.token"
							autocomplete="new-password"
							type="password"
							bind:value={token}
						>
							<p>
								{@html l.translate('app.vcs.token.help.instructions')}
								<Link
									external
									newWindow
									href="https://docs.github.com/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token"
									>GitHub</Link
								>
								{l.translate('and')}
								<Link
									external
									newWindow
									href="https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html"
									>GitLab</Link
								>{l.translate('app.vcs.token.help.leave_empty')}
							</p>
						</TextInput>
					{/if}
				</Stack>
			</FormSection>

			<FormSection title="app.environments" transparent>
				<Stack direction="column">
					{#if initialData}
						<Panel variant="help" title="app.environments.help" format="inline">
							<p>{l.translate('app.environments.help.description')}</p>
						</Panel>
					{/if}
					<FormEnvVars title="production" bind:values={production} />
					<FormEnvVars title="staging" bind:values={staging} />
				</Stack>
			</FormSection>
		</div>
	</Stack>
</Form>
