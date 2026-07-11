<script lang="ts">
	import { Dialog } from 'bits-ui';
	import X from '@lucide/svelte/icons/x';
	import type { Snippet } from 'svelte';

	interface Props {
		/** Bindable open state. */
		open?: boolean;
		title: string;
		description?: string;
		/** Trigger content. Omit to control `open` externally. */
		trigger?: Snippet;
		triggerClass?: string;
		/** Max width of the dialog. Defaults to `lg`. */
		size?: 'lg' | 'xl' | '2xl' | '3xl';
		children: Snippet;
	}

	let {
		open = $bindable(false),
		title,
		description,
		trigger,
		triggerClass,
		size = 'lg',
		children
	}: Props = $props();

	const maxW = $derived(
		{
			lg: 'max-w-lg',
			xl: 'max-w-xl',
			'2xl': 'max-w-2xl',
			'3xl': 'max-w-3xl'
		}[size]
	);
</script>

<Dialog.Root bind:open>
	{#if trigger}
		<Dialog.Trigger class={triggerClass}>
			{@render trigger()}
		</Dialog.Trigger>
	{/if}

	<Dialog.Portal>
		<Dialog.Overlay
			class="fixed inset-0 z-50 bg-black/60 backdrop-blur-sm data-[state=open]:animate-in data-[state=closed]:animate-out"
		/>
		<Dialog.Content
			class="fixed left-1/2 top-1/2 z-50 flex max-h-[85vh] w-[92vw] {maxW} -translate-x-1/2 -translate-y-1/2 flex-col overflow-hidden rounded-xl border border-rig-700 bg-rig-900 shadow-2xl outline-none"
		>
			<div class="flex items-start justify-between gap-4 border-b border-rig-800 px-5 py-4">
				<div>
					<Dialog.Title class="text-lg font-semibold text-rig-50">{title}</Dialog.Title>
					{#if description}
						<Dialog.Description class="mt-0.5 text-sm text-rig-400">
							{description}
						</Dialog.Description>
					{/if}
				</div>
				<Dialog.Close
					class="rounded-md p-1 text-rig-400 transition-colors hover:bg-rig-800 hover:text-rig-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-rig-600"
					aria-label="Close"
				>
					<X size={18} />
				</Dialog.Close>
			</div>
			<div class="overflow-y-auto px-5 py-4">
				{@render children()}
			</div>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>
