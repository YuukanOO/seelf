export type Paginated<T> = {
	data: T[];
	total: number;
	page: number;
	first_page: boolean;
	last_page: boolean;
	per_page: number;
};
