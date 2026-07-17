<script lang="ts">
	import type { Alert } from '$lib/types';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';
	import Info from '@lucide/svelte/icons/info';

	interface Props {
		/** Open alerts already filtered to this environment. */
		alerts: Alert[];
	}
	let { alerts }: Props = $props();

	const tone: Record<Alert['severity'], string> = {
		critical: 'border-danger/40 bg-danger/5 text-danger',
		warning: 'border-warn/40 bg-warn/5 text-warn',
		info: 'border-rig-700 bg-rig-900/40 text-rig-300'
	};
</script>

{#if alerts.length}
	<section class="space-y-2">
		{#each alerts as a (a.id)}
			<div class="flex items-center gap-3 rounded-lg border px-3 py-2.5 {tone[a.severity]}">
				<span class="shrink-0">
					{#if a.severity === 'info'}<Info size={16} />{:else}<TriangleAlert size={16} />{/if}
				</span>
				<div class="min-w-0 flex-1">
					<p class="text-sm font-medium">{a.title}</p>
					{#if a.message && a.message !== a.title}
						<p class="truncate text-xs opacity-80">{a.message}</p>
					{/if}
				</div>
				<a href="?tab=activity" class="shrink-0 text-xs opacity-80 transition-opacity hover:opacity-100">
					View
				</a>
			</div>
		{/each}
	</section>
{/if}
