<script lang="ts">
	import { onMount } from 'svelte';
	import Blocks from '@lucide/svelte/icons/blocks';
	import Plus from '@lucide/svelte/icons/plus';
	import Pencil from '@lucide/svelte/icons/pencil';
	import PlugZap from '@lucide/svelte/icons/plug-zap';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import Link2 from '@lucide/svelte/icons/link-2';
	import { Dialog, Switch } from '$lib/components/ui';
	import { CORE_URL, getAuthToken } from '$lib/api';
	import {
		getIntegrationBundles, getIntegrationInstances, createIntegrationInstance,
		updateIntegrationInstance, deleteIntegrationInstance, testIntegrationInstance,
		getIntegrationBindings, saveIntegrationBinding, deleteIntegrationBinding,
		getGrows
	} from '$lib/api';
	import type { IntegrationBundle, IntegrationInstance, IntegrationBinding, Grow } from '$lib/types';

	let bundles = $state<IntegrationBundle[]>([]);
	let instances = $state<IntegrationInstance[]>([]);
	let bindings = $state<IntegrationBinding[]>([]);
	let grows = $state<Grow[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let modalOpen = $state(false);
	let editing = $state<IntegrationInstance | null>(null);
	let selected = $state<IntegrationBundle | null>(null);
	let formName = $state('');
	let formConfig = $state<Record<string, string>>({});
	let saving = $state(false);
	let testing = $state<string | null>(null);
	let bindFeature = $state('grow-assistant');
	let bindCapability = $state('ai.chat');
	let bindInstance = $state('');
	let bindGrow = $state('');
	let bindingSaving = $state(false);

	const fieldClass = 'mt-1 w-full rounded-md border border-rig-700 bg-rig-950 px-3 py-2 text-sm text-rig-100 outline-none focus:border-rig-500';
	const features = [
		{ value: 'grow-assistant', label: 'Grow assistant', capability: 'ai.chat' },
		{ value: 'camera-analysis', label: 'Camera analysis', capability: 'ai.vision' },
		{ value: 'critical-alerts', label: 'Critical alerts', capability: 'notification.send' },
		{ value: 'daily-summary', label: 'Daily summary', capability: 'notification.send' },
		{ value: 'weather-context', label: 'Weather context', capability: 'weather.forecast' }
	];

	onMount(load);
	async function load() {
		loading = true; error = null;
		try { [bundles, instances, bindings, grows] = await Promise.all([getIntegrationBundles(), getIntegrationInstances(), getIntegrationBindings(), getGrows()]); }
		catch (e) { error = e instanceof Error ? e.message : 'Failed to load integrations'; }
		finally { loading = false; }
	}
	function bundle(id: string) { return bundles.find((b) => b.id === id); }
	function iconURL(b: IntegrationBundle) {
		const token = getAuthToken();
		const path = b.icon ?? '';
		return `${CORE_URL}${path}${token ? `${path.includes('?') ? '&' : '?'}token=${encodeURIComponent(token)}` : ''}`;
	}
	function openCreate(b: IntegrationBundle) { editing = null; selected = b; formName = b.name; formConfig = Object.fromEntries(b.config.map((f) => [f.key, f.default ?? ''])); modalOpen = true; error = null; }
	function openEdit(i: IntegrationInstance) { editing = i; selected = bundle(i.bundleId) ?? null; formName = i.name; formConfig = { ...i.config }; modalOpen = true; error = null; }
	async function save() {
		if (!selected) return; saving = true; error = null;
		try {
			if (editing) await updateIntegrationInstance(editing.id, { name: formName, config: formConfig });
			else await createIntegrationInstance({ bundleId: selected.id, name: formName, config: formConfig });
			modalOpen = false; await load();
		} catch (e) { error = e instanceof Error ? e.message : 'Failed to save integration'; }
		finally { saving = false; }
	}
	async function toggle(i: IntegrationInstance, enabled: boolean) { try { await updateIntegrationInstance(i.id, { enabled, config: {} }); await load(); } catch (e) { error = e instanceof Error ? e.message : 'Failed to update integration'; } }
	async function test(i: IntegrationInstance) { testing = i.id; error = null; try { await testIntegrationInstance(i.id); } catch (e) { error = e instanceof Error ? e.message : 'Connection test failed'; } finally { testing = null; await load(); } }
	async function remove(i: IntegrationInstance) { if (!confirm(`Remove “${i.name}”? Feature bindings using it will also be removed.`)) return; try { await deleteIntegrationInstance(i.id); await load(); } catch (e) { error = e instanceof Error ? e.message : 'Failed to remove integration'; } }
	function capable(capability: string) { return instances.filter((i) => i.enabled && bundle(i.bundleId)?.capabilities.includes(capability)); }
	function selectFeature(value: string) { bindFeature = value; bindCapability = features.find((f) => f.value === value)?.capability ?? ''; bindInstance = capable(bindCapability)[0]?.id ?? ''; }
	async function addBinding() { if (!bindInstance) return; bindingSaving = true; try { await saveIntegrationBinding({ feature: bindFeature, growId: bindGrow, capability: bindCapability, instanceId: bindInstance }); await load(); } catch (e) { error = e instanceof Error ? e.message : 'Failed to save binding'; } finally { bindingSaving = false; } }
	async function removeBinding(id: string) { await deleteIntegrationBinding(id); await load(); }
	function instanceName(id: string) { return instances.find((i) => i.id === id)?.name ?? 'Missing instance'; }
	function growName(id?: string) { return id ? (grows.find((g) => g.id === id)?.name ?? id) : 'All grows'; }
	function statusClass(status: string) { return status === 'healthy' ? 'bg-leaf/15 text-leaf' : status === 'error' ? 'bg-danger/15 text-danger' : 'bg-rig-800 text-rig-400'; }
</script>

<div class="space-y-8">
	<div>
		<h2 class="text-lg font-semibold">Configured integrations</h2>
		<p class="mt-1 text-sm text-rig-400">External services available to GrowRig features. Hardware remains under Devices.</p>
	</div>
	{#if error}<div class="rounded-lg bg-danger/15 px-4 py-2 text-sm text-danger">{error}</div>{/if}
	{#if loading}<p class="text-sm text-rig-400">Loading integrations…</p>
	{:else if instances.length === 0}<div class="rounded-xl border border-dashed border-rig-700 p-8 text-center"><Blocks class="mx-auto text-rig-500" size={28}/><p class="mt-2 font-medium">No integrations configured</p><p class="text-sm text-rig-400">Choose one from the available bundles below.</p></div>
	{:else}
		<div class="grid gap-3 lg:grid-cols-2">
			{#each instances as instance (instance.id)}
				{@const b = bundle(instance.bundleId)}
				<div class="rounded-xl border border-rig-800 bg-rig-900/40 p-4">
					<div class="flex items-start gap-3">
						{#if b?.icon}<img src={iconURL(b)} alt="" class="h-10 w-10 rounded-lg" />{/if}
						<div class="min-w-0 flex-1"><div class="flex items-center gap-2"><h3 class="font-medium">{instance.name}</h3><span class={`rounded-full px-2 py-0.5 text-[11px] capitalize ${statusClass(instance.status)}`}>{instance.status}</span></div><p class="text-xs text-rig-400">{b?.name ?? instance.bundleId} · {b?.capabilities.join(', ')}</p>{#if instance.statusMessage}<p class="mt-1 truncate text-xs text-rig-500">{instance.statusMessage}</p>{/if}</div>
						<Switch checked={instance.enabled} onCheckedChange={(v) => toggle(instance, v)} />
					</div>
					<div class="mt-4 flex gap-2 border-t border-rig-800 pt-3"><button onclick={() => test(instance)} disabled={!instance.enabled || testing === instance.id} class="flex items-center gap-1.5 rounded-md bg-rig-800 px-3 py-1.5 text-xs hover:bg-rig-700 disabled:opacity-50"><PlugZap size={14}/>{testing === instance.id ? 'Testing…' : 'Test connection'}</button><button onclick={() => openEdit(instance)} class="flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs text-rig-300 hover:bg-rig-800"><Pencil size={14}/>Configure</button><button onclick={() => remove(instance)} class="ml-auto rounded-md p-1.5 text-rig-500 hover:bg-danger/10 hover:text-danger" aria-label="Remove"><Trash2 size={15}/></button></div>
				</div>
			{/each}
		</div>
	{/if}

	<section class="space-y-3"><div><h2 class="text-lg font-semibold">Available integrations</h2><p class="text-sm text-rig-400">Create as many independently configured instances as you need.</p></div><div class="grid gap-3 md:grid-cols-2 xl:grid-cols-3">{#each bundles as b (b.id)}<button onclick={() => openCreate(b)} class="group rounded-xl border border-rig-800 bg-rig-900/30 p-4 text-left transition hover:border-rig-600 hover:bg-rig-900"><div class="flex items-start gap-3">{#if b.icon}<img src={iconURL(b)} alt="" class="h-11 w-11 rounded-lg" />{/if}<div><div class="flex items-center gap-2"><h3 class="font-medium">{b.name}</h3><span class="rounded bg-rig-800 px-1.5 py-0.5 text-[10px] uppercase text-rig-400">{b.category}</span></div><p class="mt-1 text-sm text-rig-400">{b.description}</p><div class="mt-3 flex items-center gap-1 text-xs text-leaf"><Plus size={13}/> Add instance</div></div></div></button>{/each}</div></section>

	<section class="space-y-3"><div><h2 class="flex items-center gap-2 text-lg font-semibold"><Link2 size={18}/>Feature bindings</h2><p class="text-sm text-rig-400">Features select a configured instance; grow-specific choices override the all-grows default.</p></div><div class="rounded-xl border border-rig-800 bg-rig-900/30 p-4"><div class="grid gap-3 md:grid-cols-[1fr_1fr_1.4fr_auto]"><label><span class="text-xs text-rig-400">Feature</span><select class={fieldClass} value={bindFeature} onchange={(e) => selectFeature(e.currentTarget.value)}>{#each features as f}<option value={f.value}>{f.label}</option>{/each}</select></label><label><span class="text-xs text-rig-400">Scope</span><select class={fieldClass} bind:value={bindGrow}><option value="">All grows (default)</option>{#each grows as grow}<option value={grow.id}>{grow.name}</option>{/each}</select></label><label><span class="text-xs text-rig-400">Integration instance ({bindCapability})</span><select class={fieldClass} bind:value={bindInstance}><option value="">Choose an instance</option>{#each capable(bindCapability) as i}<option value={i.id}>{i.name}</option>{/each}</select></label><button onclick={addBinding} disabled={!bindInstance || bindingSaving} class="mt-5 rounded-md bg-rig-500 px-4 py-2 text-sm font-medium text-rig-950 disabled:opacity-50">Bind</button></div>{#if bindings.length}<div class="mt-4 divide-y divide-rig-800 border-t border-rig-800">{#each bindings as binding (binding.id)}<div class="flex items-center gap-3 py-3 text-sm"><span class="font-medium">{features.find((f) => f.value === binding.feature)?.label ?? binding.feature}</span><span class="rounded bg-rig-800 px-1.5 py-0.5 text-[11px] text-rig-400">{growName(binding.growId)}</span><span class="text-rig-500">→</span><span>{instanceName(binding.instanceId)}</span><span class="text-xs text-rig-500">{binding.capability}</span><button onclick={() => removeBinding(binding.id)} class="ml-auto text-rig-500 hover:text-danger" aria-label="Remove binding"><Trash2 size={14}/></button></div>{/each}</div>{/if}</div></section>
</div>

<Dialog bind:open={modalOpen} title={editing ? `Configure ${editing.name}` : `Add ${selected?.name ?? 'integration'}`} description="Credentials are encrypted and never returned after saving." size="xl">
	{#if selected}<form onsubmit={(e) => { e.preventDefault(); save(); }} class="space-y-4"><label class="block"><span class="text-sm text-rig-300">Instance name</span><input class={fieldClass} bind:value={formName} required /></label>{#each selected.config as f (f.key)}<label class="block"><span class="text-sm text-rig-300">{f.label}{f.required ? ' *' : ''}</span>{#if f.type === 'select'}<select class={fieldClass} bind:value={formConfig[f.key]}>{#each f.options ?? [] as option}<option value={option}>{option}</option>{/each}</select>{:else}<input class={fieldClass} type={f.type === 'password' ? 'password' : f.type} bind:value={formConfig[f.key]} placeholder={editing && f.secret && editing.secretFields?.includes(f.key) ? 'Saved — leave blank to keep' : f.placeholder} required={f.required && !(editing && editing.secretFields?.includes(f.key))} />{/if}{#if f.help}<span class="mt-1 block text-xs text-rig-500">{f.help}</span>{/if}</label>{/each}<div class="flex justify-end gap-2 border-t border-rig-800 pt-4"><button type="button" onclick={() => modalOpen = false} class="rounded-md px-4 py-2 text-sm text-rig-300 hover:bg-rig-800">Cancel</button><button type="submit" disabled={saving} class="rounded-md bg-rig-500 px-4 py-2 text-sm font-medium text-rig-950 disabled:opacity-50">{saving ? 'Saving…' : editing ? 'Save changes' : 'Create instance'}</button></div></form>{/if}
</Dialog>
