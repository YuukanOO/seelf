<script lang="ts">
	import { goto } from '$app/navigation';
	import Breadcrumb from '$components/breadcrumb.svelte';
	import Button from '$components/button.svelte';
	import CleanupNotice from '$components/cleanup-notice.svelte';
	import Dropdown, { type DropdownOption } from '$components/dropdown.svelte';
	import FormSection from '$components/form-section.svelte';
	import Form from '$components/form.svelte';
	import InputFile from '$components/input-file.svelte';
	import Stack from '$components/stack.svelte';
	import TextArea from '$components/text-area.svelte';
	import TextInput from '$components/text-input.svelte';
	import { buildFormData } from '$lib/form.js';
	import routes from '$lib/path';
	import service, {
		type Environment,
		type SourceDataDiscriminator,
		type QueueDeploymentData
	} from '$lib/resources/deployments';
	import select from '$lib/select.js';

	export let data;

	let environment: Environment = 'production';
	let kind: SourceDataDiscriminator = data.app.vcs ? 'git' : 'raw';
	let raw = '';
	let archive: Maybe<FileList> = undefined;
	let branch = '';
	let hash: Maybe<string> = undefined;

	const options: Environment[] = ['production', 'staging'];
	const kindOptions = (
		[
			{ label: 'compose file', value: 'raw' },
			{ label: 'project archive (tar.gz)', value: 'archive' },
			{ label: 'git', value: 'git' }
		] satisfies DropdownOption<SourceDataDiscriminator>[]
	).filter((k) => (!data.app.vcs ? k.value !== 'git' : true));

	async function submit() {
		const payload = select<SourceDataDiscriminator, () => QueueDeploymentData>(kind, {
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
		let payload: string = ` `;

		switch (kind) {
			case 'git':
				payload = `-H "Content-Type: application/json" -d "{ \\"environment\\":\\"${environment}\\",\\"git\\":{ \\"branch\\": \\"${branch}\\"${
					hash ? `, \\"hash\\":\\"${hash}\\"` : ''
				} } }" `;
				break;
			case 'raw':
				const rawAsStr = JSON.stringify(raw).replaceAll('\\"', '"').substring(1).slice(0, -1);
				payload = `-H "Content-Type: application/json" -d "{ \\"environment\\":\\"${environment}\\", \\"raw\\":\\"${rawAsStr}\\"}"`;
				break;
			case 'archive':
				payload = `-F environment=${environment} -F archive=@${
					archive?.[0]?.name ?? '<path_to_a_tar_gz_archive>'
				}`;
				break;
		}

		navigator.clipboard.writeText(
			`curl -i -X POST -H "Authorization: Bearer ${data.user.api_key}" ${payload} ${location.origin}/api/v1/apps/${data.app.id}/deployments`
		);
	}
</script>

<Form handler={submit} let:submitting let:errors>
	<Breadcrumb
		segments={[
			{ path: routes.apps, title: 'Applications' },
			{ path: routes.app(data.app.id), title: data.app.name },
			'New deployment'
		]}
	>
		{#if data.app.cleanup_requested_at}
			<CleanupNotice data={data.app} />
		{:else}
			<Button type="submit" loading={submitting}>Deploy</Button>
		{/if}
	</Breadcrumb>

	<FormSection title="Environment">
		<Dropdown label="Target" {options} bind:value={environment} />
	</FormSection>

	<FormSection title="Payload">
		<svelte:fragment slot="actions">
			<Button variant="outlined" on:click={copyCurlCommand}>Copy cURL command</Button>
		</svelte:fragment>

		<Stack direction="column">
			<Dropdown label="Kind" options={kindOptions} bind:value={kind} />
			{#if kind === 'raw'}
				<TextArea
					code
					rows={20}
					required
					label="Content"
					bind:value={raw}
					remoteError={errors.content}
				>
					<p>
						Content of the service file (compose.yml if you're using Docker Compose for example).
					</p>
				</TextArea>
			{:else if kind === 'git'}
				<TextInput label="Branch" bind:value={branch} required remoteError={errors.branch} />
				<TextInput label="Commit" bind:value={hash} remoteError={errors.hash}>
					<p>Optional specific commit to deploy. Leave empty to deploy the latest branch commit.</p>
				</TextInput>
			{:else if kind === 'archive'}
				<InputFile accept="application/gzip" label="File" required bind:files={archive} />
			{/if}
		</Stack>
	</FormSection>
</Form>

<style module>
	.container {
		background-color: var(--co-background-5);
		border-radius: var(--ra-4);
		padding: var(--sp-6);
	}
</style>
