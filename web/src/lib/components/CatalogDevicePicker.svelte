<script lang="ts">
	import type { BindingKind, CatalogProduct, DiscoveredEntity, Measurement, Role } from '$lib/types';
	import { Button, Select } from '$lib/components/ui';

	export interface BindingDraft {
		kind: BindingKind;
		name: string;
		entity: string;
		measurement?: Measurement;
		role?: Role;
		rpmEntity?: string;
		wattage?: number;
	}

	interface Props {
		catalog: CatalogProduct[];
		discovered: DiscoveredEntity[];
		usedEntities: Set<string>; // already bound or picked in this session
		onAdd: (drafts: BindingDraft[]) => void;
	}
	let { catalog, discovered, usedEntities, onAdd }: Props = $props();

	const categories: { key: BindingKind; label: string }[] = [
		{ key: 'sensor', label: 'Sensors' },
		{ key: 'fan', label: 'Fans' },
		{ key: 'light', label: 'Lights' },
		{ key: 'camera', label: 'Cameras' }
	];
	// Catalog categories that yield each binding kind.
	const catCategoriesFor: Record<BindingKind, string[]> = {
		sensor: ['sensor'],
		fan: ['fan'],
		light: ['light', 'plug'],
		camera: ['camera']
	};

	let category = $state<BindingKind>('sensor');
	let productId = $state('');

	const products = $derived(catalog.filter((p) => catCategoriesFor[category].includes(p.category)));
	const product = $derived(catalog.find((p) => p.id === productId));

	// Per-provides-row entity selection + name (+ wattage for lights).
	let rowEntity = $state<string[]>([]);
	let rowName = $state<string[]>([]);
	let rowWattage = $state<number[]>([]);

	function selectProduct(id: string) {
		productId = id;
		const p = catalog.find((x) => x.id === id);
		rowEntity = (p?.provides ?? []).map(() => '');
		rowName = (p?.provides ?? []).map((t) => t.label);
		rowWattage = (p?.provides ?? []).map((t) => t.wattage ?? 0);
	}

	function candidates(kind: BindingKind, measurement?: Measurement): DiscoveredEntity[] {
		return discovered.filter(
			(d) =>
				d.kind === kind &&
				(!measurement || d.measurement === measurement) &&
				!usedEntities.has(d.entity)
		);
	}

	function commit() {
		const p = product;
		if (!p?.provides) return;
		const drafts: BindingDraft[] = [];
		p.provides.forEach((t, i) => {
			const entity = rowEntity[i]?.trim();
			if (!entity) return;
			drafts.push({
				kind: t.kind,
				name: rowName[i] || t.label,
				entity,
				measurement: t.measurement,
				role: t.role,
				wattage: t.kind === 'light' ? rowWattage[i] || 0 : undefined
			});
		});
		if (drafts.length === 0) return;
		onAdd(drafts);
		productId = '';
		rowEntity = [];
		rowName = [];
		rowWattage = [];
	}

	const field =
		'rounded-md border border-rig-700 bg-rig-950 px-2 py-1 text-sm focus:border-rig-500 focus:outline-none';
</script>

<div class="space-y-3">
	<div class="flex flex-wrap gap-1">
		{#each categories as c (c.key)}
			<button
				type="button"
				onclick={() => {
					category = c.key;
					productId = '';
				}}
				class="rounded-md px-3 py-1 text-xs transition-colors {category === c.key
					? 'bg-rig-700 text-rig-50'
					: 'text-rig-400 hover:bg-rig-800'}"
			>
				{c.label}
			</button>
		{/each}
	</div>

	<!-- product grid -->
	<div class="grid gap-2 sm:grid-cols-2">
		{#each products as p (p.id)}
			<button
				type="button"
				onclick={() => selectProduct(p.id)}
				class="rounded-lg border p-3 text-left transition-colors {productId === p.id
					? 'border-rig-500 bg-rig-800/40'
					: 'border-rig-800 bg-rig-950/40 hover:border-rig-600'}"
			>
				<div class="text-sm font-medium">{p.brand} {p.model}</div>
				<div class="text-xs text-rig-500">{p.connection}</div>
			</button>
		{/each}
	</div>

	<!-- entity mapping for the selected product -->
	{#if product?.provides}
		<div class="space-y-2 rounded-lg border border-rig-800 bg-rig-950/40 p-3">
			{#if product.description}
				<p class="text-xs text-rig-500">{product.description}</p>
			{/if}
			{#each product.provides as t, i (i)}
				{@const opts = candidates(t.kind, t.measurement)}
				<div class="grid items-center gap-2 sm:grid-cols-[1fr_1.4fr]">
					<input bind:value={rowName[i]} placeholder={t.label} class={field} />
					{#if opts.length}
						<Select
							bind:value={rowEntity[i]}
							placeholder="— select {t.entityDomain} entity —"
							items={opts.map((d) => ({ value: d.entity, label: `${d.name} (${d.entity})` }))}
						/>
					{:else}
						<input bind:value={rowEntity[i]} placeholder="{t.entityDomain}.entity_id" class={field} />
					{/if}
				</div>
				{#if t.kind === 'light'}
					<label class="flex items-center gap-2 pl-1 text-xs text-rig-400">
						Wattage
						<input
							type="number"
							min="0"
							step="1"
							bind:value={rowWattage[i]}
							placeholder="W"
							class="{field} w-24"
						/>
						<span class="text-rig-600">W</span>
					</label>
				{/if}
			{/each}
			<Button type="button" onclick={commit}>
				Add {product.brand} {product.model}
			</Button>
		</div>
	{/if}
</div>
