<script module lang="ts">
	import type { IconComponent } from '$lib/icons';

	export type DropdownItem = {
		label: string;
		href?: string;
		onSelect?: () => void;
		icon?: IconComponent;
		disabled?: boolean;
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
	}

	let { items, trigger, align = 'end', triggerClass }: Props = $props();

	const itemClass =
		'flex cursor-pointer select-none items-center gap-2 rounded-md px-3 py-2 text-sm text-rig-200 outline-none data-[disabled]:pointer-events-none data-[disabled]:opacity-40 data-[highlighted]:bg-rig-800 data-[highlighted]:text-rig-50';
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger class={triggerClass}>
		{@render trigger()}
	</DropdownMenu.Trigger>

	<DropdownMenu.Portal>
		<DropdownMenu.Content
			{align}
			sideOffset={6}
			class="z-50 min-w-44 overflow-hidden rounded-lg border border-rig-700 bg-rig-900 p-1 shadow-xl outline-none"
		>
			{#each items as item (item.label)}
				<DropdownMenu.Item disabled={item.disabled} onSelect={item.onSelect} class={itemClass}>
					{#snippet child({ props })}
						{#if item.href}
							<a {...props} href={item.href}>
								{#if item.icon}<item.icon size={16} class="text-rig-400" />{/if}
								<span>{item.label}</span>
							</a>
						{:else}
							<div {...props}>
								{#if item.icon}<item.icon size={16} class="text-rig-400" />{/if}
								<span>{item.label}</span>
							</div>
						{/if}
					{/snippet}
				</DropdownMenu.Item>
			{/each}
		</DropdownMenu.Content>
	</DropdownMenu.Portal>
</DropdownMenu.Root>
