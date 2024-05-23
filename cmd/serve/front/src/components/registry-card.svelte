<script lang="ts">
	import Card from '$components/card.svelte';
	import Link from '$components/link.svelte';
	import routes from '$lib/path';
	import type { Registry } from '$lib/resources/registries';

	export let data: Registry;
</script>

<Card>
	<h2 class="title"><Link href={routes.editRegistry(data.id)}>{data.name}</Link></h2>
	<div class="meta">{data.url}</div>
</Card>

<style module>
	.title {
		color: var(--co-text-5);
		font: var(--ty-heading-2);
		white-space: nowrap;
		text-overflow: ellipsis;
		overflow: hidden;
	}

	.meta {
		font: var(--ty-caption);
		white-space: nowrap;
		text-overflow: ellipsis;
		overflow: hidden;
	}
</style>
