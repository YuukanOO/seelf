<script lang="ts">
	export let version: string;
</script>

<div>
	<span class="name">seelf</span>
	<span class="version">v{version.substring(0, 16)}</span>
</div>

<style module>
	.name {
		color: var(--co-primary-4);
		display: block;
		font-weight: 600;
	}

	.version {
		font: var(--ty-caption-mono);
		display: block;
	}
</style>
