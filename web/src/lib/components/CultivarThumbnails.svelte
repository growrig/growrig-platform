<script lang="ts">
	import type { Cultivar, GrowCultivarRef } from '$lib/types';
	import { cultivarImageURL } from '$lib/api';
	import Sprout from '@lucide/svelte/icons/sprout';

	interface Props {
		/** Per-cultivar plant counts to render as thumbnails. */
		refs: GrowCultivarRef[];
		/** Cultivar library, used to resolve each ref's image by name. */
		cultivars?: Cultivar[];
		/** Thumbnail diameter in pixels. */
		size?: number;
	}
	let { refs, cultivars = [], size = 36 }: Props = $props();

	const byName = $derived(new Map(cultivars.map((c) => [c.name, c])));
</script>

{#if refs.length}
	<div class="flex flex-wrap items-center gap-2">
		{#each refs as pc (pc.cultivar)}
			{@const cv = byName.get(pc.cultivar)}
			<div class="relative" title={pc.cultivar || 'No cultivar'}>
				<div
					class="overflow-hidden rounded-full border border-rig-700 bg-rig-950"
					style="width:{size}px;height:{size}px"
				>
					{#if cv?.imageType}
						<img src={cultivarImageURL(cv.id)} alt={pc.cultivar} class="h-full w-full object-cover" />
					{:else}
						<div class="flex h-full w-full items-center justify-center text-rig-600">
							<Sprout size={Math.round(size * 0.42)} />
						</div>
					{/if}
				</div>
				<span
					class="absolute -bottom-1 -right-1 min-w-[16px] rounded-full bg-leaf px-1 text-center text-[10px] font-semibold leading-4 text-rig-950"
				>
					{pc.count}
				</span>
			</div>
		{/each}
	</div>
{/if}
