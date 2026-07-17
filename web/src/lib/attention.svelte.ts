// Shared "needs attention" projection (open alerts, due tasks, low stock,
// unhealthy integrations). One fetch feeds both the Home page panel and the
// header Hub indicator, so they never disagree. Cheap to recompute server-side,
// so we just re-load on a timer and after actions (completing a task, etc.).
import { getAttention } from './api';
import type { Attention } from './types';

const EMPTY: Attention = { alerts: [], tasks: [], lowStock: [], integrations: [] };

class AttentionState {
	data = $state<Attention>(EMPTY);
	loaded = $state(false);

	/** Total number of actionable items across every category. */
	get count(): number {
		const d = this.data;
		return d.alerts.length + d.tasks.length + d.lowStock.length + d.integrations.length;
	}

	/** Whether any open alert is critical — drives the header's red state. */
	get hasCritical(): boolean {
		return this.data.alerts.some((a) => a.severity === 'critical');
	}

	async load() {
		try {
			this.data = await getAttention();
			this.loaded = true;
		} catch {
			/* unauthenticated or offline — keep whatever we had */
		}
	}

	reset() {
		this.data = EMPTY;
		this.loaded = false;
	}
}

export const attention = new AttentionState();
