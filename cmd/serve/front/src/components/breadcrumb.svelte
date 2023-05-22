<script lang="ts">
	import PageTitle from '$components/page-title.svelte';
	import Stack from '$components/stack.svelte';
	import Link from '$components/link.svelte';

	type BreadcrumbSegment = string | { path: string; title: string };

	/** Breadcumb segments to render */
	export let segments: BreadcrumbSegment[] = [];

	function isString(seg: BreadcrumbSegment): seg is string {
		return typeof seg === 'string';
	}

	$: lastSegment = segments[segments.length - 1];
	$: lastSegmentTitle = isString(lastSegment) ? lastSegment : lastSegment.title;

	/** Custom title to set. If not given, will default to the last segment title */
	export let title: Maybe<string> = undefined;
</script>

<PageTitle title={title ?? lastSegmentTitle} />

<div class="breadcrumb">
	{#if segments.length > 1}
		<Stack gap={0}>
			{#each segments.slice(0, segments.length - 1) as segment}
				<div class="segment">
					{#if isString(segment)}
						{segment}
					{:else}
						<Link class="link" href={segment.path}>{segment.title}</Link>
					{/if}
				</div>
			{/each}
		</Stack>
	{/if}
	<Stack wrap="wrap" justify="space-between">
		<h1 class="title">{lastSegmentTitle}</h1>
		<Stack>
			<slot />
		</Stack>
	</Stack>
</div>

<style module>
	.breadcrumb {
		padding: var(--sp-10) 0;
	}

	.title {
		font: var(--ty-heading-1);
		color: var(--co-text-5);
		white-space: nowrap;
		text-overflow: ellipsis;
		overflow: hidden;
	}

	.segment {
		display: flex;
	}

	.segment::after {
		content: '/';
		display: block;
		margin: 0 var(--sp-1);
	}
</style>
