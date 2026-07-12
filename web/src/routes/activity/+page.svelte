<script lang="ts">
	import ActivityLog from '$lib/components/ActivityLog.svelte';
</script>

<div class="space-y-5">
	<div><h1 class="text-2xl font-semibold">Activity Log</h1><p class="mt-1 text-sm text-rig-400">Controls, warnings and system notices across all environments.</p></div>
	<ActivityLog limit={25} showEnvironment showFilters />
</div>
