<script lang="ts">
	import { onMount } from 'svelte';
	import { auth } from '$lib/auth.svelte';
	import { toast } from '$lib/toast.svelte';
	import {
		getInventoryCategories,
		getInventoryProducts,
		getInventoryItems,
		deleteInventoryItem,
		inventoryItemImageURL,
		inventoryProductImageURL
	} from '$lib/api';
	import type { InventoryCategory, InventoryItem, InventoryProduct, InventoryStockLine } from '$lib/types';
	import { titleCase } from '$lib/format';
	import InventoryItemFormModal from '$lib/components/InventoryItemFormModal.svelte';
	import Plus from '@lucide/svelte/icons/plus';
	import Package from '@lucide/svelte/icons/package';
	import Pencil from '@lucide/svelte/icons/pencil';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';

	let categories = $state<InventoryCategory[]>([]);
	let products = $state<InventoryProduct[]>([]);
	let items = $state<InventoryItem[]>([]);
	let editingItem = $state<InventoryItem | undefined>(undefined);
	let modalDefaultCategory = $state<string | undefined>(undefined);
	let modalOpen = $state(false);

	const productsById = $derived(new Map(products.map((p) => [p.id, p])));

	// The item's own uploaded image, else the bound product's catalog image.
	function itemImageURL(it: InventoryItem): string {
		if (it.imageType) return inventoryItemImageURL(it.id);
		if (it.productId && productsById.get(it.productId)?.hasImage) {
			return inventoryProductImageURL(it.productId);
		}
		return '';
	}

	const itemsByCategory = $derived.by(() => {
		const m = new Map<string, InventoryItem[]>();
		for (const c of categories) m.set(c.id, []);
		for (const it of items) {
			if (!m.has(it.category)) m.set(it.category, []);
			m.get(it.category)!.push(it);
		}
		return m;
	});

	// Count of low-stock size lines across all items.
	const lowStockCount = $derived(
		items.reduce((n, it) => n + it.variants.filter(lineLow).length, 0)
	);

	onMount(() => {
		getInventoryCategories().then((c) => (categories = c)).catch(() => {});
		getInventoryProducts().then((p) => (products = p)).catch(() => {});
		refresh();
	});

	function refresh() {
		getInventoryItems().then((i) => (items = i)).catch(() => {});
	}

	function lineLow(v: InventoryStockLine): boolean {
		return (v.lowStockAt ?? 0) > 0 && v.quantity <= (v.lowStockAt ?? 0);
	}

	// A visible size line always exists so an item without variants still renders.
	function lines(it: InventoryItem): InventoryStockLine[] {
		return it.variants.length ? it.variants : [{ size: '', quantity: 0 }];
	}

	function statusLabel(it: InventoryItem): string {
		if (it.status === 'ordered') return 'Ordered';
		if (it.status === 'archived') return 'Archived';
		return 'In stock';
	}

	function newItem(categoryId?: string) {
		editingItem = undefined;
		modalDefaultCategory = categoryId;
		modalOpen = true;
	}
	function editItem(it: InventoryItem) {
		editingItem = it;
		modalDefaultCategory = undefined;
		modalOpen = true;
	}
	async function removeItem(it: InventoryItem) {
		if (!confirm(`Delete “${it.name}” from inventory?`)) return;
		try {
			await deleteInventoryItem(it.id);
			toast.success('Item deleted', { description: it.name });
			refresh();
		} catch (e) {
			toast.error('Could not delete item', { description: it.name });
		}
	}

	// Non-empty attribute values, in the category's declared column order.
	function attrSummary(cat: InventoryCategory, it: InventoryItem): { label: string; value: string }[] {
		const out: { label: string; value: string }[] = [];
		for (const col of cat.columns ?? []) {
			const v = it.attributes?.[col.key];
			if (!v) continue;
			out.push({ label: col.label, value: col.type === 'enum' ? titleCase(v) : v });
		}
		return out;
	}

	// Canonical description resolved live from the bound product, if any.
	function itemDescription(it: InventoryItem): string {
		return (it.productId && productsById.get(it.productId)?.description) || '';
	}
</script>

<div class="mb-6 flex items-start justify-between gap-4">
	<div>
		<h1 class="text-2xl font-semibold">Inventory</h1>
		<p class="text-sm text-rig-400">
			Everything you own for growing — consumables, plant material, equipment, supplies and storage.
		</p>
	</div>
	{#if auth.isAdmin && items.length}
		<button
			onclick={() => newItem()}
			class="inline-flex shrink-0 items-center gap-1.5 rounded-md border border-rig-700 px-3 py-1.5 text-xs font-medium text-rig-200 transition-colors hover:border-leaf hover:text-white"
		>
			<Plus size={14} /> New item
		</button>
	{/if}
</div>

{#if items.length}
	<div class="mb-6 flex flex-wrap gap-3 text-sm">
		<span class="rounded-lg border border-rig-800 bg-rig-900/50 px-3 py-1.5 text-rig-300">
			{items.length} item{items.length === 1 ? '' : 's'}
		</span>
		{#if lowStockCount}
			<span class="inline-flex items-center gap-1.5 rounded-lg border border-warn/40 bg-warn/10 px-3 py-1.5 text-warn">
				<TriangleAlert size={14} /> {lowStockCount} low on stock
			</span>
		{/if}
	</div>
{/if}

{#if items.length === 0}
	<div class="rounded-xl border border-dashed border-rig-800 p-10 text-center">
		<div class="mb-3 flex justify-center text-rig-500"><Package size={40} /></div>
		<h3 class="mb-1 text-lg font-semibold">Nothing in inventory yet</h3>
		<p class="mb-5 text-sm text-rig-400">
			Track quantity, location and stock levels for the supplies, equipment and plant material you keep on hand.
		</p>
		{#if auth.isAdmin}
			<button
				onclick={() => newItem()}
				class="rounded-md bg-rig-50 px-5 py-2 text-sm font-medium text-rig-950 transition-colors hover:bg-rig-200"
			>
				Add an item
			</button>
		{/if}
	</div>
{:else}
	<div class="space-y-10">
		{#each categories as cat (cat.id)}
			{@const catItems = itemsByCategory.get(cat.id) ?? []}
			<section>
				<div class="mb-3 flex items-center justify-between gap-4">
					<div>
						<h2 class="text-sm font-semibold uppercase tracking-wide text-rig-400">
							{cat.label}{catItems.length ? ` · ${catItems.length}` : ''}
						</h2>
						{#if cat.description}<p class="text-xs text-rig-500">{cat.description}</p>{/if}
					</div>
					{#if auth.isAdmin}
						<button
							onclick={() => newItem(cat.id)}
							aria-label="Add item to {cat.label}"
							class="inline-flex shrink-0 items-center gap-1.5 rounded-md border border-rig-800 px-2.5 py-1.5 text-xs text-rig-300 transition-colors hover:border-leaf hover:text-white"
						>
							<Plus size={13} /> Add
						</button>
					{/if}
				</div>

				{#if catItems.length === 0}
					<p class="rounded-lg border border-dashed border-rig-800/70 px-4 py-3 text-xs text-rig-500">
						No items in this category yet.
					</p>
				{:else}
					<div class="overflow-x-auto rounded-xl border border-rig-800">
						<table class="w-full border-collapse text-sm">
							<thead>
								<tr class="border-b border-rig-800 bg-rig-900/50 text-left text-xs uppercase tracking-wide text-rig-500">
									<th class="px-4 py-2.5 font-medium">Item</th>
									<th class="px-4 py-2.5 font-medium">Size</th>
									<th class="px-4 py-2.5 text-right font-medium">Qty</th>
									<th class="px-4 py-2.5 font-medium">Location</th>
									<th class="px-4 py-2.5 font-medium">Status</th>
									{#if auth.isAdmin}<th class="px-4 py-2.5"></th>{/if}
								</tr>
							</thead>
							<tbody>
								{#each catItems as it (it.id)}
									{@const vl = lines(it)}
									{#each vl as v, vi (vi)}
										{@const last = vi === vl.length - 1}
										<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
										<tr
											class="hover:bg-rig-900/30 {last ? 'border-b border-rig-800/60 last:border-0' : ''}"
											class:cursor-pointer={auth.isAdmin}
											onclick={() => auth.isAdmin && editItem(it)}
										>
											<td class="px-4 py-2.5 align-top">
												{#if vi === 0}
													<div class="flex items-start gap-3">
														{#if itemImageURL(it)}
															<img
																src={itemImageURL(it)}
																alt={it.name}
																class="mt-0.5 h-9 w-9 shrink-0 rounded-md border border-rig-800 bg-rig-950 object-contain"
															/>
														{/if}
														<div class="min-w-0">
															<div class="flex items-center gap-2 font-medium text-rig-100">
																{it.name}
																{#if it.productId}
																	<span class="rounded-full bg-rig-800 px-1.5 py-0.5 text-[10px] text-rig-400" title="From product catalog">Catalog</span>
																{/if}
															</div>
															{#if attrSummary(cat, it).length}
																<div class="mt-1 flex flex-wrap gap-1">
																	{#each attrSummary(cat, it) as a (a.label)}
																		<span class="rounded bg-rig-800/70 px-1.5 py-0.5 text-[11px] text-rig-300">
																			<span class="text-rig-500">{a.label}:</span> {a.value}
																		</span>
																	{/each}
																</div>
															{/if}
															{#if itemDescription(it)}<div class="mt-1 line-clamp-1 max-w-md text-xs text-rig-500">{itemDescription(it)}</div>{/if}
															{#if it.notes}<div class="mt-1 line-clamp-1 max-w-md text-xs text-rig-400">{it.notes}</div>{/if}
														</div>
													</div>
												{/if}
											</td>
											<td class="px-4 py-2.5 align-top">
												{#if v.size}
													<span class="rounded-full bg-rig-800 px-2 py-0.5 text-[11px] text-rig-300">{v.size}</span>
												{:else}
													<span class="text-rig-600">—</span>
												{/if}
											</td>
											<td class="px-4 py-2.5 align-top text-right tabular-nums text-rig-200">
												<span class="inline-flex items-center justify-end gap-1.5">
													{v.quantity}
													{#if lineLow(v)}
														<span class="inline-flex items-center gap-1 rounded-full bg-warn/15 px-1.5 py-0.5 text-[10px] font-medium text-warn">
															<TriangleAlert size={11} /> Low
														</span>
													{/if}
												</span>
											</td>
											<td class="px-4 py-2.5 align-top text-rig-300">{#if vi === 0}{it.location || '—'}{/if}</td>
											<td class="px-4 py-2.5 align-top">
												{#if vi === 0}
													<span
														class="rounded-full px-2 py-0.5 text-[11px]"
														class:bg-rig-800={it.status !== 'ordered'}
														class:text-rig-300={it.status !== 'ordered'}
														class:bg-leaf={it.status === 'ordered'}
														class:text-rig-950={it.status === 'ordered'}
													>
														{statusLabel(it)}
													</span>
												{/if}
											</td>
											{#if auth.isAdmin}
												<td class="px-4 py-2.5 align-top">
													{#if vi === 0}
														<div class="flex justify-end gap-1">
															<button
																onclick={(e) => { e.stopPropagation(); editItem(it); }}
																aria-label="Edit item"
																class="rounded p-1.5 text-rig-400 hover:text-rig-100"
															>
																<Pencil size={14} />
															</button>
															<button
																onclick={(e) => { e.stopPropagation(); removeItem(it); }}
																aria-label="Delete item"
																class="rounded p-1.5 text-rig-400 hover:text-danger"
															>
																<Trash2 size={14} />
															</button>
														</div>
													{/if}
												</td>
											{/if}
										</tr>
									{/each}
								{/each}
							</tbody>
						</table>
					</div>
				{/if}
			</section>
		{/each}
	</div>
{/if}

{#if auth.isAdmin}
	<InventoryItemFormModal
		bind:open={modalOpen}
		item={editingItem}
		{categories}
		{products}
		defaultCategory={modalDefaultCategory}
		onSaved={refresh}
	/>
{/if}
