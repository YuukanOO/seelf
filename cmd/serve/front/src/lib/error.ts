/** Represents a generic application error */
export type AppError<TDetail> = {
	code: string;
	detail?: TDetail;
};

/** Error detail for form validation */
export type ValidationDetail = Record<string, AppError<unknown>>;

/** Validation error after a form submission */
export class BadRequestError extends Error {
	public readonly fields: Record<string, Maybe<string>>;
	public readonly isValidationError: boolean;

	public constructor(data: AppError<ValidationDetail>) {
		super(data.code);
		this.isValidationError = data.code === 'validation_failed';
		this.fields = Object.entries(data.detail ?? {}).reduce(
			(result, [name, err]) => ({
				...result,
				[name]: err.code
			}),
			{}
		);
	}
}

/** Represents an unexpected error */
export class UnexpectedError extends Error {
	public constructor() {
		super('unexpected_error');
	}
}

export class UnauthorizedError extends Error {
	public constructor() {
		super('not_authorized');
	}
}

export class HttpError extends Error {
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	private static readonly status: Record<number, Maybe<{ new (data?: any): Error }>> = {
		400: BadRequestError,
		401: UnauthorizedError,
		500: UnexpectedError
	};

	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	public constructor(public readonly status: number, public readonly data?: any) {
		super(data?.code ?? `HTTP Error ${status}`);
	}

	/**
	 * Builds a new Error based on an HTTP response. It will try to raise the appropriate
	 * error if the status code is known, else it will return a generic HttpError.
	 */
	public static async from(response: Response) {
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		let data: any | undefined;

		try {
			data = await response.json();
		} catch {
			/* empty */
		}

		const customErr = HttpError.status[response.status];

		if (customErr) {
			return new customErr(data);
		}

		return new HttpError(response.status, data);
	}
}
