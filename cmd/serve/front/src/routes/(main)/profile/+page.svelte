<script lang="ts">
	import Breadcrumb from '$components/breadcrumb.svelte';
	import Button from '$components/button.svelte';
	import FormErrors from '$components/form-errors.svelte';
	import FormSection from '$components/form-section.svelte';
	import Form from '$components/form.svelte';
	import Panel from '$components/panel.svelte';
	import Stack from '$components/stack.svelte';
	import TextArea from '$components/text-area.svelte';
	import TextInput from '$components/text-input.svelte';
	import Dropdown from '$components/dropdown.svelte';
	import { submitter } from '$lib/form.js';
	import service from '$lib/resources/users';
	import l from '$lib/localization';

	export let data;

	let email = data.user.email;
	let password: Maybe<string>;
	let apiKey = data.user.api_key;
	let locale = l.locale();

	const locales = l.locales().map((l) => ({ label: l.displayName, value: l.code }));

	const submit = () =>
		service
			.update({
				email,
				password: password ? password : undefined
			})
			.then(() => l.locale(locale));

	const {
		submit: refreshKey,
		loading,
		errors: refreshErr
	} = submitter(() => service.refreshAPIKey().then((d) => (apiKey = d.api_key)), {
		confirmation: l.translate('profile.key.refresh.confirm')
	});
</script>

<Form handler={submit} let:submitting let:errors>
	<Breadcrumb segments={[l.translate('breadcrumb.profile')]}>
		<Button type="submit" loading={submitting} text="save" />
	</Breadcrumb>

	<Stack direction="column">
		<FormErrors {errors} />

		<div>
			<FormSection title="profile.informations">
				<Stack direction="column">
					<TextInput
						label="email"
						type="email"
						required
						bind:value={email}
						remoteError={errors?.email}
					/>
					<TextInput
						label="profile.password"
						type="password"
						autocomplete="new-password"
						bind:value={password}
						remoteError={errors?.password}
					>
						<p>{l.translate('profile.password.help')}</p>
					</TextInput>
				</Stack>
			</FormSection>

			<FormSection title="profile.interface">
				<Dropdown label="profile.locale" options={locales} bind:value={locale} />
			</FormSection>

			<FormSection title="profile.integration">
				<Button
					slot="actions"
					variant="outlined"
					on:click={refreshKey}
					loading={$loading}
					text="profile.key.refresh"
				/>
				<Stack direction="column">
					<FormErrors errors={$refreshErr} />
					<Panel title="profile.integration.title" variant="help">
						<p>{@html l.translate('profile.integration.description')}</p>
					</Panel>
					<TextArea label="profile.key" rows={1} code value={apiKey} readonly>
						<p>
							{@html l.translate('profile.key.help')}
						</p>
					</TextArea>
				</Stack>
			</FormSection>
		</div>
	</Stack>
</Form>
