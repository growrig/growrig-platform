<script lang="ts">
	import type { Binding, BindingKind, CatalogProduct, DiscoveredEntity, Measurement, Role } from '$lib/types';
	import { Button, Select } from '$lib/components/ui';

	export interface BindingDraft {
		deviceId: string;
		deviceName: string;
		powerControllerId?: string;
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
		bindings?: Binding[];
		usedEntities: Set<string>; // already bound or picked in this session
		onAdd: (drafts: BindingDraft[]) => void;
		onSelectProduct?: (product: CatalogProduct) => void;
		initialCategory?: BindingKind;
	}
	let { catalog, discovered, bindings = [], usedEntities, onAdd, onSelectProduct, initialCategory = 'sensor' }: Props = $props();

	const categories: { key: BindingKind; label: string }[] = [
		{ key: 'controller', label: 'Controllers' },
		{ key: 'sensor', label: 'Sensors' },
		{ key: 'fan', label: 'Fans' },
		{ key: 'light', label: 'Lights' },
		{ key: 'power', label: 'Power & Switching' },
		{ key: 'camera', label: 'Cameras' },
		{ key: 'irrigation', label: 'Irrigation' }
	];
	// Catalog categories that yield each binding kind.
	const catCategoriesFor: Record<BindingKind, string[]> = {
		sensor: ['sensor'],
		fan: ['fan'],
		controller: ['controller'],
		light: ['light'],
		power: ['plug'],
		camera: ['camera'],
		irrigation: ['irrigation']
	};

	let category = $state<BindingKind>('sensor');
	let productId = $state('');
	$effect(() => { category = initialCategory; productId = ''; });

	const products = $derived(catalog.filter((p) => catCategoriesFor[category].includes(p.category)));
	const product = $derived(catalog.find((p) => p.id === productId));

	// Per-provides-row entity selection + name (+ wattage for lights).
	let rowEntity = $state<string[]>([]);
	let rowName = $state<string[]>([]);
	let rowWattage = $state<number[]>([]);
	let powerControllerId = $state('');
	const powerControllers = $derived.by(() => {
		const seen = new Map<string, string>();
		for (const b of bindings) if (b.kind === 'power') seen.set(b.deviceId, b.deviceName);
		return [...seen.entries()].map(([value, label]) => ({ value, label }));
	});

	function selectProduct(id: string) {
		productId = id;
		const p = catalog.find((x) => x.id === id);
		rowEntity = (p?.provides ?? []).map(() => '');
		rowName = (p?.provides ?? []).map((t) =>
			t.kind === 'light' && p ? `${p.brand} ${p.model}` : t.label
		);
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
		const deviceId = crypto.randomUUID();
		const deviceName = `${p.brand} ${p.model}`;
		p.provides.forEach((t, i) => {
			const entity = rowEntity[i]?.trim();
			if (!entity && t.kind !== 'light') return;
			drafts.push({
				deviceId,
				deviceName: t.kind === 'light' ? rowName[i]?.trim() || deviceName : deviceName,
				powerControllerId: t.kind === 'light' ? powerControllerId || undefined : undefined,
				kind: t.kind,
				name: rowName[i] || (t.kind === 'light' ? deviceName : t.label),
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
		powerControllerId = '';
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
				onclick={() => onSelectProduct ? onSelectProduct(p) : selectProduct(p.id)}
				class="rounded-lg border p-3 text-left transition-colors {productId === p.id
					? 'border-rig-500 bg-rig-800/40'
					: 'border-rig-800 bg-rig-950/40 hover:border-rig-600'}"
			>
				{#if p.image}<img src={p.image} alt="" class="mb-2 aspect-video w-full rounded-md object-contain" />{/if}
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
					{#if t.kind === 'light'}
						<input bind:value={rowName[i]} placeholder="{product.brand} {product.model}" class={field} />
						<div class="rounded-md border border-rig-800 bg-rig-900/50 px-3 py-2 text-xs text-rig-500">Fixture · no Home Assistant entity</div>
					{:else if opts.length}
						<input bind:value={rowName[i]} placeholder={t.label} class={field} />
						<Select
							bind:value={rowEntity[i]}
							placeholder="— select {t.entityDomain} entity —"
							items={opts.map((d) => ({ value: d.entity, label: `${d.name} (${d.entity})` }))}
						/>
					{:else}
						<input bind:value={rowName[i]} placeholder={t.label} class={field} />
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
			{#if product.provides.some((t) => t.kind === 'light')}
				<label class="block text-xs text-rig-400">Power controller <span class="text-rig-600">(optional)</span>
					<Select bind:value={powerControllerId} placeholder="Assign later" items={powerControllers} class="mt-1" />
				</label>
			{/if}
			<Button type="button" onclick={commit}>
				Add {product.brand} {product.model}
			</Button>
		</div>
	{/if}
</div>
