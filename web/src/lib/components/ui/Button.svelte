<script lang="ts">
	import { Button, type ButtonRootProps } from 'bits-ui';
	import { buttonBase, buttonSizes, buttonVariants, cn, type ButtonSize, type ButtonVariant } from './utils';

	type Props = ButtonRootProps & {
		variant?: ButtonVariant;
		size?: ButtonSize;
	};

	let {
		variant = 'primary',
		size = 'md',
		class: className,
		children,
		...rest
	}: Props = $props();
</script>

<Button.Root
	class={cn(buttonBase, buttonVariants[variant], buttonSizes[size], className as string | undefined)}
	{...rest}
>
	{@render children?.()}
</Button.Root>
