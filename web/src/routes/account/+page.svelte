<script lang="ts">
	import { errMsg } from '$lib/errors';
	import { onMount } from 'svelte';
	import { auth } from '$lib/auth.svelte';
	import { getPasskeys, deletePasskey, type Passkey } from '$lib/api';
	import { passkeysSupported, registerPasskey } from '$lib/webauthn';
	import { theme, type Theme } from '$lib/theme.svelte';
	import KeyRound from '@lucide/svelte/icons/key-round';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import Plus from '@lucide/svelte/icons/plus';
	import Shield from '@lucide/svelte/icons/shield';
	import Palette from '@lucide/svelte/icons/palette';
	import Monitor from '@lucide/svelte/icons/monitor';
	import Sun from '@lucide/svelte/icons/sun';
	import Moon from '@lucide/svelte/icons/moon';
	import { fmtDate } from '$lib/datetime';

	const themeOptions: { value: Theme; label: string; icon: typeof Monitor }[] = [
		{ value: 'system', label: 'System', icon: Monitor },
		{ value: 'light', label: 'Light', icon: Sun },
		{ value: 'dark', label: 'Dark', icon: Moon }
	];

	let passkeys = $state<Passkey[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	let adding = $state(false);
	let newName = $state('');
	let registering = $state(false);

	onMount(load);

	async function load() {
		loading = true;
		try {
			passkeys = await getPasskeys();
		} catch (e) {
			error = errMsg(e, 'Failed to load passkeys');
		} finally {
			loading = false;
		}
	}

	async function add() {
		registering = true;
		error = null;
		try {
			await registerPasskey(newName.trim() || 'Passkey');
			adding = false;
			newName = '';
			await load();
		} catch (e) {
			const msg = errMsg(e, 'Failed to add passkey');
			if (!/cancel|abort|not allowed/i.test(msg)) error = msg;
		} finally {
			registering = false;
		}
	}

	async function remove(p: Passkey) {
		if (!confirm(`Remove passkey "${p.name}"?`)) return;
		error = null;
		try {
			await deletePasskey(p.id);
			await load();
		} catch (e) {
			error = errMsg(e, 'Failed to remove passkey');
		}
	}

	const field =
		'rounded-md border border-rig-700 bg-rig-950 px-3 py-1.5 text-sm focus:border-rig-500 focus:outline-none';
</script>

<div class="mx-auto max-w-2xl space-y-6">
	<div>
		<h1 class="text-2xl font-semibold">Account</h1>
		<p class="mt-1 flex items-center gap-1.5 text-sm text-rig-400">
			{#if auth.isAdmin}<Shield size={14} class="text-leaf" />{/if}
			Signed in as <span class="font-medium text-rig-200">{auth.user?.username}</span>
			· <span class="capitalize">{auth.user?.role}</span>
		</p>
	</div>

	{#if error}
		<div class="rounded-lg bg-danger/15 px-4 py-2 text-sm text-danger">{error}</div>
	{/if}

	<section class="rounded-xl border border-rig-800 bg-rig-900/40 p-5">
		<h2 class="mb-1 flex items-center gap-2 font-medium"><Palette size={17} /> Appearance</h2>
		<p class="mb-4 text-sm text-rig-400">
			Choose a theme. <span class="font-medium text-rig-200">System</span> follows your device's light or dark setting.
		</p>
		<div class="grid grid-cols-3 gap-2 sm:max-w-md" role="radiogroup" aria-label="Theme">
			{#each themeOptions as opt (opt.value)}
				{@const selected = theme.preference === opt.value}
				<button
					role="radio"
					aria-checked={selected}
					onclick={() => theme.set(opt.value)}
					class="flex flex-col items-center gap-1.5 rounded-lg border px-3 py-3 text-sm transition-colors {selected
						? 'border-rig-500 bg-rig-500/10 text-rig-100'
						: 'border-rig-700 text-rig-300 hover:border-rig-500 hover:text-rig-100'}"
				>
					<opt.icon size={18} />
					{opt.label}
				</button>
			{/each}
		</div>
	</section>

	<section class="rounded-xl border border-rig-800 bg-rig-900/40 p-5">
		<div class="mb-1 flex items-center justify-between">
			<h2 class="flex items-center gap-2 font-medium"><KeyRound size={17} /> Passkeys</h2>
			{#if passkeysSupported() && !adding}
				<button
					onclick={() => { adding = true; error = null; }}
					class="flex items-center gap-1.5 rounded-md bg-rig-500 px-3 py-1.5 text-sm font-medium text-rig-950 transition-colors hover:bg-rig-400"
				>
					<Plus size={15} /> Add a passkey
				</button>
			{/if}
		</div>
		<p class="mb-4 text-sm text-rig-400">
			Sign in without a password using your device's fingerprint, face, PIN, or a security key.
		</p>

		{#if !passkeysSupported()}
			<p class="text-sm text-rig-500">This browser doesn't support passkeys.</p>
		{/if}

		{#if adding}
			<div class="mb-4 flex flex-wrap items-end gap-3 rounded-lg border border-rig-700 bg-rig-950/40 p-3">
				<label class="flex-1">
					<span class="text-sm text-rig-400">Name this passkey <span class="text-rig-600">(e.g. "MacBook", "iPhone")</span></span>
					<input bind:value={newName} placeholder="Passkey" class="{field} mt-1 w-full" />
				</label>
				<div class="flex gap-2">
					<button onclick={() => (adding = false)} class="rounded-md border border-rig-700 px-4 py-1.5 text-sm text-rig-300 hover:border-rig-500">Cancel</button>
					<button
						onclick={add}
						disabled={registering}
						class="rounded-md bg-rig-500 px-4 py-1.5 text-sm font-medium text-rig-950 hover:bg-rig-400 disabled:opacity-50"
					>
						{registering ? 'Waiting…' : 'Create passkey'}
					</button>
				</div>
			</div>
		{/if}

		{#if loading}
			<p class="text-sm text-rig-400">Loading…</p>
		{:else if passkeys.length === 0}
			<p class="text-sm text-rig-500">No passkeys yet.</p>
		{:else}
			<ul class="divide-y divide-rig-800 overflow-hidden rounded-lg border border-rig-800">
				{#each passkeys as p (p.id)}
					<li class="flex items-center justify-between gap-3 bg-rig-950/30 px-4 py-3">
						<div class="flex items-center gap-2.5">
							<KeyRound size={16} class="text-rig-400" />
							<div>
								<div class="text-sm font-medium">{p.name}</div>
								{#if fmtDate(p.created)}<div class="text-xs text-rig-500">Added {fmtDate(p.created)}</div>{/if}
							</div>
						</div>
						<button onclick={() => remove(p)} title="Remove" class="rounded p-1.5 text-rig-400 transition-colors hover:bg-danger/15 hover:text-danger">
							<Trash2 size={15} />
						</button>
					</li>
				{/each}
			</ul>
		{/if}
	</section>
</div>
