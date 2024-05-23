<script lang="ts">
	import Form from '$components/form.svelte';
	import Stack from '$components/stack.svelte';
	import FormErrors from '$components/form-errors.svelte';
	import l from '$lib/localization';
	import FormSection from '$components/form-section.svelte';
	import TextInput from '$components/text-input.svelte';
	import Checkbox from '$components/checkbox.svelte';
	import type { CreateRegistry, Registry, UpdateRegistry } from '$lib/resources/registries';

	export let initialData: Maybe<Registry> = undefined;
	export let disabled: boolean = false;
	export let handler: (data: any) => Promise<unknown>;

	let name = initialData?.name ?? '';
	let url = initialData?.url ?? '';
	let username = initialData?.credentials?.username ?? '';
	let password = initialData?.credentials?.password ?? '';
	let useCredentials = !!initialData?.credentials;

	type $$Props = {
		disabled?: boolean;
	} & (
		| {
				initialData: Registry;
				handler: (data: UpdateRegistry) => Promise<unknown>;
		  }
		| {
				handler: (data: CreateRegistry) => Promise<unknown>;
		  }
	);

	async function submit() {
		let formData: any;

		if (!initialData) {
			formData = {
				name,
				url,
				credentials: useCredentials
					? {
							username,
							password
					  }
					: undefined
			} satisfies CreateRegistry;
		} else {
			formData = {
				name,
				url,
				credentials: useCredentials
					? {
							username,
							password: password !== initialData.credentials?.password ? password : undefined
					  }
					: null
			} satisfies UpdateRegistry;
		}

		await handler(formData);
	}
</script>

<Form {disabled} handler={submit} let:submitting let:errors>
	<slot {submitting} />

	<Stack direction="column">
		<FormErrors {errors} />

		<div>
			<FormSection title="registry.general">
				<Stack direction="column">
					<TextInput autofocus label="name" bind:value={name} required remoteError={errors?.name}>
						<p>{l.translate('registry.name.help')}</p>
					</TextInput>

					<TextInput label="url" type="url" bind:value={url} required remoteError={errors?.url}>
						<p>{@html l.translate('registry.url.help')}</p>
					</TextInput>
				</Stack>
			</FormSection>
			<FormSection title="registry.authentication">
				<Stack direction="column">
					<Checkbox label="registry.auth" bind:checked={useCredentials}>
						<p>{@html l.translate('registry.auth.help')}</p>
					</Checkbox>

					{#if useCredentials}
						<TextInput
							label="registry.username"
							bind:value={username}
							required
							remoteError={errors?.['credentials.username']}
						/>
						<TextInput
							label="registry.password"
							type="password"
							autocomplete="new-password"
							bind:value={password}
							required
							remoteError={errors?.['credentials.password']}
						/>
					{/if}
				</Stack>
			</FormSection>
		</div>
	</Stack>
</Form>
