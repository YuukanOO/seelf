<script lang="ts">
	export let direction: 'row' | 'column' = 'row';
	export let align: 'flex-start' | 'center' | 'stretch' | 'flex-end' =
		direction === 'row' ? 'center' : 'stretch';
	export let justify:
		| 'flex-start'
		| 'center'
		| 'stretch'
		| 'space-around'
		| 'space-between'
		| 'flex-end' = 'flex-start';
	export let gap: number = 4;
	export let wrap: 'no-wrap' | 'wrap' | 'wrap-reverse' = 'no-wrap';
	export let style: Maybe<string> = undefined;

	let className: string = '';

	/** Additional css classes */
	export { className as class };

	/** DOM element to render (defaults to div) */
	export let as: string = 'div';
</script>

<svelte:element
	this={as}
	{...$$restProps}
	class="stack {className}"
	style="--stack-gap: var(--sp-{gap}, 0); {style}"
>
	<slot />
</svelte:element>

<style module>
	.stack {
		display: flex;
		flex-direction: bind(direction);
		align-items: bind(align);
		flex-wrap: bind(wrap);
		justify-content: bind(justify);
		gap: var(--stack-gap);
	}
</style>
