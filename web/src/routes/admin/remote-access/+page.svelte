<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { errMsg } from '$lib/errors';
	import { getTailscale, enableTailscale, disableTailscale } from '$lib/api';
	import type { TailscaleStatus } from '$lib/types';
	import { Switch } from '$lib/components/ui';
	import Globe from '@lucide/svelte/icons/globe';
	import ExternalLink from '@lucide/svelte/icons/external-link';
	import Copy from '@lucide/svelte/icons/copy';
	import Check from '@lucide/svelte/icons/check';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';
	import CircleCheck from '@lucide/svelte/icons/circle-check';
	import LoaderCircle from '@lucide/svelte/icons/loader-circle';

	let status = $state<TailscaleStatus | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let busy = $state(false);

	// Editable options (seeded from status once loaded).
	let hostname = $state('growrig');
	let controlUrl = $state('');
	let showAdvanced = $state(false);
	let seeded = false;

	let copied = $state<string | null>(null);
	let poll: ReturnType<typeof setInterval> | undefined;

	onMount(() => {
		load();
		return () => clearInterval(poll);
	});
	onDestroy(() => clearInterval(poll));

	async function load() {
		try {
			const s = await getTailscale();
			apply(s);
			error = null;
		} catch (e) {
			error = errMsg(e, 'Failed to load remote access status');
		} finally {
			loading = false;
		}
	}

	function apply(s: TailscaleStatus) {
		status = s;
		if (!seeded) {
			hostname = s.hostname || 'growrig';
			controlUrl = s.controlUrl || '';
			seeded = true;
		}
		// Poll while the node is coming up or awaiting authorization, so the auth
		// link and remote URL appear without a manual refresh.
		const transient = s.enabled && (s.state === 'starting' || s.state === 'needs-login');
		clearInterval(poll);
		if (transient) poll = setInterval(load, 2000);
	}

	async function enable() {
		busy = true;
		error = null;
		try {
			apply(await enableTailscale(hostname, controlUrl));
		} catch (e) {
			error = errMsg(e, 'Failed to enable remote access');
		} finally {
			busy = false;
		}
	}

	async function disable() {
		busy = true;
		error = null;
		try {
			apply(await disableTailscale());
		} catch (e) {
			error = errMsg(e, 'Failed to disable remote access');
		} finally {
			busy = false;
		}
	}

	async function copy(text: string, key: string) {
		try {
			await navigator.clipboard.writeText(text);
			copied = key;
			setTimeout(() => (copied === key ? (copied = null) : null), 1500);
		} catch {
			/* clipboard blocked — the link is still selectable */
		}
	}

	function onToggle(on: boolean) {
		if (busy) return;
		if (on) enable();
		else disable();
	}

	// Key-expiry warning: the node becomes unreachable once its device key
	// expires. Warn when expired or within two weeks.
	const expiryDays = $derived(
		status?.keyExpiry ? Math.floor((new Date(status.keyExpiry).getTime() - Date.now()) / 86_400_000) : null
	);
	const expirySoon = $derived(status?.keyExpired || (expiryDays !== null && expiryDays <= 14));

	const stateLabel: Record<string, string> = {
		stopped: 'Off',
		starting: 'Starting…',
		'needs-login': 'Waiting for authorization',
		running: 'Connected',
		error: 'Error'
	};
	const stateDot: Record<string, string> = {
		stopped: 'bg-rig-600',
		starting: 'bg-warn',
		'needs-login': 'bg-warn',
		running: 'bg-leaf',
		error: 'bg-danger'
	};
</script>

<div class="space-y-6">
	<div>
		<h2 class="flex items-center gap-2 text-lg font-semibold"><Globe size={20} class="text-rig-400" /> Remote access</h2>
		<p class="mt-1 text-sm text-rig-400">
			Reach GrowRig from anywhere over <span class="text-rig-200">Tailscale</span> — private, encrypted, no
			port forwarding. Your phone or laptop needs the Tailscale app connected to the same tailnet. LAN access keeps working alongside it.
		</p>
	</div>

	{#if error}
		<div class="rounded-lg bg-danger/15 px-4 py-2 text-sm text-danger">{error}</div>
	{/if}

	{#if loading}
		<p class="text-sm text-rig-400">Loading…</p>
	{:else if status && !status.available}
		<div class="rounded-xl border border-dashed border-rig-800 p-5 text-sm text-rig-400">
			<p class="mb-1 font-medium text-rig-200">Remote access unavailable</p>
			Tailscale support isn't included in this build.
		</div>
	{:else if status}
		<!-- Enable / status card -->
		<div class="rounded-xl border border-rig-800 bg-rig-900/40 p-5">
			<div class="flex items-center justify-between gap-4">
				<div class="flex items-center gap-3">
					<span class="h-2.5 w-2.5 rounded-full {stateDot[status.state] ?? 'bg-rig-600'}"></span>
					<div>
						<div class="font-medium">Tailscale</div>
						<div class="text-sm text-rig-400">{stateLabel[status.state] ?? status.state}</div>
					</div>
				</div>
				<Switch checked={status.enabled} disabled={busy} onCheckedChange={onToggle} aria-label="Enable remote access" />
			</div>

			{#if status.error}
				<div class="mt-4 flex items-start gap-2 rounded-lg border border-danger/40 bg-danger/10 px-3 py-2 text-sm text-danger">
					<TriangleAlert size={16} class="mt-0.5 shrink-0" /> {status.error}
				</div>
			{/if}

			{#if status.state === 'starting'}
				<p class="mt-4 flex items-center gap-2 text-sm text-rig-400"><LoaderCircle size={15} class="animate-spin" /> Bringing the tailnet node up…</p>
			{/if}

			<!-- Authorization step -->
			{#if status.state === 'needs-login' && status.authUrl}
				<div class="mt-4 space-y-2 rounded-lg border border-warn/40 bg-warn/10 p-4">
					<p class="text-sm font-medium text-warn">Authorize GrowRig into your tailnet</p>
					<p class="text-xs text-rig-300">Open this link to sign in with your Tailscale account and approve this device. You only do this once.</p>
					<div class="flex items-center gap-2">
						<a href={status.authUrl} target="_blank" rel="noopener noreferrer" class="flex min-w-0 flex-1 items-center gap-1.5 truncate rounded-md border border-rig-700 bg-rig-950 px-3 py-2 text-sm text-leaf hover:border-leaf">
							<ExternalLink size={14} class="shrink-0" /> <span class="truncate">{status.authUrl}</span>
						</a>
						<button onclick={() => copy(status!.authUrl!, 'auth')} class="rounded-md border border-rig-700 p-2 text-rig-400 hover:border-leaf hover:text-rig-100" aria-label="Copy link">
							{#if copied === 'auth'}<Check size={15} class="text-leaf" />{:else}<Copy size={15} />{/if}
						</button>
					</div>
				</div>
			{/if}

			<!-- Connected -->
			{#if status.state === 'running' && status.url}
				<div class="mt-4 space-y-2 rounded-lg border border-leaf/30 bg-leaf/10 p-4">
					<p class="flex items-center gap-1.5 text-sm font-medium text-leaf"><CircleCheck size={15} /> Reachable on your tailnet</p>
					<div class="flex items-center gap-2">
						<a href={status.url} target="_blank" rel="noopener noreferrer" class="flex min-w-0 flex-1 items-center gap-1.5 truncate rounded-md border border-rig-700 bg-rig-950 px-3 py-2 text-sm text-rig-100 hover:border-leaf">
							<ExternalLink size={14} class="shrink-0" /> <span class="truncate">{status.url}</span>
						</a>
						<button onclick={() => copy(status!.url!, 'url')} class="rounded-md border border-rig-700 p-2 text-rig-400 hover:border-leaf hover:text-rig-100" aria-label="Copy URL">
							{#if copied === 'url'}<Check size={15} class="text-leaf" />{:else}<Copy size={15} />{/if}
						</button>
					</div>
					<p class="text-xs text-rig-500">Any device on your tailnet with Tailscale connected can open this URL.</p>
				</div>
			{/if}

			<!-- Key-expiry warning -->
			{#if status.enabled && expirySoon}
				<div class="mt-4 flex items-start gap-2 rounded-lg border border-warn/40 bg-warn/10 px-3 py-2 text-sm text-warn">
					<TriangleAlert size={16} class="mt-0.5 shrink-0" />
					<div>
						{#if status.keyExpired}
							This device's Tailscale key has <span class="font-medium">expired</span> — GrowRig is unreachable remotely until you reauthorize it.
						{:else}
							This device's Tailscale key expires in {expiryDays} day{expiryDays === 1 ? '' : 's'}. When it does, remote access stops until reauthorized.
						{/if}
						<a href="https://tailscale.com/kb/1028/key-expiry" target="_blank" rel="noopener noreferrer" class="underline hover:text-rig-100">Disable key expiry for this server</a> in the Tailscale admin console to keep it always reachable.
					</div>
				</div>
			{/if}
		</div>

		<!-- Options -->
		<div class="rounded-xl border border-rig-800 bg-rig-900/40 p-5">
			<h3 class="mb-3 text-sm font-semibold text-rig-300">Options</h3>
			<label class="block">
				<span class="text-xs text-rig-400">Hostname</span>
				<input
					bind:value={hostname}
					disabled={status.enabled || busy}
					placeholder="growrig"
					class="mt-1 w-full max-w-xs rounded-md border border-rig-700 bg-rig-950 px-3 py-1.5 text-sm focus:border-leaf focus:outline-none disabled:opacity-50"
				/>
				<span class="mt-1 block text-xs text-rig-500">Becomes the device name on your tailnet (letters, digits, hyphens). Change it while remote access is off.</span>
			</label>

			<button onclick={() => (showAdvanced = !showAdvanced)} class="mt-4 text-xs text-rig-400 hover:text-rig-100">
				{showAdvanced ? '− Hide' : '+ Show'} advanced
			</button>
			{#if showAdvanced}
				<label class="mt-3 block">
					<span class="text-xs text-rig-400">Control server URL <span class="text-rig-600">(optional)</span></span>
					<input
						bind:value={controlUrl}
						disabled={status.enabled || busy}
						placeholder="https://controlplane.tailscale.com"
						class="mt-1 w-full max-w-md rounded-md border border-rig-700 bg-rig-950 px-3 py-1.5 text-sm focus:border-leaf focus:outline-none disabled:opacity-50"
					/>
					<span class="mt-1 block text-xs text-rig-500">Leave blank for Tailscale's default coordination server. Point at a self-hosted control plane (e.g. Headscale) if you run one.</span>
				</label>
			{/if}
		</div>
	{/if}
</div>
