<script lang="ts">
	import { onDestroy, onMount, tick } from 'svelte';
	import { page } from '$app/state';
	import { replaceState } from '$app/navigation';
	import { getGrowAIStatus, chatWithGrowAI, getAIChat, setAIChatArchived } from '$lib/api';
	import type { GrowAIMessage } from '$lib/api';
	import { marked } from 'marked';
	import DOMPurify from 'dompurify';
	import Sparkles from '@lucide/svelte/icons/sparkles';
	import Send from '@lucide/svelte/icons/send';
	import Bot from '@lucide/svelte/icons/bot';
	import Minus from '@lucide/svelte/icons/minus';
	import MessagesSquare from '@lucide/svelte/icons/messages-square';
	import Plus from '@lucide/svelte/icons/plus';
	import Archive from '@lucide/svelte/icons/archive';
	import ArchiveRestore from '@lucide/svelte/icons/archive-restore';

	interface Props { growId: string; growName: string }
	let { growId, growName }: Props = $props();

	let checked = $state(false);
	let available = $state(false);
	let opened = $state(false);
	let instanceName = $state('');
	let chatID = $state('');
	let chatTitle = $state('');
	let archived = $state(false);
	let archiving = $state(false);
	let messages = $state<GrowAIMessage[]>([]);
	let draft = $state('');
	let sending = $state(false);
	let elapsedSeconds = $state(0);
	let error = $state('');
	let messageList = $state<HTMLDivElement>();
	let timer: ReturnType<typeof setInterval> | undefined;
	let startedAt = 0;
	let waitingLabel = $derived(
		elapsedSeconds < 3
			? 'Preparing grow context'
			: elapsedSeconds < 15
				? 'Ollama is thinking'
				: elapsedSeconds < 30
					? 'Still thinking — local models can take a moment'
					: 'Still working'
	);

	const suggestions = [
		'What changed during the last week?',
		'Are there any patterns in recent care?',
		'How is the current environment looking?'
	];

	onMount(async () => {
		try {
			const status = await getGrowAIStatus(growId);
			available = status.available;
			instanceName = status.instanceName ?? '';
		} catch {
			available = false;
		}
		const requestedChatID = page.url.searchParams.get('chat');
		if (requestedChatID) {
			try {
				const chat = await getAIChat(requestedChatID);
				if (chat.growId !== growId) throw new Error('This chat belongs to a different grow.');
				chatID = chat.id;
				chatTitle = chat.title;
				archived = chat.archived;
				instanceName = chat.instanceName || instanceName;
				messages = chat.messages ?? [];
				opened = true;
			} catch (e) {
				error = e instanceof Error ? e.message : 'The chat could not be loaded.';
			}
		}
		checked = true;
	});

	onDestroy(stopTimer);

	function startTimer() {
		stopTimer();
		startedAt = Date.now();
		elapsedSeconds = 0;
		timer = setInterval(() => {
			elapsedSeconds = Math.floor((Date.now() - startedAt) / 1000);
		}, 250);
	}

	function stopTimer() {
		if (timer) clearInterval(timer);
		timer = undefined;
	}

	function renderMarkdown(content: string) {
		const html = marked.parse(content, { async: false, breaks: true, gfm: true });
		return DOMPurify.sanitize(html);
	}

	async function scrollToLatest() {
		await tick();
		messageList?.scrollTo({ top: messageList.scrollHeight, behavior: 'smooth' });
	}

	async function openChat() {
		opened = true;
		await scrollToLatest();
	}

	function newChat() {
		chatID = '';
		chatTitle = '';
		archived = false;
		messages = [];
		draft = '';
		error = '';
		opened = true;
		replaceState(page.url.pathname, {});
	}

	async function toggleArchived() {
		if (!chatID || archiving) return;
		archiving = true;
		error = '';
		try {
			const chat = await setAIChatArchived(chatID, !archived);
			archived = chat.archived;
		} catch (e) {
			error = e instanceof Error ? e.message : 'The chat could not be updated.';
		} finally {
			archiving = false;
		}
	}

	async function send(content = draft) {
		const question = content.trim();
		if (!question || sending || archived) return;
		const previousMessages = messages;
		messages = [...messages, { role: 'user', content: question }];
		draft = '';
		sending = true;
		startTimer();
		error = '';
		await scrollToLatest();
		try {
			const reply = await chatWithGrowAI(growId, chatID, question);
			messages = [...messages, reply.message];
			chatID = reply.chat.id;
			chatTitle = reply.chat.title;
			archived = reply.chat.archived;
			instanceName = reply.instanceName;
			replaceState(`${page.url.pathname}?chat=${encodeURIComponent(chatID)}`, {});
		} catch (e) {
			messages = previousMessages;
			draft = question;
			error = e instanceof Error ? e.message : 'The assistant could not answer.';
		} finally {
			sending = false;
			stopTimer();
			await scrollToLatest();
		}
	}

	function onKeydown(event: KeyboardEvent) {
		if (event.key === 'Enter' && !event.shiftKey) {
			event.preventDefault();
			send();
		}
	}
</script>

{#if checked && (available || chatID)}
	{#if opened}
		<section aria-label="Ask GrowRig chat" class="chat-window fixed z-50 flex h-[min(42rem,calc(100dvh-1.5rem))] w-[calc(100vw-1.5rem)] flex-col overflow-hidden rounded-2xl border border-rig-600/80 bg-rig-900 shadow-2xl shadow-black/50 sm:h-[min(42rem,calc(100dvh-2.5rem))] sm:w-[28rem]">
			<div class="flex shrink-0 items-center gap-3 border-b border-rig-800 px-4 py-3">
				<div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-leaf/10 text-leaf">
					<Sparkles size={18} />
				</div>
				<div class="min-w-0 flex-1">
					<h2 class="truncate text-sm font-semibold text-rig-200">{chatTitle || 'Ask GrowRig'}</h2>
					<p class="truncate text-xs text-rig-500">{growName} · {instanceName}{archived ? ' · Archived' : ''}</p>
				</div>
				<a href="/ai/chats" aria-label="All AI chats" title="All chats" class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-rig-400 transition hover:bg-rig-800 hover:text-rig-100"><MessagesSquare size={17} /></a>
				{#if chatID}
					{#if available}<button onclick={newChat} aria-label="Start new chat" title="New chat" class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-rig-400 transition hover:bg-rig-800 hover:text-rig-100"><Plus size={18} /></button>{/if}
					<button onclick={toggleArchived} disabled={archiving} aria-label={archived ? 'Restore chat' : 'Archive chat'} title={archived ? 'Restore' : 'Archive'} class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-rig-400 transition hover:bg-rig-800 hover:text-rig-100 disabled:opacity-40">
						{#if archived}<ArchiveRestore size={17} />{:else}<Archive size={17} />{/if}
					</button>
				{/if}
				<button onclick={() => opened = false} aria-label="Minimize chat" title="Minimize" class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-rig-400 transition hover:bg-rig-800 hover:text-rig-100">
					<Minus size={20} />
				</button>
			</div>

			<div bind:this={messageList} class="min-h-0 flex-1 space-y-3 overflow-y-auto p-4" aria-live="polite">
			{#if messages.length === 0}
				<div class="flex gap-3 text-sm text-rig-300">
					<div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-leaf/10 text-leaf"><Bot size={16} /></div>
					<div>
						<p>Ask about this grow’s plants, stage, current environment, recent care, or activity.</p>
						<div class="mt-3 flex flex-wrap gap-2">
							{#each suggestions as suggestion}
								<button onclick={() => send(suggestion)} class="rounded-full border border-rig-700 px-3 py-1.5 text-left text-xs text-rig-400 transition hover:border-rig-500 hover:text-rig-100">{suggestion}</button>
							{/each}
						</div>
					</div>
				</div>
			{:else}
				{#each messages as message, index (`${message.role}-${index}`)}
					<div class={`flex ${message.role === 'user' ? 'justify-end' : 'justify-start'}`}>
						{#if message.role === 'assistant'}
							<div class="markdown max-w-[85%] rounded-xl bg-rig-950 px-3.5 py-2.5 text-sm leading-relaxed text-rig-200">{@html renderMarkdown(message.content)}</div>
						{:else}
							<div class="max-w-[85%] whitespace-pre-wrap rounded-xl bg-rig-700 px-3.5 py-2.5 text-sm leading-relaxed text-rig-50">{message.content}</div>
						{/if}
					</div>
				{/each}
				{#if sending}
					<div class="flex justify-start">
						<div class="rounded-xl bg-rig-950 px-3.5 py-2.5 text-sm text-rig-400" role="status">
							<span class="inline-flex items-center gap-2">
								<span class="h-2 w-2 animate-pulse rounded-full bg-leaf"></span>
								{waitingLabel}… <span class="tabular-nums text-rig-600">{elapsedSeconds}s</span>
							</span>
						</div>
					</div>
				{/if}
			{/if}
			</div>

			{#if error}<p class="mx-4 mb-2 shrink-0 rounded-md bg-danger/10 px-3 py-2 text-xs text-danger">{error}</p>{/if}
			{#if archived}
				<div class="flex shrink-0 items-center justify-between gap-3 border-t border-rig-800 bg-rig-900 p-3">
					<p class="text-xs text-rig-400">This conversation is archived and read-only.</p>
					<button onclick={toggleArchived} disabled={archiving} class="shrink-0 rounded-md bg-rig-700 px-3 py-2 text-xs font-medium text-rig-100 hover:bg-rig-600 disabled:opacity-40">Restore chat</button>
				</div>
			{:else}
			<div class="shrink-0 border-t border-rig-800 bg-rig-900 p-3">
				<div class="flex items-end gap-2">
					<textarea bind:value={draft} onkeydown={onKeydown} rows="2" maxlength="4000" placeholder="Ask about this grow…" class="min-h-11 flex-1 resize-none rounded-lg border border-rig-700 bg-rig-950 px-3 py-2 text-sm outline-none placeholder:text-rig-600 focus:border-rig-500"></textarea>
					<button onclick={() => send()} disabled={sending || !draft.trim()} aria-label="Send message" class="flex h-11 w-11 shrink-0 items-center justify-center rounded-lg bg-rig-500 text-rig-950 transition hover:bg-rig-400 disabled:cursor-not-allowed disabled:opacity-40"><Send size={17} /></button>
				</div>
				<p class="mt-2 text-[11px] text-rig-600">Read-only · Grow context is sent to {instanceName}</p>
			</div>
			{/if}
		</section>
	{:else}
		<button onclick={openChat} aria-label="Open Ask GrowRig chat" title="Ask GrowRig" class="chat-launcher fixed z-50 flex h-16 w-16 items-center justify-center rounded-full border border-leaf/30 bg-rig-500 text-rig-950 shadow-xl shadow-black/40 transition hover:scale-105 hover:bg-rig-400 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-leaf">
			<Sparkles size={27} />
			{#if sending}
				<span class="absolute -right-1 -top-1 flex h-6 min-w-6 items-center justify-center rounded-full border-2 border-rig-900 bg-leaf px-1 text-[10px] font-semibold tabular-nums text-rig-950">{elapsedSeconds}s</span>
			{/if}
		</button>
	{/if}
{/if}

<style>
	.chat-window,
	.chat-launcher {
		right: 0.75rem;
		bottom: max(0.75rem, env(safe-area-inset-bottom));
	}
	@media (min-width: 640px) {
		.chat-window,
		.chat-launcher {
			right: 1.25rem;
			bottom: max(1.25rem, env(safe-area-inset-bottom));
		}
	}
	.markdown :global(p),
	.markdown :global(ul),
	.markdown :global(ol),
	.markdown :global(pre),
	.markdown :global(blockquote),
	.markdown :global(table) {
		margin: 0.7rem 0;
	}
	.markdown :global(:first-child) { margin-top: 0; }
	.markdown :global(:last-child) { margin-bottom: 0; }
	.markdown :global(ul) { list-style: disc; padding-left: 1.25rem; }
	.markdown :global(ol) { list-style: decimal; padding-left: 1.25rem; }
	.markdown :global(li + li) { margin-top: 0.2rem; }
	.markdown :global(strong) { color: rgb(220 232 223); font-weight: 650; }
	.markdown :global(h1),
	.markdown :global(h2),
	.markdown :global(h3) {
		margin: 1rem 0 0.45rem;
		color: rgb(220 232 223);
		font-weight: 650;
		line-height: 1.3;
	}
	.markdown :global(h1) { font-size: 1.2rem; }
	.markdown :global(h2) { font-size: 1.08rem; }
	.markdown :global(h3) { font-size: 1rem; }
	.markdown :global(a) { color: rgb(74 222 128); text-decoration: underline; text-underline-offset: 2px; }
	.markdown :global(code) {
		border-radius: 0.25rem;
		background: rgb(8 16 11);
		padding: 0.1rem 0.3rem;
		font-size: 0.86em;
	}
	.markdown :global(pre) { overflow-x: auto; border-radius: 0.5rem; background: rgb(8 16 11); padding: 0.75rem; }
	.markdown :global(pre code) { background: transparent; padding: 0; }
	.markdown :global(blockquote) { border-left: 2px solid rgb(46 125 76); padding-left: 0.75rem; color: rgb(143 166 149); }
	.markdown :global(table) { width: 100%; border-collapse: collapse; font-size: 0.92em; }
	.markdown :global(th),
	.markdown :global(td) { border: 1px solid rgb(34 63 43); padding: 0.35rem 0.5rem; text-align: left; }
	.markdown :global(th) { background: rgb(18 35 23); color: rgb(220 232 223); }
</style>
