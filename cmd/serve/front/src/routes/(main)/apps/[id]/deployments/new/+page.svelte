<script lang="ts">
	import { goto } from '$app/navigation';
	import Breadcrumb from '$components/breadcrumb.svelte';
	import Button from '$components/button.svelte';
	import CleanupNotice from '$components/cleanup-notice.svelte';
	import Console from '$components/console.svelte';
	import Dropdown, { type DropdownOption } from '$components/dropdown.svelte';
	import FormErrors from '$components/form-errors.svelte';
	import FormSection from '$components/form-section.svelte';
	import Form from '$components/form.svelte';
	import InputFile from '$components/input-file.svelte';
	import Stack from '$components/stack.svelte';
	import TextArea from '$components/text-area.svelte';
	import TextInput from '$components/text-input.svelte';
	import { buildCommand, type CurlPayload } from '$lib/curl';
	import { buildFormData } from '$lib/form';
	import l from '$lib/localization';
	import routes from '$lib/path';
	import service, {
		type Environment,
		type QueueDeployment,
		type SourceDataDiscriminator
	} from '$lib/resources/deployments';
	import select from '$lib/select';

	export let data;

	let environment: Environment = 'production';
	let kind: SourceDataDiscriminator = data.app.version_control ? 'git' : 'raw';
	let raw = '';
	let archive: Maybe<FileList> = undefined;
	let branch = '';
	let hash: Maybe<string> = undefined;
	let curlPanelVisible = false;

	const options: Environment[] = ['production', 'staging'];
	const kindOptions = (
		[
			{ label: l.translate('deployment.payload.raw'), value: 'raw' },
			{ label: l.translate('deployment.payload.archive'), value: 'archive' },
			{ label: l.translate('deployment.payload.vcs'), value: 'git' }
		] satisfies DropdownOption<SourceDataDiscriminator>[]
	).filter((k) => (!data.app.version_control ? k.value !== 'git' : true));

	async function submit() {
		const payload = select<SourceDataDiscriminator, () => QueueDeployment>(kind, {
			archive: () =>
				buildFormData({
					environment,
					archive: archive![0]
				}),
			git: () => ({
				environment,
				git: {
					branch,
					hash: hash ? hash : undefined
				}
			}),
			raw: () => ({
				environment,
				raw
			})
		})?.();

		if (!payload) {
			return;
		}

		const { deployment_number } = await service.queue(data.app.id, payload);
		await goto(routes.deployment(data.app.id, deployment_number));
	}

	function toggleCurlPanel() {
		curlPanelVisible = !curlPanelVisible;
	}

	$: curlCommand = curlPanelVisible // Only build the command if the panel is visible
		? buildCommand({
				apiKey: data.user.api_key,
				appId: data.app.id,
				environment,
				origin: location.origin,
				...select<SourceDataDiscriminator, CurlPayload>(kind, {
					raw: { kind: 'raw', raw },
					git: { kind: 'git', branch, hash },
					archive: { kind: 'archive', filename: archive?.[0]?.name }
				})!
		  })
		: undefined;
</script>

<Form handler={submit} let:submitting let:errors>
	<Breadcrumb
		segments={[
			{ path: routes.apps, title: l.translate('breadcrumb.applications') },
			{ path: routes.app(data.app.id), title: data.app.name },
			l.translate('breadcrumb.deployment.new')
		]}
	>
		{#if data.app.cleanup_requested_at}
			<CleanupNotice requested_at={data.app.cleanup_requested_at} />
		{:else}
			<Button type="submit" loading={submitting} text="deployment.deploy" />
		{/if}
	</Breadcrumb>

	<Stack direction="column">
		<FormErrors {errors} />

		<div>
			<FormSection title="deployment.environment">
				<Dropdown label="deployment.environment" {options} bind:value={environment} />
			</FormSection>

			<FormSection title="deployment.payload">
				<svelte:fragment slot="actions">
					<Button
						variant="outlined"
						on:click={toggleCurlPanel}
						ariaExpanded={curlPanelVisible}
						ariaControls="curl-command-panel"
						text="deployment.payload.toggle_curl_command"
					/>
				</svelte:fragment>

				<Stack direction="column">
					{#if curlPanelVisible}
						<Console
							id="curl-command-panel"
							title="command"
							titleElement="h3"
							copyToClipboardEnabled
							selectAllEnabled
							data={curlCommand}
						/>
					{/if}

					<Dropdown label="deployment.payload.kind" options={kindOptions} bind:value={kind} />
					{#if kind === 'raw'}
						<TextArea
							code
							rows={20}
							required
							label="deployment.payload.raw.content"
							bind:value={raw}
							remoteError={errors?.['raw.content']}
						>
							<p>{l.translate('deployment.payload.raw.content.help')}</p>
						</TextArea>
					{:else if kind === 'git'}
						<TextInput
							label="deployment.payload.vcs.branch"
							bind:value={branch}
							required
							remoteError={errors?.['git.branch']}
						/>
						<TextInput
							label="deployment.payload.vcs.commit"
							bind:value={hash}
							remoteError={errors?.['git.hash']}
						>
							<p>{l.translate('deployment.payload.vcs.commit.help')}</p>
						</TextInput>
					{:else if kind === 'archive'}
						<InputFile accept="application/gzip" label="file" required bind:files={archive} />
					{/if}
				</Stack>
			</FormSection>
		</div>
	</Stack>
</Form>

<style module>
	.container {
		background-color: var(--co-background-5);
		border-radius: var(--ra-4);
		padding: var(--sp-6);
	}
</style>
