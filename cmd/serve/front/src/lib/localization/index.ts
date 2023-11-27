import { browser } from '$app/environment';
import en from '$lib/localization/en';
import fr from '$lib/localization/fr';

/**
 * Provides formatting methods for a specific locale.
 */
export interface FormatProvider {
	date(value: DateValue): string;
	datetime(value: DateValue): string;
	duration(start: DateValue, end: DateValue): string;
}

export type TranslationsArgs<T> = T extends (...args: infer P) => string ? P : never;
export type TranslationFunc = (this: FormatProvider, ...args: any[]) => string;
export type Translations = Record<string, string | TranslationFunc>;

/**
 * Represents a single locale in the application.
 */
export type Locale<T extends Translations> = {
	code: string;
	displayName: string;
	translations: T;
};

export type LocaleCode<T extends Locale<Translations>[]> = T[number]['code'];

/**
 * Localize resources accross the application.
 */
export interface LocalizationService<T extends Translations, TLocales extends Locale<T>[]>
	extends FormatProvider {
	/** Sets the current locale */
	locale(code: LocaleCode<TLocales>): void;
	/** Gets the current locale */
	locale(): LocaleCode<TLocales>;
	/** Gets all the locales supported by this localization service */
	locales(): TLocales;
	/** Translate a given key */
	translate<TKey extends KeysOfType<T, string>>(key: TKey): string;
	/** Translate a given key with arguments */
	translate<TKey extends KeysOfType<T, TranslationFunc>>(
		key: TKey,
		args: TranslationsArgs<T[TKey]>
	): string;
}

export type LocalLocalizationOptions<T extends Translations, TLocales extends Locale<T>[]> = {
	onLocaleChanged?(locale: LocaleCode<TLocales>, oldLocale: Maybe<LocaleCode<TLocales>>): void;
	default?: string;
	fallback: LocaleCode<TLocales>;
	locales: TLocales;
};

export class LocalLocalizationService<T extends Translations, TLocales extends Locale<T>[]>
	implements LocalizationService<T, TLocales>
{
	private _currentLocaleCode?: LocaleCode<TLocales>;
	private _currentTranslations!: T;

	private readonly _dateOptions: Intl.DateTimeFormatOptions = {
		day: '2-digit',
		month: '2-digit',
		year: 'numeric'
	};

	private readonly _dateTimeOptions: Intl.DateTimeFormatOptions = {
		day: '2-digit',
		month: '2-digit',
		year: 'numeric',
		hour: '2-digit',
		minute: '2-digit',
		second: '2-digit'
	};

	public constructor(private readonly _options: LocalLocalizationOptions<T, TLocales>) {
		this.locale(_options.default ?? _options.fallback);
	}

	date(value: DateValue): string {
		return new Date(value).toLocaleDateString(this._currentLocaleCode, this._dateOptions);
	}

	datetime(value: DateValue): string {
		return new Date(value).toLocaleString(this._currentLocaleCode, this._dateTimeOptions);
	}

	duration(start: DateValue, end: DateValue): string {
		const diffInSeconds = Math.max(
			Math.floor((new Date(end!).getTime() - new Date(start).getTime()) / 1000),
			0
		);

		const numberOfMinutes = Math.floor(diffInSeconds / 60);
		const numberOfSeconds = diffInSeconds - numberOfMinutes * 60;

		// FIXME: handle it better but since for now I only support french and english, this is not needed.
		if (numberOfMinutes === 0) {
			return `${numberOfSeconds}s`;
		}

		return `${numberOfMinutes}m ${numberOfSeconds}s`;
	}

	translate<TKey extends keyof T>(key: TKey, args?: TranslationsArgs<T[TKey]> | undefined): string {
		const v = this._currentTranslations[key];

		if (!v) {
			return key as string;
		}

		if (typeof v === 'function') {
			return v.apply(this, args ?? []);
		}

		return v as string;
	}

	locale(code: LocaleCode<TLocales>): void;
	locale(): LocaleCode<TLocales>;
	locale(code?: LocaleCode<TLocales>) {
		if (!code) {
			return this._currentLocaleCode;
		}

		const targetLocale =
			this._options.locales.find((l) => l.code === code) ??
			this._options.locales.find((l) => l.code === this._options.fallback)!;

		if (targetLocale.code === this._currentLocaleCode) {
			return;
		}

		const oldCode = this._currentLocaleCode;

		this._currentLocaleCode = targetLocale.code;
		this._currentTranslations = targetLocale.translations;
		this._options.onLocaleChanged?.(this._currentLocaleCode, oldCode);
	}

	locales(): TLocales {
		return this._options.locales;
	}
}

/** Type the application translations to provide strong typings. */
export type AppTranslations = (typeof en)['translations'];

const locales = [en, fr] satisfies Locale<AppTranslations>[];

export type AppLocales = typeof locales;
export type AppLocaleCodes = LocaleCode<AppLocales>;
export type AppTranslationsString = KeysOfType<AppTranslations, string>;
export type AppTranslationsFunc = KeysOfType<AppTranslations, TranslationFunc>;

const service: LocalizationService<AppTranslations, AppLocales> = new LocalLocalizationService({
	onLocaleChanged(value, old) {
		if (!browser) {
			return;
		}

		localStorage.setItem('locale', value);

		// Old not defined, this is the localization initialization so no need to reload the page.
		if (!old) {
			return;
		}

		// Reload the page to force the application to re-render with the new locale set.
		// I don't want to make every translations reactive because it will bloat the application for nothing...
		window.location.reload();
	},
	default: browser ? localStorage.getItem('locale') ?? navigator.language : undefined,
	fallback: 'en',
	locales
});

export default service;
