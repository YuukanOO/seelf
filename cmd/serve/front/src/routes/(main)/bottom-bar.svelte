<script lang="ts">
	import Link from '$components/link.svelte';
	import Prose from '$components/prose.svelte';
	import Stack from '$components/stack.svelte';
	import l from '$lib/localization';

	export let version: string;
</script>

<Prose class="bottom-bar">
	<Stack justify="space-between" wrap="wrap">
		<p>{l.translate('footer.description', [version])}</p>
		<Stack as="ul">
			<li>
				<Link href="https://github.com/YuukanOO/seelf/" external newWindow>
					{l.translate('footer.source')}
				</Link>
			</li>
			<li>
				<Link href="https://yuukanoo.github.io/seelf/guide/quickstart.html" external newWindow>
					{l.translate('footer.documentation')}
				</Link>
			</li>
			<li>
				<Link href="https://yuukanoo.github.io/seelf/contributing/donating.html" external newWindow>
					{l.translate('footer.donate')}
				</Link>
			</li>
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
