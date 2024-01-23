<div class="grid">
	<slot />
</div>

<style module>
	.grid {
		align-items: flex-start;
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(15rem, 1fr));
		gap: var(--sp-4);
	}
</style>
