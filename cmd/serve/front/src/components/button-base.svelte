<script lang="ts">
	import Loader from '$components/loader.svelte';
	import { type ActionDefinition, useAction } from '$lib/actions';
	import l, { type AppTranslationsString } from '$lib/localization';

	type ButtonType = 'button' | 'submit' | 'reset';

	/** If set, render the button as an anchor */
	export let href: Maybe<string> = undefined;

	export let title: Maybe<AppTranslationsString> = undefined;
	export let type: ButtonType = 'button';
	export let disabled: Maybe<boolean> = undefined;
	export let variant: 'primary' | 'outlined' | 'danger' = 'primary';
	export let loading: boolean = false;
	export let use: Maybe<ActionDefinition> = undefined;

	const outlined = variant === 'outlined';
	const danger = variant === 'danger';
</script>

{#if href}
	<a
		class="button"
		aria-disabled={disabled}
		use:useAction={use}
		class:outlined
		class:danger
		{href}
		title={title && l.translate(title)}
	>
		<slot />
	</a>
{:else}
	<button
		class="button"
		disabled={disabled || loading}
		use:useAction={use}
		class:loading
		class:outlined
		class:danger
		on:click
		{type}
		aria-label={title && l.translate(title)}
	>
		{#if loading}
			<Loader />
		{/if}
		<span class="content">
			<slot />
		</span>
	</button>
{/if}

<style module>
	.button {
		background-color: var(--co-primary-4);
		box-shadow: inset 0 1px var(--co-primary-5);
		border-radius: var(--ra-4);
		border-bottom: 1px solid transparent;
		color: var(--co-primary-0);
		cursor: pointer;
		font: 600 var(--ty-caption);
		padding: var(--sp-1) var(--sp-3);
		display: flex;
		flex-direction: column;
		align-items: center;
		user-select: none;
	}

	.button svg {
		display: block;
		width: 1rem;
		height: 1rem;
	}

	.button:hover,
	.button:focus {
		background-color: var(--co-primary-5);
		color: var(--co-primary-0);
	}

	.button:focus {
		outline: var(--ou-primary);
	}

	.outlined {
		background-color: transparent;
		box-shadow: none;
		border: 1px solid var(--co-divider-4);
		color: var(--co-text-4);
	}

	.button.outlined:hover,
	.button.outlined:focus {
		background-color: transparent;
		color: var(--co-primary-4);
		border-color: var(--co-primary-4);
	}

	.danger {
		background-color: transparent;
		box-shadow: none;
		border: 1px solid var(--co-danger-3);
		color: var(--co-danger-4);
	}

	.button.danger:hover,
	.button.danger:focus {
		background-color: var(--co-danger-1);
		color: var(--co-danger-4);
		border-color: var(--co-danger-4);
	}

	.button.danger:focus {
		outline-color: var(--co-danger-3);
	}

	/** 
	 * Simple trick to avoid the 1px offset when visually removing the content
	 * on loading.
	 */
	.loading {
		border-bottom-width: 0;
	}

	.danger.loading,
	.outlined.loading {
		border-bottom-width: 1px;
	}

	/**
	 * Since I want the button to keep its original width, let's use the position
	 * relative here.
	 */
	.loading .content {
		position: relative;
		left: -10000px;
		top: auto;
		height: 1px;
		overflow: hidden;
	}
</style>
