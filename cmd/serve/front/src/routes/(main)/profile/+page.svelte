<script lang="ts">
	import Breadcrumb from '$components/breadcrumb.svelte';
	import Button from '$components/button.svelte';
	import FormSection from '$components/form-section.svelte';
	import Form from '$components/form.svelte';
	import Panel from '$components/panel.svelte';
	import Stack from '$components/stack.svelte';
	import TextArea from '$components/text-area.svelte';
	import TextInput from '$components/text-input.svelte';
	import service from '$lib/resources/users.js';

	export let data;

	let email = data.user.email;
	let password: Maybe<string>;

	const submit = () =>
		service.update({
			email,
			password: password ? password : undefined
		});
</script>

<Form handler={submit} let:submitting let:errors>
	<Breadcrumb segments={['Profile']}>
		<Button type="submit" loading={submitting}>Save</Button>
	</Breadcrumb>

	<FormSection title="User information">
		<Stack direction="column">
			<TextInput label="Email" type="email" bind:value={email} remoteError={errors.email} />
			<TextInput
				label="New password"
				type="password"
				autocomplete="new-password"
				bind:value={password}
				remoteError={errors.password}
			>
				<p>Leave empty if you don't want to change your password.</p>
			</TextInput>
		</Stack>
	</FormSection>

	<FormSection title="Integration">
		<Stack direction="column">
			<Panel title="Integration with CI" variant="help">
				<p>
					If you want to trigger a deployment for an application, you'll need this token. You can
					also click the <strong>Copy cURL command</strong> from the deployment page and use it in your
					pipeline since it includes the token in the appropriate header.
				</p>
			</Panel>
			<TextArea label="API Key" rows={1} code value={data.user.api_key} readonly>
				<p>
					Pass this token as an <code>Authorization: Bearer</code> header to communicate with the
					seelf API. <strong>You MUST keep it secret!</strong>
				</p>
			</TextArea>
		</Stack>
	</FormSection>
</Form>
