const dateFormat = new Intl.DateTimeFormat('en', {
	day: '2-digit',
	month: '2-digit',
	year: 'numeric'
});

const datetimeFormat = new Intl.DateTimeFormat('en', {
	day: '2-digit',
	month: '2-digit',
	year: 'numeric',
	hour: '2-digit',
	minute: '2-digit',
	second: '2-digit'
});

/** Returns a human readable date from a raw date value */
export const formatDate = (value: string | number | Date): string =>
	dateFormat.format(new Date(value));

/** Returns a human readable datetime from a raw date value */
export const formatDatetime = (value: string | number | Date): string =>
	datetimeFormat.format(new Date(value));

/** Returns a human readable duration string from a start and end date */
export function formatDuration(start: string | number | Date, end: string | number | Date): string {
	const diffInSeconds = Math.max(
		Math.floor((new Date(end).getTime() - new Date(start).getTime()) / 1000),
		0
	);

	const numberOfMinutes = Math.floor(diffInSeconds / 60);
	const numberOfSeconds = diffInSeconds - numberOfMinutes * 60;

	if (numberOfMinutes === 0) {
		return `${numberOfSeconds}s`;
	}

	return `${numberOfMinutes}m ${numberOfSeconds}s`;
}
