<script lang="ts">
	import Button from '$components/button.svelte';
	import FormErrors from '$components/form-errors.svelte';
	import Form from '$components/form.svelte';
	import PageTitle from '$components/page-title.svelte';
	import Stack from '$components/stack.svelte';
	import TextInput from '$components/text-input.svelte';
	import auth from '$lib/auth';
	import l from '$lib/localization';

	let email = '';
	let password = '';

	async function submit() {
		await auth.login(email, password);
	}
</script>

<PageTitle title={l.translate('auth.signin.title')} />

<div class="signin">
	<div class="signin-card">
		<Form class="signin-form" handler={submit} let:submitting let:errors>
			<Stack direction="column">
				<div>
					<h1 class="title">{l.translate('auth.signin.title')}</h1>
					<p>{l.translate('auth.signin.description')}</p>
				</div>

				<FormErrors {errors} />

				<TextInput
					label="email"
					type="email"
					bind:value={email}
					required
					remoteError={errors?.email}
				/>
				<TextInput
					label="password"
					type="password"
					bind:value={password}
					required
					remoteError={errors?.password}
				/>
				<Stack justify="flex-end">
					<Button type="submit" text="auth.signin.title" loading={submitting} />
				</Stack>
			</Stack>
		</Form>
	</div>
</div>

<style module>
	.signin {
		display: flex;
		align-items: center;
		justify-content: center;
		min-height: 100vh;
	}

	.signin-card {
		background-color: var(--co-background-5);
		background-image: linear-gradient(to top, var(--co-background-5), 94%, transparent),
			repeating-linear-gradient(
				-45deg,
				transparent,
				transparent 8px,
				var(--co-divider-4) 8px,
				var(--co-divider-4) 10px
			);
		border: 1px solid var(--co-primary-4);
		border-radius: var(--ra-4);
		margin: var(--sp-4);
		max-width: 24rem;
		width: 100%;
	}

	.signin-form {
		padding: var(--sp-6);
	}

	.title {
		font: var(--ty-heading-2);
		color: var(--co-text-5);
	}
</style>
