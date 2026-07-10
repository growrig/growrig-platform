// Shared helpers for the GrowRig UI kit (bits-ui + Tailwind design tokens).

/** Join class values, dropping falsy entries. Later strings win only by order,
 *  so keep per-call overrides last. */
export function cn(...parts: Array<string | false | null | undefined>): string {
	return parts.filter(Boolean).join(' ');
}

export type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'danger';
export type ButtonSize = 'sm' | 'md' | 'lg' | 'icon';

export const buttonVariants: Record<ButtonVariant, string> = {
	primary: 'bg-rig-500 text-rig-950 hover:bg-rig-400 focus-visible:ring-rig-400',
	secondary:
		'border border-rig-700 bg-rig-900 text-rig-100 hover:border-rig-600 hover:bg-rig-800 focus-visible:ring-rig-600',
	ghost: 'text-rig-300 hover:bg-rig-800/60 hover:text-rig-100 focus-visible:ring-rig-700',
	danger: 'bg-danger text-white hover:bg-danger/85 focus-visible:ring-danger'
};

export const buttonSizes: Record<ButtonSize, string> = {
	sm: 'h-8 gap-1.5 px-3 text-xs',
	md: 'h-9 gap-2 px-4 text-sm',
	lg: 'h-11 gap-2 px-6 text-base',
	icon: 'h-9 w-9'
};

export const buttonBase =
	'inline-flex select-none items-center justify-center rounded-md font-medium transition-colors ' +
	'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-offset-rig-950 ' +
	'disabled:pointer-events-none disabled:opacity-50';
