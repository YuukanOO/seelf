import { writable, type Readable } from 'svelte/store';
import { BadRequestError } from '$lib/error';

/**
 * Keys inside SubmitterErrors representing an error not tied to a specific field.
 */
export const GLOBAL_ERROR_NAME = '__global';

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

/**
 * Builds a FormData instance from the given record data.
 */
export function buildFormData(data: Record<string, string | Blob>): FormData {
	return Object.entries(data).reduce((fd, [key, value]) => {
		fd.append(key, value);
		return fd;
	}, new FormData());
}

export type SubmitterErrors = Maybe<Record<string, Maybe<string>>>;

export type Submitter<T> = {
	loading: Readable<boolean>;
	submit: () => Promise<T>;
	errors: Readable<SubmitterErrors>;
};

export type SubmitterOptions = {
	/** Optional confirmation string to display before submitting */
	confirmation?: string;
};

/**
 * Wrap the given function exposing a loading state, a submitter and formatted errors.
 * It also handle common options such as displaying a confirmation message before submitting.
 */
export function submitter<T = unknown>(
	fn: () => Promise<T>,
	options?: SubmitterOptions
): Submitter<T> {
	const loading = writable(false);
	const errors = writable<SubmitterErrors>(undefined);

	async function submit(): Promise<T> {
		if (options?.confirmation && !confirm(options.confirmation)) {
			return undefined as T;
		}

		loading.set(true);
		errors.set(undefined);

		try {
			return await fn();
		} catch (ex) {
			if (ex instanceof BadRequestError) {
				errors.set(ex.isValidationError ? ex.fields : { [GLOBAL_ERROR_NAME]: ex.message });
			} else if (ex instanceof Error) {
				errors.set({ [GLOBAL_ERROR_NAME]: ex.message });
			}

			throw ex;
		} finally {
			loading.set(false);
		}
	}

	return {
		loading,
		errors,
		submit
	};
}
