<script lang="ts">
	// Settings moved into the environment page as its last tab.
	// Redirect any old /env/[id]/settings deep links to ?tab=settings.
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';

	onMount(() => {
		goto(`/env/${page.params.id}?tab=settings`, { replaceState: true });
	});
</script>

<p class="text-rig-400">Redirecting…</p>
