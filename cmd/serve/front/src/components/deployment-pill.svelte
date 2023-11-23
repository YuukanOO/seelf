<script lang="ts">
	import type { DeploymentData } from '$lib/resources/deployments';
	import Git from '$assets/icons/git.svelte';
	import Archive from '$assets/icons/archive.svelte';
	import File from '$assets/icons/file.svelte';
	import Stack from '$components/stack.svelte';
	import select from '$lib/select';
	import routes from '$lib/path';
	import l from '$lib/localization';

	export let data: DeploymentData;
</script>

<Stack
	as="a"
	href={routes.deployment(data.app_id, data.deployment_number)}
	title={l.translate('deployment.details_tooltip', [data.deployment_number])}
	class="pill"
	gap={1}
>
	<svelte:component
		this={select(data.source.discriminator, {
			git: Git,
			archive: Archive,
			raw: File
		})}
	/>
	<span>#{data.deployment_number}</span>
</Stack>

<style module>
	.pill {
		box-shadow: 0 0 6px var(--co-background-4);
		background-color: var(--co-background-5);
		border-radius: var(--ra-4);
		border: 1px solid var(--co-divider-4);
		font: var(--ty-caption);
		padding: var(--sp-1) var(--sp-2);
	}

	.pill:hover,
	.pill:focus {
		border-color: var(--co-primary-4);
		color: var(--co-primary-4);
	}

	.pill:focus {
		outline: var(--ou-primary);
	}

	.pill svg {
		height: 1rem;
		width: 1rem;
	}
</style>
