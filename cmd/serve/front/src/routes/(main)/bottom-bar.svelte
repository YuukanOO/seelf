<script lang="ts">
	import Link from '$components/link.svelte';
	import Prose from '$components/prose.svelte';
	import Stack from '$components/stack.svelte';
</script>

<Prose class="bottom-bar">
	<Stack justify="space-between" wrap="wrap">
		<p>seelf &mdash; Painless self-hosted deployment platform</p>
		<Stack as="ul">
			<li><Link href="https://github.com/YuukanOO/seelf/" external newWindow>Source</Link></li>
			<li>
				<Link href="https://github.com/YuukanOO/seelf/blob/main/DOCUMENTATION.md" external newWindow
					>Docs</Link
				>
			</li>
			<li><Link href="https://liberapay.com/YuukanOO/" external newWindow>❤️ Donate</Link></li>
		</Stack>
	</Stack>
</Prose>

<style module>
	.bottom-bar {
		border-block-start: 1px solid var(--co-divider-4);
		padding: var(--sp-4);
		font: var(--ty-caption);

		margin-inline: calc(var(--sp-4) * -1);
		margin-block-start: var(--sp-10);
	}
</style>
