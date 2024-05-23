<script lang="ts">
	import type { Profile } from '$lib/resources/users';
	import Link from '$components/link.svelte';
	import Account from './account.svelte';
	import Stack from '$components/stack.svelte';
	import Menu from '$assets/icons/menu.svelte';
	import routes from '$lib/path';
	import l from '$lib/localization';
	import Logo from '$assets/logo-dark.svelte';

	export let user: Profile;
</script>

<Stack class="topbar" gap={10} justify="space-between">
	<a href="/" class="logo" title={l.translate('breadcrumb.home')}>
		<Logo />
	</a>
	<div class="menu-container">
		<input id="menu" type="checkbox" />
		<label class="menu-toggle" aria-label={l.translate('menu.toggle')} for="menu"><Menu /></label>
		<div class="menu-content">
			<nav class="nav">
				<Link href={routes.apps} class="link">{l.translate('breadcrumb.applications')}</Link>
				<Link href={routes.targets} class="link">{l.translate('breadcrumb.targets')}</Link>
				<Link href={routes.registries} class="link">{l.translate('breadcrumb.registries')}</Link>
				<Link href={routes.jobs} class="link">{l.translate('breadcrumb.jobs')}</Link>
			</nav>
			<Account {user} />
		</div>
	</div>
</Stack>

<style module>
	body:has(#menu:checked) {
		overflow: hidden;
	}

	.topbar {
		border-block-end: 1px solid var(--co-divider-4);
		padding: var(--sp-4);
		margin: 0 calc(var(--sp-4) * -1);
	}

	.logo svg {
		display: block;
		height: 1.5rem;
		width: auto;
	}

	.nav {
		display: flex;
		flex-direction: column;
		align-items: center;
		flex: 1;
		gap: var(--sp-6);
	}

	.link {
		font-weight: 600;
		color: var(--co-text-5);
	}

	.menu-toggle {
		cursor: pointer;
		background-color: transparent;
		transition: all 0.2s ease-in-out;
	}

	.menu-toggle svg {
		display: block;
		height: 1.5rem;
		width: 1.5rem;
	}

	.menu-content {
		border-start-start-radius: var(--ra-4);
		border-end-start-radius: var(--ra-4);
		display: flex;
		flex-direction: column;
		position: fixed;
		gap: var(--sp-10);
		padding: var(--sp-6);
		background-color: var(--co-background-5);
		z-index: 10;
		inset: 0;
		inset-inline-start: 25%;
		opacity: 0;
		transform: translateX(100%);
		transition: all 0.2s ease-in-out;
		box-shadow: 0 10px 10px var(--co-background-4);
		overflow: auto;
	}

	#menu {
		appearance: none;
		position: absolute;
	}

	#menu:checked ~ .menu-content {
		opacity: 1;
		transform: translateX(0);
	}

	#menu:checked ~ .menu-toggle {
		position: fixed;
		inset: 0;
		background-color: var(--co-shadow--4);
		z-index: 5;
	}

	#menu:checked ~ .menu-toggle svg {
		display: none;
	}

	@media screen and (min-width: 56rem) {
		body {
			overflow: auto !important;
		}

		#menu,
		.menu-toggle {
			display: none;
		}

		.menu-container {
			flex: 1;
		}

		.nav {
			flex-direction: row;
			justify-content: flex-start;
			align-items: center;
			flex-wrap: wrap;
		}

		.menu-content {
			background-color: transparent;
			flex-direction: row;
			position: initial;
			opacity: 1;
			padding: 0;
			transform: translateX(0);
			transition: none;
		}
	}
</style>
