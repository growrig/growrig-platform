<script module lang="ts">
	export type SelectItem = { value: string; label: string; disabled?: boolean };
</script>

<script lang="ts">
	import { Select } from 'bits-ui';
	import ChevronDown from '@lucide/svelte/icons/chevron-down';
	import { cn } from './utils';

	interface Props {
		items: SelectItem[];
		/** Bindable selected value. */
		value?: string;
		placeholder?: string;
		disabled?: boolean;
		name?: string;
		/** Extra classes for the trigger. */
		class?: string;
		onValueChange?: (value: string) => void;
	}

	let {
		items,
		value = $bindable(''),
		placeholder = 'Select…',
		disabled = false,
		name,
		class: className,
		onValueChange
	}: Props = $props();

	const selectedLabel = $derived(items.find((i) => i.value === value)?.label);
</script>

<Select.Root type="single" bind:value {name} {disabled} {items} {onValueChange}>
	<Select.Trigger
		class={cn(
			'inline-flex h-9 w-full items-center justify-between gap-2 rounded-md border border-rig-700 bg-rig-950 px-3 text-sm text-rig-100 transition-colors',
			'hover:border-rig-600 focus:border-rig-500 focus:outline-none data-[placeholder]:text-rig-500',
			'disabled:cursor-not-allowed disabled:opacity-50',
			className
		)}
	>
		<span class="truncate">{selectedLabel ?? placeholder}</span>
		<ChevronDown size={16} class="shrink-0 text-rig-400" />
	</Select.Trigger>

	<Select.Portal>
		<Select.Content
			sideOffset={6}
			class="z-50 max-h-72 w-[var(--bits-select-anchor-width)] min-w-[8rem] overflow-hidden rounded-lg border border-rig-700 bg-rig-900 shadow-xl outline-none"
		>
			<Select.Viewport class="p-1">
				{#each items as item (item.value)}
					<Select.Item
						value={item.value}
						label={item.label}
						disabled={item.disabled}
						class="flex cursor-pointer select-none items-center justify-between rounded-md px-3 py-1.5 text-sm text-rig-200 outline-none data-[disabled]:pointer-events-none data-[disabled]:opacity-40 data-[highlighted]:bg-rig-800 data-[highlighted]:text-rig-50"
					>
						{#snippet children({ selected })}
							<span class="truncate">{item.label}</span>
							{#if selected}<span class="text-rig-400">✓</span>{/if}
						{/snippet}
					</Select.Item>
				{/each}
			</Select.Viewport>
		</Select.Content>
	</Select.Portal>
</Select.Root>
