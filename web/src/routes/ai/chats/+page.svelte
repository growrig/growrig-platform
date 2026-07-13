<script lang="ts">
	import { onMount } from 'svelte';
	import { getAIChats, setAIChatArchived } from '$lib/api';
	import type { AIChat } from '$lib/api';
	import { fmtDateTime } from '$lib/datetime';
	import MessagesSquare from '@lucide/svelte/icons/messages-square';
	import Archive from '@lucide/svelte/icons/archive';
	import ArchiveRestore from '@lucide/svelte/icons/archive-restore';
	import ExternalLink from '@lucide/svelte/icons/external-link';

	let chats = $state<AIChat[]>([]);
	let loading = $state(true);
	let error = $state('');
	let view = $state<'active' | 'archived'>('active');
	let updating = $state('');
	let activeCount = $derived(chats.filter((chat) => !chat.archived).length);
	let archivedCount = $derived(chats.filter((chat) => chat.archived).length);
	let visibleChats = $derived(chats.filter((chat) => chat.archived === (view === 'archived')));

	onMount(load);

	async function load() {
		loading = true;
		error = '';
		try {
			chats = await getAIChats();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Chats could not be loaded.';
		} finally {
			loading = false;
		}
	}

	async function toggleArchived(chat: AIChat) {
		updating = chat.id;
		error = '';
		try {
			const updated = await setAIChatArchived(chat.id, !chat.archived);
			chats = chats.map((item) => item.id === chat.id ? { ...item, ...updated } : item);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Chat could not be updated.';
		} finally {
			updating = '';
		}
	}

	function contextName(chat: AIChat) {
		if (chat.growId) return `Grow · ${chat.growName || 'Deleted grow'}`;
		if (chat.environmentId) return `Environment · ${chat.environmentName || 'Deleted environment'}`;
		return 'All GrowRig';
	}
</script>

<div class="space-y-5">
	<div class="flex flex-wrap items-end justify-between gap-3">
		<div>
			<h1 class="flex items-center gap-2 text-2xl font-semibold"><MessagesSquare size={23} class="text-leaf" /> AI Chats</h1>
			<p class="mt-1 text-sm text-rig-400">Your grow-scoped conversations, including archived history.</p>
		</div>
		<div class="flex rounded-lg border border-rig-800 bg-rig-900 p-1 text-sm">
			<button onclick={() => view = 'active'} class="rounded-md px-3 py-1.5 transition {view === 'active' ? 'bg-rig-700 text-rig-50' : 'text-rig-400 hover:text-rig-100'}">Active <span class="ml-1 text-xs opacity-70">{activeCount}</span></button>
			<button onclick={() => view = 'archived'} class="rounded-md px-3 py-1.5 transition {view === 'archived' ? 'bg-rig-700 text-rig-50' : 'text-rig-400 hover:text-rig-100'}">Archived <span class="ml-1 text-xs opacity-70">{archivedCount}</span></button>
		</div>
	</div>

	{#if error}<div class="rounded-lg border border-danger/30 bg-danger/5 px-4 py-3 text-sm text-danger">{error}</div>{/if}

	{#if loading}
		<div class="rounded-xl border border-rig-800 bg-rig-900/40 p-8 text-center text-sm text-rig-500">Loading conversations…</div>
	{:else if visibleChats.length === 0}
		<div class="rounded-xl border border-dashed border-rig-700 p-10 text-center">
			<MessagesSquare size={28} class="mx-auto mb-3 text-rig-600" />
			<p class="font-medium text-rig-300">No {view} chats</p>
			<p class="mt-1 text-sm text-rig-500">{view === 'active' ? 'Open a grow and use the Ask GrowRig button to start one.' : 'Archived conversations will remain available here.'}</p>
		</div>
	{:else}
		<div class="overflow-x-auto rounded-xl border border-rig-800 bg-rig-900/30">
			<table class="w-full min-w-[700px] text-left text-sm">
				<thead class="border-b border-rig-800 bg-rig-900/70 text-xs uppercase tracking-wide text-rig-500">
					<tr><th class="px-4 py-3">Conversation</th><th class="px-4 py-3">Context</th><th class="px-4 py-3">Provider</th><th class="px-4 py-3">Updated</th><th class="px-4 py-3 text-right">Actions</th></tr>
				</thead>
				<tbody class="divide-y divide-rig-800">
					{#each visibleChats as chat (chat.id)}
						<tr class="transition hover:bg-rig-800/25">
							<td class="max-w-md px-4 py-3">
								<div class="font-medium text-rig-100">{chat.title}</div>
								<div class="mt-0.5 truncate text-xs text-rig-500">{chat.preview || 'No messages'}</div>
							</td>
							<td class="px-4 py-3 text-rig-300">{contextName(chat)}</td>
							<td class="px-4 py-3"><span class="rounded-full bg-rig-800 px-2 py-1 text-xs text-rig-400">{chat.instanceName || 'Unavailable'}</span></td>
							<td class="whitespace-nowrap px-4 py-3 text-rig-400"><div>{fmtDateTime(chat.updatedAt)}</div><div class="mt-0.5 text-xs text-rig-600">{chat.messageCount} messages</div></td>
							<td class="px-4 py-3">
								<div class="flex justify-end gap-2">
									<a href={`/ai/chats?chat=${encodeURIComponent(chat.id)}`} class="inline-flex items-center gap-1.5 rounded-md bg-rig-700 px-3 py-2 text-xs font-medium text-rig-100 hover:bg-rig-600">Open <ExternalLink size={13} /></a>
									<button onclick={() => toggleArchived(chat)} disabled={updating === chat.id} class="inline-flex items-center gap-1.5 rounded-md border border-rig-700 px-3 py-2 text-xs text-rig-300 hover:border-rig-500 hover:text-rig-100 disabled:opacity-40">
										{#if chat.archived}<ArchiveRestore size={13} /> Restore{:else}<Archive size={13} /> Archive{/if}
									</button>
								</div>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>
