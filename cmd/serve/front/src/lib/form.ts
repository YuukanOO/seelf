import { writable, type Readable } from 'svelte/store';

/**
 * Computes the validation message based on HTML attributes on an element.
 */
export function messageFromAttributes(attributes: {
	required?: boolean;
	type?: HtmlInputType;
}): string {
	const validations: string[] = [];

	if (attributes.required) {
		validations.push('required');
	}

	if (attributes.type && !['text', 'password'].includes(attributes.type)) {
		validations.push(attributes.type);
	}

	return validations.join(', ');
}

export type PromiseResult = {
	loading: Readable<boolean>;
	submit: () => Promise<unknown>;
};

/**
 * Wrap the given promise and expose a readable loading boolean indicating the state of the request +
 * a submit function to launch the request.
 */
export function promise(fn: () => Promise<unknown>, confirmation?: string): PromiseResult {
	const loading = writable(false);

	async function submit() {
		if (confirmation && !confirm(confirmation)) {
			return;
		}

		loading.set(true);

		try {
			await fn();
		} finally {
			loading.set(false);
		}
	}

	return { loading, submit };
}
