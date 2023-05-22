<script lang="ts">
	import Link from '$components/link.svelte';
	import Stack from '$components/stack.svelte';
	import auth from '$lib/auth';
	import routes from '$lib/path';
	import type { Profile } from '$lib/resources/users';

	export let user: Profile;
</script>

<Stack class="account" direction="column" gap={0}>
	<span>{user.email}</span>
	<div class="links">
		<Link href={routes.profile}>my profile</Link> -
		<button class="logout" type="button" on:click={() => auth.logout()}>log out</button>
	</div>
</Stack>

<style module>
	.account {
		background-color: var(--co-background-5);
		border-radius: var(--ra-4);
		font: var(--ty-caption);
		padding: var(--sp-2) var(--sp-4);
		align-items: center;
	}

	.links a,
	.links button {
		font: var(--ty-caption);
		color: var(--co-primary-4);
		cursor: pointer;
	}

	.logout:focus,
	.logout:hover {
		text-decoration: underline;
		text-underline-offset: 2px;
	}

	@media screen and (min-width: 56rem) {
		.account {
			align-items: flex-end;
		}
	}
</style>
