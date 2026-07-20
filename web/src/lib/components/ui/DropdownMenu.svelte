<script module lang="ts">
	import type { IconComponent } from '$lib/icons';

	export type DropdownItem = {
		label: string;
		href?: string;
		onSelect?: () => void;
		icon?: IconComponent;
		disabled?: boolean;
		/** Destructive action — red text. */
		danger?: boolean;
	};
</script>

<script lang="ts">
	import { DropdownMenu } from 'bits-ui';
	import type { Snippet } from 'svelte';

	interface Props {
		items: DropdownItem[];
		/** Trigger content. Receives no args. */
		trigger: Snippet;
		align?: 'start' | 'center' | 'end';
		triggerClass?: string;
		/** Open on hover (as well as click). Useful for header menus. */
		hover?: boolean;
	}

	let { items, trigger, align = 'end', triggerClass, hover = false }: Props = $props();

	const itemClass =
		'flex cursor-pointer select-none items-center gap-2 rounded-md px-3 py-2 text-sm outline-none data-[disabled]:pointer-events-none data-[disabled]:opacity-40';

	// Hover control: open immediately on enter, close after a short grace period
	// so moving the pointer across the gap onto the menu doesn't dismiss it.
	let open = $state(false);
	let closeTimer: ReturnType<typeof setTimeout> | undefined;
	function openNow() {
		clearTimeout(closeTimer);
		open = true;
	}
	function scheduleClose() {
		clearTimeout(closeTimer);
		closeTimer = setTimeout(() => (open = false), 150);
	}
</script>

<DropdownMenu.Root bind:open>
	<DropdownMenu.Trigger
		class={triggerClass}
		onpointerenter={hover ? openNow : undefined}
		onpointerleave={hover ? scheduleClose : undefined}
	>
		{@render trigger()}
	</DropdownMenu.Trigger>

	<DropdownMenu.Portal>
		<DropdownMenu.Content
			{align}
			sideOffset={6}
			onpointerenter={hover ? openNow : undefined}
			onpointerleave={hover ? scheduleClose : undefined}
			class="z-50 min-w-44 overflow-hidden rounded-lg border border-rig-700 bg-rig-900 p-1 shadow-xl outline-none"
		>
			{#each items as item (item.label)}
				<DropdownMenu.Item
					disabled={item.disabled}
					onSelect={item.onSelect}
					class="{itemClass} {item.danger
						? 'text-danger data-[highlighted]:bg-danger/15 data-[highlighted]:text-danger'
						: 'text-rig-200 data-[highlighted]:bg-rig-800 data-[highlighted]:text-rig-50'}"
				>
					{#snippet child({ props })}
						{#if item.href}
							<a {...props} href={item.href}>
								{#if item.icon}
									<item.icon size={16} class={item.danger ? 'text-danger' : 'text-rig-400'} />
								{/if}
								<span>{item.label}</span>
							</a>
						{:else}
							<div {...props}>
								{#if item.icon}
									<item.icon size={16} class={item.danger ? 'text-danger' : 'text-rig-400'} />
								{/if}
								<span>{item.label}</span>
							</div>
						{/if}
					{/snippet}
				</DropdownMenu.Item>
			{/each}
		</DropdownMenu.Content>
	</DropdownMenu.Portal>
</DropdownMenu.Root>
