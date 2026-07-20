<script lang="ts">
	import { errMsg } from '$lib/errors';
	import { toast } from '$lib/toast.svelte';
	import type { InventoryCategory, InventoryItem, InventoryProduct, InventoryStatus } from '$lib/types';
	import {
		createInventoryItem,
		updateInventoryItem,
		inventoryItemImageURL,
		inventoryProductImageURL
	} from '$lib/api';
	import { Button, Dialog, Select } from '$lib/components/ui';
	import AttributeFields from '$lib/components/AttributeFields.svelte';
	import ImagePlus from '@lucide/svelte/icons/image-plus';
	import Plus from '@lucide/svelte/icons/plus';
	import X from '@lucide/svelte/icons/x';

	interface Props {
		open?: boolean;
		/** Provided in edit mode; omit to create. */
		item?: InventoryItem;
		categories: InventoryCategory[];
		/** Built-in product templates across all categories. */
		products: InventoryProduct[];
		/** Preselect a category (create mode). */
		defaultCategory?: string;
		onSaved?: (i: InventoryItem) => void;
	}
	let { open = $bindable(false), item, categories, products, defaultCategory, onSaved }: Props = $props();

	interface LineDraft { size: string; quantity: number | string; lowStockAt: number | string }

	let categoryId = $state('');
	let productId = $state(''); // bound preset, '' = custom
	let name = $state('');
	let lines = $state<LineDraft[]>([]);
	let location = $state('');
	let status = $state<InventoryStatus>('active');
	let notes = $state('');
	let attributes = $state<Record<string, string>>({});
	// Image editing state: `imageData` is a freshly-picked data URL; `removeImage`
	// clears the user's stored one.
	let imageData = $state('');
	let removeImage = $state(false);
	let busy = $state(false);
	let err = $state('');

	const selectedCategory = $derived(categories.find((c) => c.id === categoryId));
	const colSchema = $derived(selectedCategory?.columns ?? []);
	const categoryProducts = $derived(products.filter((p) => p.category === categoryId));
	const boundProduct = $derived(products.find((p) => p.id === productId));
	// Predefined pack sizes offered as datalist suggestions (still free-typeable).
	const presetVariants = $derived(boundProduct?.variants ?? []);
	const canSave = $derived(!!name.trim() && !!categoryId);

	const previewSrc = $derived(
		imageData ||
			(!removeImage && item?.imageType
				? inventoryItemImageURL(item.id)
				: boundProduct?.hasImage
					? inventoryProductImageURL(boundProduct.id)
					: '')
	);
	const canClearImage = $derived(!!imageData || (!removeImage && !!item?.imageType));

	// The catalog code for a given size, when it matches a bound product variant.
	function codeFor(size: string): string {
		return presetVariants.find((v) => v.size === size.trim())?.code ?? '';
	}

	function blankLine(size = ''): LineDraft {
		return { size, quantity: 0, lowStockAt: 0 };
	}

	// Reseed on open. A saved preset is pre-selected but does NOT re-autofill —
	// the item keeps its own (possibly customized) sizes and quantities.
	$effect(() => {
		if (!open) return;
		categoryId = item?.category ?? defaultCategory ?? categories[0]?.id ?? '';
		productId = item?.productId ?? '';
		name = item?.name ?? '';
		lines =
			item?.variants?.length
				? item.variants.map((v) => ({ size: v.size, quantity: v.quantity, lowStockAt: v.lowStockAt ?? 0 }))
				: [blankLine()];
		location = item?.location ?? '';
		status = item?.status ?? 'active';
		notes = item?.notes ?? '';
		attributes = { ...(item?.attributes ?? {}) };
		imageData = '';
		removeImage = false;
		err = '';
	});

	function onCategoryChange() {
		if (productId && boundProduct?.category !== categoryId) productId = '';
	}

	// Picking a preset fills name and columns, and seeds a stock line per pack
	// size (quantity 0) for the user to fill in.
	function onPresetChange() {
		const p = products.find((x) => x.id === productId);
		if (!p) return; // "Custom" — leave fields, just unbind
		name = p.name;
		attributes = { ...(p.attributes ?? {}) };
		if (p.variants?.length) {
			lines = p.variants.map((v) => blankLine(v.size));
		} else {
			lines = [blankLine(p.unit ?? '')];
		}
	}

	function addLine() {
		// Suggest the next predefined size not already listed, else a blank row.
		const used = new Set(lines.map((l) => l.size.trim()));
		const next = presetVariants.find((v) => !used.has(v.size));
		lines = [...lines, blankLine(next?.size ?? '')];
	}
	function removeLine(i: number) {
		lines = lines.filter((_, idx) => idx !== i);
	}

	function onFile(e: Event) {
		const input = e.target as HTMLInputElement;
		const file = input.files?.[0];
		if (!file) return;
		if (!file.type.startsWith('image/')) {
			err = 'Please choose an image file.';
			return;
		}
		if (file.size > 4 * 1024 * 1024) {
			err = 'Image must be under 4 MB.';
			return;
		}
		const reader = new FileReader();
		reader.onload = () => {
			imageData = String(reader.result ?? '');
			removeImage = false;
			err = '';
		};
		reader.readAsDataURL(file);
	}

	function clearImage() {
		imageData = '';
		removeImage = true;
	}

	async function save() {
		if (!canSave) return;
		busy = true;
		err = '';
		try {
			const attrs: Record<string, string> = {};
			for (const [k, v] of Object.entries(attributes)) {
				if (v === null || v === undefined || v === '') continue;
				attrs[k] = String(v);
			}
			const variants = lines
				.map((l) => ({
					size: l.size.trim(),
					quantity: Number(l.quantity) || 0,
					lowStockAt: Number(l.lowStockAt) || 0
				}))
				.filter((l) => l.size !== '' || l.quantity !== 0 || l.lowStockAt !== 0);
			const input = {
				category: categoryId,
				name: name.trim(),
				variants,
				location: location.trim(),
				status,
				notes: notes.trim(),
				attributes: attrs,
				productId,
				...(imageData ? { image: imageData } : {}),
				...(removeImage ? { removeImage: true } : {})
			};
			const saved = item ? await updateInventoryItem(item.id, input) : await createInventoryItem(input);
			open = false;
			toast.success(item ? 'Item updated' : 'Item added', { description: saved.name });
			onSaved?.(saved);
		} catch (e) {
			err = errMsg(e, 'Save failed');
		} finally {
			busy = false;
		}
	}

	const field =
		'w-full rounded-md border border-rig-700 bg-rig-950 px-3 py-1.5 text-sm focus:border-leaf focus:outline-none';
</script>

<Dialog
	bind:open
	title={item ? 'Edit item' : 'New item'}
	description="A thing you own for growing. Pick a preset to auto-fill, or fill it in yourself."
>
	<div class="space-y-4">
		{#if err}<p class="text-xs text-danger">{err}</p>{/if}

		<div class="flex gap-4">
			<!-- Image: user upload, falling back to the bound product's catalog image -->
			<div class="shrink-0">
				<span class="text-xs text-rig-400">Image</span>
				<div class="relative mt-1 h-24 w-24 overflow-hidden rounded-lg border border-rig-700 bg-rig-950">
					{#if previewSrc}
						<img src={previewSrc} alt={name} class="h-full w-full object-contain" />
						{#if canClearImage}
							<button
								type="button"
								onclick={clearImage}
								aria-label="Remove image"
								class="absolute right-1 top-1 rounded-full bg-rig-950/80 p-1 text-rig-300 hover:text-white"
							>
								<X size={13} />
							</button>
						{/if}
					{:else}
						<label class="flex h-full w-full cursor-pointer flex-col items-center justify-center gap-1 text-rig-500 hover:text-rig-300">
							<ImagePlus size={20} />
							<span class="text-[10px]">Upload</span>
							<input type="file" accept="image/*" class="hidden" onchange={onFile} />
						</label>
					{/if}
				</div>
			</div>

			<!-- Name, then Category, then Preset -->
			<div class="flex-1 space-y-3">
				<label class="block">
					<span class="text-xs text-rig-400">Name</span>
					<input bind:value={name} placeholder="e.g. BioBizz Bio·Grow" class="{field} mt-1" />
				</label>
				<div class="grid gap-3 sm:grid-cols-2">
					<label class="block">
						<span class="text-xs text-rig-400">Category</span>
						<Select
							class="mt-1"
							bind:value={categoryId}
							placeholder="Select…"
							items={categories.map((c) => ({ value: c.id, label: c.label }))}
							onValueChange={onCategoryChange}
						/>
					</label>
					<label class="block">
						<span class="text-xs text-rig-400">Preset <span class="text-rig-600">(optional)</span></span>
						<Select
							class="mt-1"
							bind:value={productId}
							disabled={!categoryProducts.length}
							placeholder={categoryProducts.length ? 'Custom (no preset)' : 'No presets'}
							items={[
								{ value: '', label: categoryProducts.length ? 'Custom (no preset)' : 'No presets' },
								...categoryProducts.map((p) => ({ value: p.id, label: p.name }))
							]}
							onValueChange={onPresetChange}
						/>
					</label>
				</div>
			</div>
		</div>

		{#if boundProduct?.description}
			<p class="rounded-md border border-rig-800 bg-rig-900/40 px-3 py-2 text-xs text-rig-400">{boundProduct.description}</p>
		{/if}

		<!-- Sizes / stock lines: each size has its own quantity and low-stock level -->
		<div>
			<div class="mb-1.5 flex items-center justify-between">
				<span class="text-xs text-rig-400">Sizes &amp; quantities</span>
				<button
					type="button"
					onclick={addLine}
					class="inline-flex items-center gap-1 rounded-md border border-rig-700 px-2 py-1 text-[11px] text-rig-300 hover:border-leaf hover:text-white"
				>
					<Plus size={12} /> Add size
				</button>
			</div>
			<datalist id="inv-preset-sizes">
				{#each presetVariants as v (v.size)}<option value={v.size}></option>{/each}
			</datalist>
			<div class="space-y-2">
				<!-- Column headers -->
				<div class="grid grid-cols-[1fr_5rem_6rem_auto] items-center gap-2 text-[10px] uppercase tracking-wide text-rig-600">
					<span>Size</span>
					<span>Qty</span>
					<span>Low at</span>
					<span></span>
				</div>
				{#each lines as line, i (i)}
					<div class="grid grid-cols-[1fr_5rem_6rem_auto] items-center gap-2">
						<div>
							<input
								bind:value={line.size}
								list="inv-preset-sizes"
								placeholder={presetVariants.length ? 'Pick or type a size' : 'e.g. 1 L, pcs (optional)'}
								class={field}
							/>
							{#if codeFor(line.size)}<span class="mt-0.5 block text-[10px] text-rig-500">Code: {codeFor(line.size)}</span>{/if}
						</div>
						<input type="number" inputmode="decimal" step="any" min="0" bind:value={line.quantity} class={field} />
						<input type="number" inputmode="decimal" step="any" min="0" bind:value={line.lowStockAt} class={field} />
						<button
							type="button"
							onclick={() => removeLine(i)}
							aria-label="Remove size"
							disabled={lines.length === 1}
							class="rounded p-1.5 text-rig-500 hover:text-danger disabled:opacity-30 disabled:hover:text-rig-500"
						>
							<X size={15} />
						</button>
					</div>
				{/each}
			</div>
		</div>

		<div class="grid gap-3 sm:grid-cols-2">
			<label class="block">
				<span class="text-xs text-rig-400">Location</span>
				<input bind:value={location} placeholder="e.g. Shelf A, Tent 1" class="{field} mt-1" />
			</label>
			<label class="block">
				<span class="text-xs text-rig-400">Status</span>
				<Select
					class="mt-1"
					value={status}
					onValueChange={(v) => (status = v as InventoryStatus)}
					items={[
						{ value: 'active', label: 'In stock' },
						{ value: 'ordered', label: 'Ordered' },
						{ value: 'archived', label: 'Archived' }
					]}
				/>
			</label>
		</div>

		<!-- Category-specific columns, rendered from the category schema -->
		{#if colSchema.length}
			<AttributeFields schema={colSchema} bind:values={attributes} />
		{/if}

		<label class="block">
			<span class="text-xs text-rig-400">Notes</span>
			<textarea bind:value={notes} rows="2" class="{field} mt-1"></textarea>
		</label>

		<div class="flex justify-end gap-2 border-t border-rig-800 pt-4">
			<Button variant="ghost" onclick={() => (open = false)}>Cancel</Button>
			<Button onclick={save} disabled={busy || !canSave}>Save</Button>
		</div>
	</div>
</Dialog>
