<script lang="ts">
	import type { Measurement, SensorReading } from '$lib/types';
	import { measurementLabel, measurementUnit } from '$lib/format';
	import { measurementIcon } from '$lib/icons';
	import { Dialog } from '$lib/components/ui';
	import Thermometer from '@lucide/svelte/icons/thermometer';

	interface Props {
		sensors: SensorReading[];
		open?: boolean;
	}
	let { sensors, open = $bindable(false) }: Props = $props();

	const order: Measurement[] = ['temperature', 'humidity', 'co2'];

	// Group readings by measurement, preserving a stable measurement order.
	const groups = $derived(
		order
			.map((m) => ({ measurement: m, items: sensors.filter((s) => s.measurement === m) }))
			.filter((g) => g.items.length > 0)
	);

	function average(items: SensorReading[]): number | null {
		const ok = items.filter((s) => s.ok);
		if (ok.length === 0) return null;
		return ok.reduce((sum, s) => sum + s.value, 0) / ok.length;
	}
</script>

<Dialog
	bind:open
	title="Sensors"
	description="Every bound sensor and its latest reading. Values marked offline are excluded from the aggregate."
>
	{#snippet trigger()}
		<span class="inline-flex items-center gap-1.5 rounded-md border border-rig-700 px-3 py-1.5 text-sm text-rig-300 transition-colors hover:border-rig-500 hover:text-rig-100">
			<Thermometer size={15} /> Sensors <span class="text-rig-500">({sensors.length})</span>
		</span>
	{/snippet}

	{#if sensors.length === 0}
		<p class="text-sm text-rig-400">No sensors bound to this environment yet.</p>
	{:else}
		<div class="space-y-5">
			{#each groups as group (group.measurement)}
				{@const avg = average(group.items)}
				{@const GroupIcon = measurementIcon[group.measurement]}
				<div>
					<div class="mb-2 flex items-baseline justify-between">
						<h3 class="flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wide text-rig-400">
							<GroupIcon size={14} />
							{measurementLabel[group.measurement]}
						</h3>
						<span class="text-xs text-rig-500">
							{#if avg !== null}
								avg {avg.toFixed(group.measurement === 'temperature' ? 1 : 0)}{measurementUnit[group.measurement]}
								· {group.items.length} sensor{group.items.length > 1 ? 's' : ''}
							{:else}
								no live reading
							{/if}
						</span>
					</div>
					<ul class="space-y-1.5">
						{#each group.items as s (s.id)}
							<li class="flex items-center justify-between gap-3 rounded-lg border border-rig-800 bg-rig-950/40 px-3 py-2">
								<div class="min-w-0">
									<div class="truncate text-sm text-rig-100">{s.name}</div>
									<div class="truncate font-mono text-xs text-rig-500">{s.entity}</div>
								</div>
								<div class="flex items-center gap-2 whitespace-nowrap">
									<span class="text-sm font-semibold tabular-nums {s.ok ? 'text-rig-100' : 'text-rig-600'}">
										{s.ok ? `${s.value}${measurementUnit[s.measurement]}` : '—'}
									</span>
									<span
										class="rounded-full px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide {s.ok
											? 'bg-leaf/15 text-leaf'
											: 'bg-danger/15 text-danger'}"
									>
										{s.ok ? 'online' : 'offline'}
									</span>
								</div>
							</li>
						{/each}
					</ul>
				</div>
			{/each}
		</div>
	{/if}
</Dialog>
