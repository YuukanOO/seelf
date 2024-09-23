<script lang="ts">
	import l from '$lib/localization';

	export let scheme: string;
	export let host: string;
	export let appName: string;
	export let prefix = '';
	export let suffix = '';
</script>

{#if scheme}
	{scheme}{prefix}{appName}{suffix}.{host}
{:else}
	- ({l.translate('target.manual_proxy')})
{/if}
