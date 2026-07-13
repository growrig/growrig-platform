<script lang="ts">
	import type { CareActionDef, CareField, GrowCareActionConfig } from '$lib/types';
	import { saveCareConfig } from '$lib/api';
	import { Button, Dialog } from '$lib/components/ui';
	import ArrowUp from '@lucide/svelte/icons/arrow-up';
	import ArrowDown from '@lucide/svelte/icons/arrow-down';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import Plus from '@lucide/svelte/icons/plus';

	interface Props {
		open?: boolean;
		growId: string;
		/** The grow's current effective care actions (from GET care-config). */
		actions: CareActionDef[];
		onSaved?: (actions: CareActionDef[]) => void;
	}
	let { open = $bindable(false), growId, actions, onSaved }: Props = $props();

	// Editable working copy; `orig` label lets us send a rename override only when
	// a built-in's label actually changed.
	type Row = CareActionDef & { orig: string };
	let rows = $state<Row[]>([]);
	let busy = $state(false);
	let err = $state('');

	$effect(() => {
		if (!open) return;
		rows = actions.map((a) => ({ ...a, orig: a.label }));
		err = '';
		showAdd = false;
	});

	function move(i: number, delta: number) {
		const j = i + delta;
		if (j < 0 || j >= rows.length) return;
		const next = [...rows];
		[next[i], next[j]] = [next[j], next[i]];
		rows = next;
	}
	function remove(i: number) {
		rows = rows.filter((_, k) => k !== i);
	}

	// --- add a custom action ---
	let showAdd = $state(false);
	let newLabel = $state('');
	let newFields = $state<Set<CareField>>(new Set(['note']));
	const ALL_FIELDS: { key: CareField; label: string }[] = [
		{ key: 'amount', label: 'Amount' },
		{ key: 'ph', label: 'pH' },
		{ key: 'ec', label: 'EC' },
		{ key: 'runoff', label: 'Runoff' },
		{ key: 'recipe', label: 'Recipe' },
		{ key: 'product', label: 'Product' },
		{ key: 'trainType', label: 'Method' },
		{ key: 'potSize', label: 'Pot size' },
		{ key: 'note', label: 'Note' },
		{ key: 'photos', label: 'Photos' }
	];
	function slug(s: string) {
		return s.trim().toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
	}
	function toggleField(f: CareField) {
		const next = new Set(newFields);
		next.has(f) ? next.delete(f) : next.add(f);
		newFields = next;
	}
	function addCustom() {
		const label = newLabel.trim();
		if (!label) return;
		let key = slug(label) || 'custom';
		const taken = new Set(rows.map((r) => r.key));
		let k = key, n = 2;
		while (taken.has(k)) k = `${key}-${n++}`;
		rows = [
			...rows,
			{ key: k, label, icon: 'list-plus', fields: [...newFields], quick: false, enabled: true, custom: true, orig: label }
		];
		newLabel = '';
		newFields = new Set(['note']);
		showAdd = false;
	}

	const canSave = $derived(rows.every((r) => r.label.trim() !== ''));

	async function save() {
		busy = true;
		err = '';
		try {
			const payload: GrowCareActionConfig[] = rows.map((r) => {
				const cfg: GrowCareActionConfig = { key: r.key, enabled: r.enabled, quick: !!r.quick };
				if (r.custom) {
					cfg.custom = true;
					cfg.label = r.label.trim();
					cfg.fields = r.fields;
				} else if (r.label.trim() !== r.orig) {
					cfg.label = r.label.trim();
				}
				return cfg;
			});
			const res = await saveCareConfig(growId, payload);
			open = false;
			onSaved?.(res.actions);
		} catch (e) {
			err = e instanceof Error ? e.message : 'Failed to save';
		} finally {
			busy = false;
		}
	}

	async function resetDefaults() {
		if (!confirm('Reset care actions to the species defaults? Custom actions will be removed.')) return;
		busy = true;
		err = '';
		try {
			const res = await saveCareConfig(growId, []);
			open = false;
			onSaved?.(res.actions);
		} catch (e) {
			err = e instanceof Error ? e.message : 'Failed to reset';
		} finally {
			busy = false;
		}
	}

	const field =
		'w-full rounded-md border border-rig-700 bg-rig-950 px-2.5 py-1 text-sm focus:border-rig-500 focus:outline-none';
</script>

<Dialog bind:open title="Care actions" description="Enable, reorder, rename and add the actions this grow can log." size="2xl">
	<div class="space-y-4">
		{#if err}<p class="rounded-md border border-danger/40 bg-danger/10 px-3 py-2 text-xs text-danger">{err}</p>{/if}

		<div class="overflow-hidden rounded-xl border border-rig-800">
			<div class="grid grid-cols-[auto_1fr_auto_auto_auto] items-center gap-2 border-b border-rig-800 px-3 py-2 text-[11px] uppercase tracking-wide text-rig-500">
				<span>Order</span><span>Action</span><span class="px-1">On</span><span class="px-1">Quick</span><span></span>
			</div>
			{#each rows as row, i (row.key)}
				<div class="grid grid-cols-[auto_1fr_auto_auto_auto] items-center gap-2 border-b border-rig-800/60 px-3 py-2 last:border-0">
					<div class="flex flex-col">
						<button onclick={() => move(i, -1)} disabled={i === 0} aria-label="Move up" class="text-rig-500 hover:text-rig-200 disabled:opacity-30"><ArrowUp size={13} /></button>
						<button onclick={() => move(i, 1)} disabled={i === rows.length - 1} aria-label="Move down" class="text-rig-500 hover:text-rig-200 disabled:opacity-30"><ArrowDown size={13} /></button>
					</div>
					<div class="flex items-center gap-2">
						<input bind:value={row.label} class={field} />
						{#if row.custom}<span class="rounded-full bg-rig-800 px-2 py-0.5 text-[10px] uppercase tracking-wide text-rig-400">Custom</span>{/if}
					</div>
					<input type="checkbox" bind:checked={row.enabled} class="accent-leaf justify-self-center" aria-label="Enabled" />
					<input type="checkbox" bind:checked={row.quick} disabled={!row.enabled} class="accent-leaf justify-self-center disabled:opacity-40" aria-label="Quick action" />
					<button onclick={() => remove(i)} disabled={!row.custom} aria-label="Delete" title={row.custom ? 'Delete custom action' : 'Built-in actions can be disabled but not deleted'} class="text-rig-500 hover:text-danger disabled:opacity-20"><Trash2 size={14} /></button>
				</div>
			{/each}
		</div>

		{#if showAdd}
			<div class="space-y-3 rounded-xl border border-rig-700 bg-rig-950/40 p-3">
				<label class="block"><span class="text-xs text-rig-400">Custom action name</span>
					<input bind:value={newLabel} placeholder="e.g. Foliar spray" class="{field} mt-1" /></label>
				<div>
					<span class="text-xs text-rig-400">Fields</span>
					<div class="mt-1.5 flex flex-wrap gap-1.5">
						{#each ALL_FIELDS as f (f.key)}
							<button
								onclick={() => toggleField(f.key)}
								class="rounded-full border px-2.5 py-0.5 text-xs transition-colors {newFields.has(f.key) ? 'border-leaf/60 bg-leaf/15 text-leaf' : 'border-rig-700 text-rig-400 hover:border-rig-500'}"
							>{f.label}</button>
						{/each}
					</div>
				</div>
				<div class="flex justify-end gap-2">
					<Button size="sm" variant="ghost" onclick={() => (showAdd = false)}>Cancel</Button>
					<Button size="sm" onclick={addCustom} disabled={!newLabel.trim()}>Add</Button>
				</div>
			</div>
		{:else}
			<button onclick={() => (showAdd = true)} class="inline-flex items-center gap-1.5 text-sm text-rig-400 hover:text-rig-100"><Plus size={14} /> Add custom action</button>
		{/if}

		<div class="flex items-center justify-between gap-2 border-t border-rig-800 pt-4">
			<Button variant="ghost" onclick={resetDefaults} disabled={busy}>Reset to defaults</Button>
			<div class="flex gap-2">
				<Button variant="ghost" onclick={() => (open = false)}>Cancel</Button>
				<Button onclick={save} disabled={busy || !canSave}>Save</Button>
			</div>
		</div>
	</div>
</Dialog>
