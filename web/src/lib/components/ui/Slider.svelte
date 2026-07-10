<script lang="ts">
	import { Slider } from 'bits-ui';
	import { cn } from './utils';

	interface Props {
		value?: number;
		min?: number;
		max?: number;
		step?: number;
		disabled?: boolean;
		/** Accent for the range fill; use 'warn' for emergency-style controls. */
		tone?: 'rig' | 'warn';
		class?: string;
		onValueChange?: (value: number) => void;
	}

	let {
		value = $bindable(0),
		min = 0,
		max = 100,
		step = 1,
		disabled = false,
		tone = 'rig',
		class: className,
		onValueChange
	}: Props = $props();

	const rangeColor = $derived(tone === 'warn' ? 'bg-warn' : 'bg-rig-500');
	const thumbColor = $derived(
		tone === 'warn' ? 'border-warn focus-visible:ring-warn' : 'border-rig-500 focus-visible:ring-rig-400'
	);
</script>

<Slider.Root
	type="single"
	bind:value
	{min}
	{max}
	{step}
	{disabled}
	{onValueChange}
	class={cn('relative flex h-5 w-full touch-none select-none items-center', className)}
>
	{#snippet children({ thumbItems })}
		<span class="relative h-1.5 w-full grow overflow-hidden rounded-full bg-rig-800">
			<Slider.Range class={cn('absolute h-full', rangeColor)} />
		</span>
		{#each thumbItems as thumb (thumb.index)}
			<Slider.Thumb
				index={thumb.index}
				class={cn(
					'block h-4 w-4 rounded-full border-2 bg-rig-50 shadow transition-colors',
					'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-offset-rig-950',
					'disabled:pointer-events-none disabled:opacity-50',
					thumbColor
				)}
			/>
		{/each}
	{/snippet}
</Slider.Root>
