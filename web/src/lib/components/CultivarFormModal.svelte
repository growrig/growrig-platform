<script lang="ts">
	import type { Cultivar, Species } from '$lib/types';
	import { createCultivar, updateCultivar, cultivarImageURL } from '$lib/api';
	import { Button, Dialog, Select, fieldClass } from '$lib/components/ui';
	import AttributeFields from '$lib/components/AttributeFields.svelte';
	import ImagePlus from '@lucide/svelte/icons/image-plus';
	import X from '@lucide/svelte/icons/x';

	interface Props {
		open?: boolean;
		/** Provided in edit mode; omit to create. */
		cultivar?: Cultivar;
		species: Species[];
		/** Preselect a species (create mode). */
		defaultSpecies?: string;
		onSaved?: (c: Cultivar) => void;
	}
	let { open = $bindable(false), cultivar, species, defaultSpecies, onSaved }: Props = $props();

	let speciesId = $state('');
	let name = $state('');
	let creator = $state('');
	let description = $state('');
	let attributes = $state<Record<string, string>>({});
	// Image editing state: `imageData` is a freshly-picked data URL; `removeImage`
	// clears an existing one. When both are unset the stored image is left as-is.
	let imageData = $state('');
	let removeImage = $state(false);
	let busy = $state(false);
	let err = $state('');

	const selected = $derived(species.find((s) => s.id === speciesId));
	const attrSchema = $derived(selected?.cultivarAttributes ?? []);
	const canSave = $derived(!!name.trim() && !!speciesId);

	// The preview shows a newly-picked image, else the stored one (unless cleared).
	const previewSrc = $derived(
		imageData || (!removeImage && cultivar?.imageType ? cultivarImageURL(cultivar.id) : '')
	);

	// Reseed on open.
	$effect(() => {
		if (!open) return;
		speciesId = cultivar?.species ?? defaultSpecies ?? '';
		name = cultivar?.name ?? '';
		creator = cultivar?.creator ?? '';
		description = cultivar?.description ?? '';
		attributes = { ...(cultivar?.attributes ?? {}) };
		imageData = '';
		removeImage = false;
		err = '';
	});

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
			// Number inputs bind as numbers; the API expects string values. Coerce
			// everything and drop empty entries.
			const attrs: Record<string, string> = {};
			for (const [k, v] of Object.entries(attributes)) {
				if (v === null || v === undefined || v === '') continue;
				attrs[k] = String(v);
			}
			const input = {
				species: speciesId,
				name: name.trim(),
				creator: creator.trim(),
				description: description.trim(),
				attributes: attrs,
				...(imageData ? { image: imageData } : {}),
				...(removeImage ? { removeImage: true } : {})
			};
			const saved = cultivar ? await updateCultivar(cultivar.id, input) : await createCultivar(input);
			open = false;
			onSaved?.(saved);
		} catch (e) {
			err = e instanceof Error ? e.message : 'Save failed';
		} finally {
			busy = false;
		}
	}

	const field = fieldClass;
</script>

<Dialog
	bind:open
	title={cultivar ? 'Edit cultivar' : 'New cultivar'}
	description="A strain or variety you can bind to plants. Fields adapt to the species."
>
	<div class="space-y-4">
		{#if err}<p class="text-xs text-danger">{err}</p>{/if}

		<div class="flex gap-4">
			<!-- Image -->
			<div class="shrink-0">
				<span class="text-xs text-rig-400">Image</span>
				<div class="relative mt-1 h-24 w-24 overflow-hidden rounded-lg border border-rig-700 bg-rig-950">
					{#if previewSrc}
						<img src={previewSrc} alt={name} class="h-full w-full object-cover" />
						<button
							type="button"
							onclick={clearImage}
							aria-label="Remove image"
							class="absolute right-1 top-1 rounded-full bg-rig-950/80 p-1 text-rig-300 hover:text-white"
						>
							<X size={13} />
						</button>
					{:else}
						<label class="flex h-full w-full cursor-pointer flex-col items-center justify-center gap-1 text-rig-500 hover:text-rig-300">
							<ImagePlus size={20} />
							<span class="text-[10px]">Upload</span>
							<input type="file" accept="image/*" class="hidden" onchange={onFile} />
						</label>
					{/if}
				</div>
			</div>

			<!-- Core fields -->
			<div class="flex-1 space-y-3">
				<label class="block">
					<span class="text-xs text-rig-400">Name</span>
					<input bind:value={name} placeholder="e.g. Pink Gelato" class="{field} mt-1" />
				</label>
				<label class="block">
					<span class="text-xs text-rig-400">Species</span>
					<Select
						class="mt-1"
						bind:value={speciesId}
						placeholder="Select a species…"
						items={species.map((sp) => ({ value: sp.id, label: sp.label }))}
					/>
				</label>
			</div>
		</div>

		<label class="block">
			<span class="text-xs text-rig-400">Creator <span class="text-rig-600">(breeder)</span></span>
			<input bind:value={creator} placeholder="e.g. Cookies Fam" class="{field} mt-1" />
		</label>

		<!-- Species-specific attributes, rendered from the species schema -->
		{#if attrSchema.length}
			<AttributeFields schema={attrSchema} bind:values={attributes} />
		{:else if speciesId}
			<p class="text-xs text-rig-500">This species has no extra attributes.</p>
		{/if}

		<label class="block">
			<span class="text-xs text-rig-400">Description</span>
			<textarea bind:value={description} rows="3" class="{field} mt-1"></textarea>
		</label>

		<div class="flex justify-end gap-2 border-t border-rig-800 pt-4">
			<Button variant="ghost" onclick={() => (open = false)}>Cancel</Button>
			<Button onclick={save} disabled={busy || !canSave}>Save</Button>
		</div>
	</div>
</Dialog>
