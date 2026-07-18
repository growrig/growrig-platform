<script lang="ts">
	import { Switch } from 'bits-ui';
	import { cn } from './utils';

	type Props = Omit<Switch.RootProps, 'child' | 'children'> & { class?: string };

	let { checked = $bindable(false), class: className, ...rest }: Props = $props();
</script>

<Switch.Root
	bind:checked
	class={cn(
		'peer inline-flex h-6 w-11 shrink-0 cursor-pointer items-center rounded-full border border-transparent transition-colors',
		'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-leaf focus-visible:ring-offset-2 focus-visible:ring-offset-rig-950',
		'disabled:cursor-not-allowed disabled:opacity-50',
		'data-[state=checked]:bg-leaf data-[state=unchecked]:bg-rig-700',
		className
	)}
	{...rest}
>
	<Switch.Thumb
		class="pointer-events-none block h-5 w-5 rounded-full bg-rig-50 shadow-sm transition-transform data-[state=checked]:translate-x-5 data-[state=unchecked]:translate-x-0.5"
	/>
</Switch.Root>
