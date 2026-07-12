<script lang="ts">
	import type { FeedingPreset, FeedingProduct, FeedingPhase, Species } from '$lib/types';
	import { createFeedingPreset, updateFeedingPreset, type FeedingPresetInput } from '$lib/api';
	import { Button, Dialog } from '$lib/components/ui';
	import { titleCase } from '$lib/format';
	import Plus from '@lucide/svelte/icons/plus';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import ChevronUp from '@lucide/svelte/icons/chevron-up';
	import ChevronDown from '@lucide/svelte/icons/chevron-down';

	interface Props {
		open?: boolean;
		/** Provided to edit (user preset) or duplicate (built-in preset); omit to create. */
		preset?: FeedingPreset;
		species: Species[];
		/** Preselect a species (create mode). */
		defaultSpecies?: string;
		onSaved?: (p: FeedingPreset) => void;
	}
	let { open = $bindable(false), preset, species, defaultSpecies, onSaved }: Props = $props();

	// A local phase/week shape whose doses hold numbers as edited (may be empty).
	type Week = { doses: Record<string, number | null> };
	type Phase = { name: string; stage: string; weeks: Week[] };

	let speciesId = $state('');
	let name = $state('');
	let brand = $state('');
	let description = $state('');
	let unit = $state('ml/L');
	let products = $state<FeedingProduct[]>([]);
	let phases = $state<Phase[]>([]);
	let busy = $state(false);
	let err = $state('');

	// Built-ins are read-only: editing one means duplicating into a new user preset.
	const isDuplicate = $derived(preset?.source === 'builtin');
	const isEdit = $derived(!!preset && preset.source === 'user');
	const selected = $derived(species.find((s) => s.id === speciesId));
	const stages = $derived(selected?.stages ?? []);
	const canSave = $derived(!!name.trim() && !!speciesId);

	const title = $derived(
		isEdit ? 'Edit feeding preset' : isDuplicate ? 'Duplicate feeding preset' : 'New feeding preset'
	);

	// Reseed on open.
	$effect(() => {
		if (!open) return;
		speciesId = preset?.species ?? defaultSpecies ?? '';
		name = preset ? (isDuplicate ? `${preset.name} (copy)` : preset.name) : '';
		brand = preset?.brand ?? '';
		description = preset?.description ?? '';
		unit = preset?.unit || 'ml/L';
		products = (preset?.products ?? []).map((p) => ({ ...p }));
		phases = (preset?.phases ?? []).map((ph) => ({
			name: ph.name,
			stage: ph.stage ?? '',
			weeks: (ph.weeks ?? []).map((wk) => ({ doses: { ...wk.doses } }))
		}));
		err = '';
	});

	function nextProductKey(): string {
		const used = new Set(products.map((p) => p.key));
		let n = 1;
		while (used.has(`p${n}`)) n++;
		return `p${n}`;
	}
	function addProduct() {
		products = [...products, { key: nextProductKey(), label: '', unit: '' }];
	}
	function removeProduct(i: number) {
		const key = products[i].key;
		products = products.filter((_, k) => k !== i);
		// Drop that product's doses from every week.
		for (const ph of phases) for (const wk of ph.weeks) delete wk.doses[key];
	}

	function addPhase() {
		phases = [...phases, { name: `Phase ${phases.length + 1}`, stage: '', weeks: [{ doses: {} }] }];
	}
	function removePhase(i: number) {
		phases = phases.filter((_, k) => k !== i);
	}
	function movePhase(i: number, dir: -1 | 1) {
		const j = i + dir;
		if (j < 0 || j >= phases.length) return;
		const next = [...phases];
		[next[i], next[j]] = [next[j], next[i]];
		phases = next;
	}
	function addWeek(pi: number) {
		phases[pi].weeks = [...phases[pi].weeks, { doses: {} }];
	}
	function removeWeek(pi: number, wi: number) {
		phases[pi].weeks = phases[pi].weeks.filter((_, k) => k !== wi);
	}

	function unitFor(p: FeedingProduct): string {
		return p.unit?.trim() || unit.trim();
	}

	async function save() {
		if (!canSave) return;
		busy = true;
		err = '';
		try {
			const input: FeedingPresetInput = {
				species: speciesId,
				name: name.trim(),
				brand: brand.trim(),
				description: description.trim(),
				unit: unit.trim(),
				products: products
					.filter((p) => p.label.trim())
					.map((p) => ({ key: p.key, label: p.label.trim(), ...(p.unit?.trim() ? { unit: p.unit.trim() } : {}) })),
				phases: phases
					.filter((ph) => ph.name.trim())
					.map((ph) => ({
						name: ph.name.trim(),
						...(ph.stage ? { stage: ph.stage } : {}),
						weeks: ph.weeks.map((wk) => {
							const doses: Record<string, number> = {};
							for (const [k, v] of Object.entries(wk.doses)) {
								const n = typeof v === 'number' ? v : parseFloat(String(v));
								if (Number.isFinite(n) && n > 0) doses[k] = n;
							}
							return { doses };
						})
					}))
			};
			// Duplicating a built-in always creates a new user preset.
			const saved = isEdit ? await updateFeedingPreset(preset!.id, input) : await createFeedingPreset(input);
			open = false;
			onSaved?.(saved);
		} catch (e) {
			err = e instanceof Error ? e.message : 'Save failed';
		} finally {
			busy = false;
		}
	}

	const field =
		'w-full rounded-md border border-rig-700 bg-rig-950 px-3 py-1.5 text-sm focus:border-rig-500 focus:outline-none';
	const iconBtn = 'rounded p-1.5 text-rig-400 transition-colors hover:text-rig-100 disabled:opacity-30';
</script>

<Dialog
	bind:open
	{title}
	size="3xl"
	description="A nutrient schedule: products dosed per week across the phases of a grow."
>
	<div class="space-y-5">
		{#if err}<p class="text-xs text-danger">{err}</p>{/if}
		{#if isDuplicate}
			<p class="rounded-md border border-rig-700 bg-rig-950 px-3 py-2 text-xs text-rig-400">
				This is a built-in preset. Saving creates your own editable copy.
			</p>
		{/if}

		<!-- Core fields -->
		<div class="grid gap-3 sm:grid-cols-2">
			<label class="block">
				<span class="text-xs text-rig-400">Name</span>
				<input bind:value={name} placeholder="e.g. My veg + bloom" class="{field} mt-1" />
			</label>
			<label class="block">
				<span class="text-xs text-rig-400">Species</span>
				<select bind:value={speciesId} class="{field} mt-1 capitalize">
					<option value="" disabled>Select a species…</option>
					{#each species as sp (sp.id)}<option value={sp.id}>{sp.label}</option>{/each}
				</select>
			</label>
			<label class="block">
				<span class="text-xs text-rig-400">Brand <span class="text-rig-600">(optional)</span></span>
				<input bind:value={brand} placeholder="e.g. BioBizz" class="{field} mt-1" />
			</label>
			<label class="block">
				<span class="text-xs text-rig-400">Default unit</span>
				<input bind:value={unit} placeholder="ml/L" class="{field} mt-1" />
			</label>
		</div>
		<label class="block">
			<span class="text-xs text-rig-400">Description <span class="text-rig-600">(optional)</span></span>
			<textarea bind:value={description} rows="2" class="{field} mt-1"></textarea>
		</label>

		<!-- Products -->
		<section class="space-y-2">
			<div class="flex items-center justify-between">
				<h3 class="text-xs font-semibold uppercase tracking-wide text-leaf">Products</h3>
				<button
					type="button"
					onclick={addProduct}
					class="inline-flex items-center gap-1 rounded-md border border-rig-700 px-2 py-1 text-xs text-rig-200 hover:border-rig-500 hover:text-white"
				>
					<Plus size={13} /> Add product
				</button>
			</div>
			{#if products.length === 0}
				<p class="text-xs text-rig-500">Add the nutrient lines this schedule doses (e.g. Bio·Grow, Bio·Bloom).</p>
			{:else}
				<div class="space-y-1.5">
					{#each products as p, i (p.key)}
						<div class="flex items-center gap-2">
							<input bind:value={p.label} placeholder="Product name" class="{field} flex-1" />
							<input
								bind:value={p.unit}
								placeholder={unit || 'unit'}
								class="{field} w-24"
								title="Unit override (blank = default)"
							/>
							<button type="button" onclick={() => removeProduct(i)} aria-label="Remove product" class={iconBtn}>
								<Trash2 size={14} />
							</button>
						</div>
					{/each}
				</div>
			{/if}
		</section>

		<!-- Phases -->
		<section class="space-y-3">
			<div class="flex items-center justify-between">
				<h3 class="text-xs font-semibold uppercase tracking-wide text-leaf">Phases &amp; weeks</h3>
				<button
					type="button"
					onclick={addPhase}
					class="inline-flex items-center gap-1 rounded-md border border-rig-700 px-2 py-1 text-xs text-rig-200 hover:border-rig-500 hover:text-white"
				>
					<Plus size={13} /> Add phase
				</button>
			</div>
			{#if phases.length === 0}
				<p class="text-xs text-rig-500">Add phases (e.g. Vegetative, Flowering) and the weeks within each.</p>
			{/if}

			{#each phases as phase, pi (pi)}
				<div class="rounded-lg border border-rig-800 bg-rig-900/40 p-3">
					<div class="mb-2 flex items-center gap-2">
						<input bind:value={phase.name} placeholder="Phase name" class="{field} flex-1" />
						<select bind:value={phase.stage} class="{field} w-40 capitalize" title="Linked stage (optional)">
							<option value="">No stage link</option>
							{#each stages as st (st.name)}<option value={st.name}>{titleCase(st.name)}</option>{/each}
						</select>
						<button type="button" onclick={() => movePhase(pi, -1)} disabled={pi === 0} aria-label="Move up" class={iconBtn}>
							<ChevronUp size={14} />
						</button>
						<button
							type="button"
							onclick={() => movePhase(pi, 1)}
							disabled={pi === phases.length - 1}
							aria-label="Move down"
							class={iconBtn}
						>
							<ChevronDown size={14} />
						</button>
						<button type="button" onclick={() => removePhase(pi)} aria-label="Remove phase" class={iconBtn}>
							<Trash2 size={14} />
						</button>
					</div>

					{#if products.length === 0}
						<p class="text-xs text-rig-500">Add products above to dose them here.</p>
					{:else}
						<div class="overflow-x-auto">
							<table class="w-full border-collapse text-sm">
								<thead>
									<tr class="text-left text-xs text-rig-500">
										<th class="sticky left-0 z-10 bg-rig-900/40 py-1 pr-3 font-medium">Product</th>
										{#each phase.weeks as _wk, wi (wi)}
											<th class="px-1 py-1 text-center font-medium">
												<div class="flex items-center justify-center gap-0.5">
													Wk {wi + 1}
													<button
														type="button"
														onclick={() => removeWeek(pi, wi)}
														aria-label="Remove week"
														class="text-rig-600 hover:text-danger"
													>
														<Trash2 size={11} />
													</button>
												</div>
											</th>
										{/each}
										<th class="px-1 py-1">
											<button
												type="button"
												onclick={() => addWeek(pi)}
												aria-label="Add week"
												class="inline-flex items-center gap-0.5 rounded border border-rig-700 px-1.5 py-0.5 text-[11px] text-rig-300 hover:border-rig-500 hover:text-white"
											>
												<Plus size={11} /> Week
											</button>
										</th>
									</tr>
								</thead>
								<tbody>
									{#each products as p (p.key)}
										<tr class="border-t border-rig-800/60">
											<td class="sticky left-0 z-10 whitespace-nowrap bg-rig-900/40 py-1 pr-3">
												<span class="text-rig-200">{p.label || '—'}</span>
												<span class="ml-1 text-[10px] text-rig-500">{unitFor(p)}</span>
											</td>
											{#each phase.weeks as _wk, wi (wi)}
												<td class="px-1 py-1 text-center">
													<input
														type="number"
														inputmode="decimal"
														step="any"
														min="0"
														bind:value={phase.weeks[wi].doses[p.key]}
														class="w-14 rounded border border-rig-700 bg-rig-950 px-1.5 py-1 text-center text-xs focus:border-rig-500 focus:outline-none"
													/>
												</td>
											{/each}
											<td></td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
						{#if phase.weeks.length === 0}
							<p class="mt-1 text-xs text-rig-500">No weeks yet — add one to start dosing.</p>
						{/if}
					{/if}
				</div>
			{/each}
		</section>

		<div class="flex justify-end gap-2 border-t border-rig-800 pt-4">
			<Button variant="ghost" onclick={() => (open = false)}>Cancel</Button>
			<Button onclick={save} disabled={busy || !canSave}>
				{isEdit ? 'Save' : isDuplicate ? 'Save copy' : 'Create'}
			</Button>
		</div>
	</div>
</Dialog>
