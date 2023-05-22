<script lang="ts">
	import { navigating } from '$app/stores';
	import BottomBar from './bottom-bar.svelte';
	import ProgressBar from './progress-bar.svelte';
	import Topbar from './topbar.svelte';

	export let data;
</script>

<ProgressBar visible={$navigating != null} />

<div class="container">
	<Topbar user={data.user} version={data.health.version} />
	<div class="content">
		<slot />
	</div>
	<BottomBar />
</div>

<style module>
	.container {
		display: flex;
		flex-direction: column;
		margin: auto;
		padding-inline: var(--sp-4);
		max-width: var(--mi-container-width);
		min-height: 100vh;
	}

	.content {
		flex: 1;
	}
</style>
