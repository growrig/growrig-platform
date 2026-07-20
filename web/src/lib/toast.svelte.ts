// App-wide toast notifications. Any component can call `toast.success(...)`,
// `toast.info(...)`, etc.; the global <Toaster /> mounted in the root layout
// renders the stack. Toasts auto-dismiss after `duration` ms (0 = sticky).
export type ToastKind = 'info' | 'success' | 'warning' | 'error';

/** An optional call-to-action rendered under the message. */
export interface ToastAction {
	label: string;
	/** Navigate here on click (rendered as a link). */
	href?: string;
	/** Or run this on click. Both dismiss the toast afterwards. */
	onClick?: () => void;
}

export interface Toast {
	id: number;
	kind: ToastKind;
	title: string;
	description?: string;
	action?: ToastAction;
	/** ms until auto-dismiss; 0 keeps it until the user dismisses it. */
	duration: number;
}

export interface ToastOptions {
	description?: string;
	action?: ToastAction;
	duration?: number;
}

const DEFAULT_DURATION = 5000;

class ToastState {
	items = $state<Toast[]>([]);
	#seq = 0;
	#timers = new Map<number, ReturnType<typeof setTimeout>>();

	/** Show a toast of a given kind; returns its id for manual dismissal. */
	show(kind: ToastKind, title: string, opts: ToastOptions = {}): number {
		const id = ++this.#seq;
		const duration = opts.duration ?? DEFAULT_DURATION;
		this.items = [
			...this.items,
			{ id, kind, title, description: opts.description, action: opts.action, duration }
		];
		if (duration > 0 && typeof setTimeout !== 'undefined') {
			this.#timers.set(id, setTimeout(() => this.dismiss(id), duration));
		}
		return id;
	}

	info = (title: string, opts?: ToastOptions) => this.show('info', title, opts);
	success = (title: string, opts?: ToastOptions) => this.show('success', title, opts);
	warning = (title: string, opts?: ToastOptions) => this.show('warning', title, opts);
	error = (title: string, opts?: ToastOptions) => this.show('error', title, opts);

	dismiss(id: number) {
		const timer = this.#timers.get(id);
		if (timer) {
			clearTimeout(timer);
			this.#timers.delete(id);
		}
		this.items = this.items.filter((t) => t.id !== id);
	}

	clear() {
		for (const timer of this.#timers.values()) clearTimeout(timer);
		this.#timers.clear();
		this.items = [];
	}
}

export const toast = new ToastState();
