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

export type Submitter<T> = {
	loading: Readable<boolean>;
	submit: () => Promise<T>;
};

export type SubmitterOptions = {
	/** Optional confirmation string to display before submitting */
	confirmation?: string;
};

/**
 * Wrap the given function exposing a loading state and a submitter. It also
 * handle common options such as displaying a confirmation message before submitting.
 */
export function submitter<T = unknown>(
	fn: () => Promise<T>,
	options?: SubmitterOptions
): Submitter<T> {
	const loading = writable(false);

	async function submit(): Promise<T> {
		if (options?.confirmation && !confirm(options.confirmation)) {
			return undefined as T;
		}

		loading.set(true);

		try {
			return await fn();
		} finally {
			loading.set(false);
		}
	}

	return {
		loading,
		submit
	};
}
