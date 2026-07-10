<script lang="ts">
	import type { BindingKind } from '$lib/types';
	import { bindingKindIcon, fallbackIcon } from '$lib/icons';

	interface Props {
		kind: BindingKind;
		size?: number;
		class?: string;
	}
	let { kind, size = 18, class: className = 'text-rig-400' }: Props = $props();

	const Icon = $derived(bindingKindIcon[kind] ?? fallbackIcon);
</script>

<Icon {size} class={className} />
