<script lang="ts">
	import { goto } from '$app/navigation';
	import Breadcrumb from '$components/breadcrumb.svelte';
	import Button from '$components/button.svelte';
	import CleanupNotice from '$components/cleanup-notice.svelte';
	import Dropdown, { type DropdownOption } from '$components/dropdown.svelte';
	import FormErrors from '$components/form-errors.svelte';
	import FormSection from '$components/form-section.svelte';
	import Form from '$components/form.svelte';
	import InputFile from '$components/input-file.svelte';
	import Stack from '$components/stack.svelte';
	import TextArea from '$components/text-area.svelte';
	import TextInput from '$components/text-input.svelte';
	import { buildFormData } from '$lib/form';
	import l from '$lib/localization';
	import routes from '$lib/path';
	import service, {
		type EnvironmentName,
		type QueueDeployment,
		type SourceDataDiscriminator
	} from '$lib/resources/deployments';
	import select from '$lib/select';

	export let data;

	let environment: EnvironmentName = 'production';
	let kind: SourceDataDiscriminator = data.app.version_control ? 'git' : 'raw';
	let raw = '';
	let archive: Maybe<FileList> = undefined;
	let branch = '';
	let hash: Maybe<string> = undefined;

	const options: EnvironmentName[] = ['production', 'staging'];
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

	function copyCurlCommand() {
		const payload = select(kind, {
			git: `-H "Content-Type: application/json" -d "{ \\"environment\\":\\"${environment}\\",\\"git\\":{ \\"branch\\": \\"${branch}\\"${
				hash ? `, \\"hash\\":\\"${hash}\\"` : ''
			} } }" `,
			raw: `-H "Content-Type: application/json" -d "{ \\"environment\\":\\"${environment}\\", \\"raw\\":\\"${JSON.stringify(
				raw
			)
				.replaceAll('\\"', '"')
				.substring(1)
				.slice(0, -1)}\\"}"`,
			archive: `-F environment=${environment} -F archive=@${
				archive?.[0]?.name ?? '<path_to_a_tar_gz_archive>'
			}`
		});

		if (!payload) {
			return;
		}

		navigator.clipboard.writeText(
			`curl -i -X POST -H "Authorization: Bearer ${data.user.api_key}" ${payload} ${location.origin}/api/v1/apps/${data.app.id}/deployments`
		);
	}
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
						on:click={copyCurlCommand}
						text="deployment.payload.copy_curl"
					/>
				</svelte:fragment>

				<Stack direction="column">
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
						<InputFile
							accept="application/gzip"
							label="file"
							required
							bind:files={archive}
							remoteError={errors?.['archive.file']}
						/>
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
