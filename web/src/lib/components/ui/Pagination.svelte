<script lang="ts">
	import { Pagination } from 'bits-ui';
	import ChevronLeft from '@lucide/svelte/icons/chevron-left';
	import ChevronRight from '@lucide/svelte/icons/chevron-right';
	import { cn } from './utils';

	interface Props {
		/** Total number of items across all pages. */
		count: number;
		/** Items per page. */
		perPage?: number;
		/** Bindable current page (1-based). */
		page?: number;
		onPageChange?: (page: number) => void;
		class?: string;
	}
	let { count, perPage = 20, page = $bindable(1), onPageChange, class: className }: Props = $props();

	const cell =
		'inline-flex h-8 min-w-8 items-center justify-center rounded-md border border-rig-800 px-2 text-sm text-rig-300 transition-colors hover:border-rig-600 disabled:cursor-not-allowed disabled:opacity-40';
</script>

<Pagination.Root {count} {perPage} bind:page {onPageChange} class={cn('flex items-center gap-1.5', className)}>
	{#snippet children({ pages, currentPage })}
		<Pagination.PrevButton class={cell} aria-label="Previous page">
			<ChevronLeft size={16} />
		</Pagination.PrevButton>
		{#each pages as p (p.key)}
			{#if p.type === 'ellipsis'}
				<span class="px-1 text-sm text-rig-600">…</span>
			{:else}
				<Pagination.Page
					page={p}
					class={cn(cell, currentPage === p.value && 'border-rig-500 bg-rig-800 text-rig-100')}
				>
					{p.value}
				</Pagination.Page>
			{/if}
		{/each}
		<Pagination.NextButton class={cell} aria-label="Next page">
			<ChevronRight size={16} />
		</Pagination.NextButton>
	{/snippet}
</Pagination.Root>
