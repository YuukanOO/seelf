<script lang="ts">
	import Form from '$components/form.svelte';
	import Stack from '$components/stack.svelte';
	import FormErrors from '$components/form-errors.svelte';
	import l from '$lib/localization';
	import {
		TargetStatus,
		type CreateTarget,
		type ProviderTypes,
		type Target,
		type UpdateTarget
	} from '$lib/resources/targets';
	import FormSection from '$components/form-section.svelte';
	import TextInput from '$components/text-input.svelte';
	import Dropdown, { type DropdownOption } from '$components/dropdown.svelte';
	import TextArea from '$components/text-area.svelte';
	import Checkbox from '$components/checkbox.svelte';
	import Panel from '$components/panel.svelte';

	export let initialData: Maybe<Target> = undefined;
	export let disabled: boolean = false;
	export let handler: (data: any) => Promise<unknown>;

	const providerTypes = ['docker'] satisfies DropdownOption<ProviderTypes>[];

	let name = initialData?.name ?? '';
	let url = initialData?.url ?? '';
	let provider: ProviderTypes = initialData?.provider.kind ?? providerTypes[0];
	let isRemote = !!initialData?.provider.data.host ?? false;

	const docker = { ...initialData?.provider.data };

	type $$Props = {
		disabled?: boolean;
	} & (
		| {
				initialData: Target;
				handler: (data: UpdateTarget) => Promise<unknown>;
		  }
		| {
				handler: (data: CreateTarget) => Promise<unknown>;
		  }
	);

	async function submit() {
		let formData: any;

		if (!initialData) {
			formData = {
				name,
				url,
				docker:
					provider === 'docker'
						? isRemote
							? {
									host: docker?.host || undefined,
									user: docker?.user || undefined,
									port: parseInt(docker?.port?.toString() ?? '') || undefined,
									private_key: docker?.private_key || undefined
							  }
							: {}
						: undefined
			} satisfies CreateTarget;
		} else {
			formData = {
				name,
				url,
				docker:
					provider === 'docker'
						? isRemote
							? {
									host: docker?.host || undefined,
									user: docker?.user || undefined,
									port: parseInt(docker?.port?.toString() ?? '') || undefined,
									private_key:
										docker?.private_key !== initialData?.provider.data.private_key
											? docker?.private_key || null
											: undefined
							  }
							: undefined
						: undefined
			} satisfies UpdateTarget;
		}

		await handler(formData);
	}
</script>

<Form {disabled} handler={submit} let:submitting let:errors>
	<slot {submitting} />

	<Stack direction="column">
		<FormErrors {errors} />

		{#if initialData?.state.status === TargetStatus.Failed}
			<Panel title="target.failed" variant="danger">
				<p>{initialData.state.error_code}</p>
			</Panel>
		{:else if initialData?.state.status === TargetStatus.Configuring}
			<Panel title="target.configuring" variant="warning">
				<p>{l.translate('target.configuring.description')}</p>
			</Panel>
		{:else if initialData?.state.status === TargetStatus.Ready && initialData.state.last_ready_version}
			<Panel title="target.ready" variant="success">
				<p>
					{l.translate('target.ready.description', [initialData.state.last_ready_version])}
				</p>
			</Panel>
		{/if}

		<div>
			<FormSection title="target.general">
				<Stack direction="column">
					<TextInput label="name" bind:value={name} required remoteError={errors?.name}>
						<p>{l.translate('target.name.help')}</p>
					</TextInput>
					<TextInput label="url" bind:value={url} required type="url" remoteError={errors?.url}>
						<p>{@html l.translate('target.url.help')}</p>
					</TextInput>
				</Stack>
			</FormSection>

			<FormSection title="target.provider">
				<Stack direction="column">
					<Dropdown
						label="target.provider"
						disabled={!!initialData}
						bind:value={provider}
						options={providerTypes}
						remoteError={isRemote ? undefined : errors?.docker}
					/>

					{#if provider === 'docker'}
						<Checkbox
							label="target.docker.is_remote"
							disabled={!!initialData}
							bind:checked={isRemote}
						>
							<p>{@html l.translate('target.docker.is_remote.help')}</p>
						</Checkbox>

						{#if isRemote}
							<TextInput
								label="target.docker.host"
								bind:value={docker.host}
								disabled={!!initialData}
								required
								remoteError={errors?.docker ?? errors?.['docker.host']}
							>
								<p>{@html l.translate('target.docker.host.help')}</p>
							</TextInput>
							<TextInput
								label="target.docker.user"
								bind:value={docker.user}
								placeholder="docker"
								remoteError={errors?.['docker.user']}
							/>
							<TextInput
								label="target.docker.port"
								bind:value={docker.port}
								type="number"
								placeholder="22"
								remoteError={errors?.['docker.port']}
							/>
							<TextArea
								label="target.docker.private_key"
								bind:value={docker.private_key}
								rows={7}
								remoteError={errors?.['docker.private_key']}
							>
								<p>{@html l.translate('target.docker.private_key.help')}</p>
							</TextArea>
						{/if}
					{/if}
				</Stack>
			</FormSection>
		</div>
	</Stack>
</Form>
