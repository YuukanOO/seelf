<script lang="ts">
	import Stack from '$components/stack.svelte';
</script>

<Stack class="container" direction="column" justify="center" align="center">
	<h1 class="title">An unexpected error occured.</h1>
	<div>
		<p>Looks like something went wrong. Try refreshing the page.</p>
		<p>If the problem persists, please contact the administrator to investigate.</p>
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
