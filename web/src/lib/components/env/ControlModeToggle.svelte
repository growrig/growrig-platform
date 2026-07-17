<script lang="ts">
	import type { ControlMode } from '$lib/types';
	import Zap from '@lucide/svelte/icons/zap';
	import Hand from '@lucide/svelte/icons/hand';

	interface Props {
		value: ControlMode;
		onChange: (mode: ControlMode) => void;
		disabled?: boolean;
	}
	let { value, onChange, disabled = false }: Props = $props();

	const opts: { mode: ControlMode; label: string; icon: typeof Zap }[] = [
		{ mode: 'auto', label: 'Automatic', icon: Zap },
		{ mode: 'manual', label: 'Manual', icon: Hand }
	];
</script>

<div class="inline-flex rounded-lg border border-rig-800 bg-rig-950/60 p-0.5" role="group">
	{#each opts as o (o.mode)}
		{@const active = value === o.mode}
		<button
			type="button"
			{disabled}
			onclick={() => !active && onChange(o.mode)}
			aria-pressed={active}
			class="inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-colors disabled:opacity-40 {active
				? o.mode === 'auto'
					? 'bg-leaf/15 text-leaf'
					: 'bg-rig-700 text-rig-50'
				: 'text-rig-400 hover:text-rig-100'}"
		>
			<o.icon size={13} />
			{o.label}
		</button>
	{/each}
</div>
