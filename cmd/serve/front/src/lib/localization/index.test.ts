import { describe, expect, it } from 'vitest';
import {
	LocalLocalizationService,
	type Locale,
	type LocalizationService,
	type Translations
} from '.';

const translations = {
	greet: (name: string) => `Hello, ${name}!`,
	bye: 'Goodbye!'
} satisfies Translations;

const en = {
	code: 'en',
	displayName: 'English',
	translations
} as const satisfies Locale<typeof translations>;

const fr = {
	code: 'fr',
	displayName: 'Français',
	translations: {
		greet: (name: string) => `Bonjour, ${name}!`,
		bye: 'Au revoir!'
	}
} as const satisfies Locale<typeof translations>;

describe('the LocalLocalizationService', () => {
	const locales = [en, fr] satisfies Locale<typeof translations>[];

	it('should be initialized with the fallback locale if no locale is set', () => {
		const service: LocalizationService<typeof translations, typeof locales> =
			new LocalLocalizationService({
				fallback: 'en',
				locales
			});

		expect(service.locale()).toBe('en');
	});

	it('should be initialized with the fallback locale if the default locale does not exist', () => {
		const service: LocalizationService<typeof translations, typeof locales> =
			new LocalLocalizationService({
				fallback: 'en',
				default: 'it',
				locales
			});

		expect(service.locale()).toBe('en');
	});

	it('should be initialized with the given locale if set', () => {
		const service: LocalizationService<typeof translations, typeof locales> =
			new LocalLocalizationService({
				fallback: 'en',
				default: 'fr',
				locales
			});

		expect(service.locale()).toBe('fr');
	});

	it('should be able to change the locale', () => {
		const service: LocalizationService<typeof translations, typeof locales> =
			new LocalLocalizationService({
				fallback: 'en',
				locales
			});

		service.locale('fr');

		expect(service.locale()).toBe('fr');
	});

	const service: LocalizationService<typeof translations, typeof locales> =
		new LocalLocalizationService({
			fallback: 'en',
			locales
		});

	it('should returns all available locales', () => {
		expect(service.locales()).toEqual(locales);
	});

	it('should returns the translation key if the translation does not exist', () => {
		expect(service.translate('not-found' as any)).toBe('not-found');
	});

	it('should correctly translate a string', () => {
		expect(service.translate('bye')).toBe('Goodbye!');
	});

	it('should correctly translate a string with arguments', () => {
		expect(service.translate('greet', ['John'])).toBe('Hello, John!');
	});
});
