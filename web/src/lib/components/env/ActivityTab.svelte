<script lang="ts">
	import ActivityLog from '$lib/components/ActivityLog.svelte';

	interface Props {
		environmentId: string;
	}
	let { environmentId }: Props = $props();
</script>

<ActivityLog {environmentId} limit={50} title="Activity" />
