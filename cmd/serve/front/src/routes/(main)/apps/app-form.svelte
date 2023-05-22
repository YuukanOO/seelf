<script lang="ts">
	import Checkbox from '$components/checkbox.svelte';
	import FormEnvVars from '$components/form-env-vars.svelte';
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

	$: appName = name || '<app-name>';

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

	{#if !initialData}
		<FormSection title="General settings">
			<Stack direction="column">
				<TextInput
					autofocus={!initialData}
					label="Name"
					bind:value={name}
					required
					remoteError={errors.name}
				>
					<p>
						The application name will determine the subdomain from which deployments will be
						available. That's why you should <strong>only</strong> use
						<strong>alphanumeric characters</strong> and a <strong>unique</strong> name accross seelf.
					</p>
				</TextInput>
				<Panel title="How does seelf expose services?" variant="help" format="collapsable">
					<p>
						Services with <strong>port mappings defined</strong> will be exposed with the following convention:
					</p>
					<table>
						<thead>
							<tr>
								<th>Environnment</th>
								<th>Default service (first one in alphabetical order)</th>
								<th>Other exposed services (example: <code>dashboard</code>)</th>
							</tr>
						</thead>
						<tbody>
							<tr>
								<td data-label="Environment"><strong>production</strong></td>
								<td data-label="Default service (first one in alphabetical order)">
									{domainUrl.protocol}//{appName}.{domainUrl.host}
								</td>
								<td data-label="Other exposed services (example: dashboard)">
									{domainUrl.protocol}//dashboard.{appName}.{domainUrl.host}
								</td>
							</tr>
							<tr>
								<td data-label="Environment"><strong>staging</strong></td>
								<td data-label="Default service (first one in alphabetical order)">
									{domainUrl.protocol}//{appName}-staging.{domainUrl.host}
								</td>
								<td data-label="Other exposed services (example: dashboard)">
									{domainUrl.protocol}//dashboard.{appName}-staging.{domainUrl.host}
								</td>
							</tr>
						</tbody>
					</table>
				</Panel>
			</Stack>
		</FormSection>
	{/if}

	<FormSection title="Version control">
		<Stack direction="column">
			<Checkbox label="Use version control system?" bind:checked={useVCS}>
				<p>
					If not under version control, you will still be able to manually deploy your application.
				</p>
			</Checkbox>
			{#if useVCS}
				<TextInput label="Url" required type="url" bind:value={url} />
				<TextInput
					label="Access token"
					autocomplete="new-password"
					type="password"
					bind:value={token}
				>
					<p>
						Token used to fetch the provided repository. Generally known as <strong
							>Personal Access Token</strong
						>, you can find some instructions for <Link
							external
							newWindow
							href="https://docs.github.com/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token"
							>GitHub</Link
						> and <Link
							external
							newWindow
							href="https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html"
							>GitLab</Link
						>, leave empty if the repository is public.
					</p>
				</TextInput>
			{/if}
		</Stack>
	</FormSection>

	<FormSection title="Environment variables" transparent>
		<Stack direction="column">
			{#if initialData}
				<Panel variant="help" title="Note about variables" format="inline">
					<p>
						Updates to your application environment variables will be effective on the next
						deployment.
					</p>
				</Panel>
			{/if}
			<FormEnvVars title="production" bind:values={production} />
			<FormEnvVars title="staging" bind:values={staging} />
		</Stack>
	</FormSection>
</Form>
