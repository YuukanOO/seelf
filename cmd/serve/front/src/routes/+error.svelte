<script lang="ts">
	import Stack from '$components/stack.svelte';
	import l from '$lib/localization';
</script>

<Stack class="container" direction="column" justify="center" align="center">
	<h1 class="title">{l.translate('unexpected_error')}</h1>
	<div>
		{@html l.translate('unexpected_error.description')}
	</div>
</Stack>

<style module>
	.container {
		margin: auto;
		padding: var(--sp-4);
		max-width: var(--mi-container-width);
		min-height: 100vh;
		text-align: center;
	}

	.title {
		color: var(--co-primary-4);
		font: var(--ty-heading-1);
	}
</style>
