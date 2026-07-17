// Theme selection: 'system' follows the OS, 'light'/'dark' pin a choice.
// The resolved theme drives the `dark`/`light` class on <html>, which is what
// the `rig-*` ramp in app.css keys off (see the token comment there).
//
// First paint is handled by an inline script in app.html so there's no flash of
// the wrong theme before this module hydrates; this store keeps things in sync
// afterward and owns writes to localStorage.

export type Theme = 'system' | 'light' | 'dark';
export type ResolvedTheme = 'light' | 'dark';

const STORAGE_KEY = 'growrig-theme';

function isTheme(v: unknown): v is Theme {
	return v === 'system' || v === 'light' || v === 'dark';
}

function stored(): Theme {
	if (typeof localStorage === 'undefined') return 'system';
	const v = localStorage.getItem(STORAGE_KEY);
	return isTheme(v) ? v : 'system';
}

function systemPrefersDark(): boolean {
	return typeof matchMedia !== 'undefined' && matchMedia('(prefers-color-scheme: dark)').matches;
}

function resolve(theme: Theme): ResolvedTheme {
	if (theme === 'system') return systemPrefersDark() ? 'dark' : 'light';
	return theme;
}

function applyToDocument(resolved: ResolvedTheme) {
	if (typeof document === 'undefined') return;
	const el = document.documentElement;
	el.classList.toggle('dark', resolved === 'dark');
	el.classList.toggle('light', resolved === 'light');
	// Keep the address-bar / PWA chrome color in step with the page background.
	document
		.querySelector('meta[name="theme-color"]')
		?.setAttribute('content', resolved === 'dark' ? '#0b0f0c' : '#f7fbf7');
}

class ThemeState {
	/** The user's choice, including 'system'. */
	preference = $state<Theme>('system');
	/** The concrete theme in effect right now — 'system' collapsed to light/dark. */
	resolved = $state<ResolvedTheme>('dark');

	/** Call once on mount (client only). Wires up OS-preference changes. */
	init(): () => void {
		this.preference = stored();
		this.resolved = resolve(this.preference);
		applyToDocument(this.resolved);

		const mq = matchMedia('(prefers-color-scheme: dark)');
		const onChange = () => {
			if (this.preference === 'system') {
				this.resolved = resolve('system');
				applyToDocument(this.resolved);
			}
		};
		mq.addEventListener('change', onChange);
		return () => mq.removeEventListener('change', onChange);
	}

	set(theme: Theme) {
		this.preference = theme;
		this.resolved = resolve(theme);
		applyToDocument(this.resolved);
		if (typeof localStorage !== 'undefined') localStorage.setItem(STORAGE_KEY, theme);
	}
}

export const theme = new ThemeState();
