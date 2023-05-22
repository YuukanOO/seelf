<script lang="ts">
	import Prose from '$components/prose.svelte';
</script>

<Prose class="blank-slate">
	<slot />
</Prose>

<style module>
	.blank-slate {
		background-color: var(--co-background-5);
		border-radius: var(--ra-4);
		min-height: 25vh;
		padding: var(--sp-6);
		text-align: center;
		display: flex;
		align-items: center;
		justify-content: center;
	}
</style>
