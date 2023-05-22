<script lang="ts">
	let className: string = '';

	/** Additional css classes */
	export { className as class };

	export let href: string;

	/** Mark the link as external adding needed attributes (rel mostly) */
	export let external: boolean = false;
	/** Open the link in a new tab/window */
	export let newWindow: boolean = false;
	export let title: Maybe<string> = undefined;
</script>

<a
	class="link {className}"
	{href}
	{title}
	rel={external ? 'noopener noreferrer' : undefined}
	target={newWindow ? '_blank' : undefined}
>
	<slot />
</a>

<style module>
	.link:hover,
	.link:focus {
		text-decoration: underline;
		text-underline-offset: 2px;
	}
</style>
