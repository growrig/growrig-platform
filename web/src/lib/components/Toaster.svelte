<script lang="ts">
	import { toast, type ToastKind } from '$lib/toast.svelte';
	import { fly } from 'svelte/transition';
	import CircleCheck from '@lucide/svelte/icons/circle-check';
	import Info from '@lucide/svelte/icons/info';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';
	import CircleX from '@lucide/svelte/icons/circle-x';
	import X from '@lucide/svelte/icons/x';

	const icons = {
		success: CircleCheck,
		info: Info,
		warning: TriangleAlert,
		error: CircleX
	};
	const accent: Record<ToastKind, string> = {
		success: 'text-leaf',
		info: 'text-sky-400',
		warning: 'text-warn',
		error: 'text-danger'
	};
</script>

<!-- Global toast stack. Mounted once in the root layout; driven by `toast`. -->
<div
	class="pointer-events-none fixed inset-x-0 bottom-0 z-[100] flex flex-col items-center gap-2 p-4 sm:items-start"
	role="region"
	aria-label="Notifications"
>
	{#each toast.items as t (t.id)}
		{@const Icon = icons[t.kind]}
		<div
			in:fly={{ y: 16, duration: 200 }}
			out:fly={{ y: 16, duration: 150 }}
			role={t.kind === 'error' ? 'alert' : 'status'}
			aria-live={t.kind === 'error' ? 'assertive' : 'polite'}
			class="pointer-events-auto flex w-full max-w-sm items-start gap-3 rounded-lg border border-rig-700 bg-rig-900 px-4 py-3 shadow-xl"
		>
			<Icon size={18} class="mt-0.5 shrink-0 {accent[t.kind]}" />
			<div class="min-w-0 flex-1">
				<p class="text-sm font-medium text-rig-100">{t.title}</p>
				{#if t.description}<p class="mt-0.5 break-words text-xs text-rig-400">{t.description}</p>{/if}
				{#if t.action}
					{#if t.action.href}
						<a
							href={t.action.href}
							onclick={() => toast.dismiss(t.id)}
							class="mt-1.5 inline-block text-xs font-medium text-leaf hover:underline"
						>
							{t.action.label}
						</a>
					{:else}
						<button
							type="button"
							onclick={() => {
								t.action?.onClick?.();
								toast.dismiss(t.id);
							}}
							class="mt-1.5 text-xs font-medium text-leaf hover:underline"
						>
							{t.action.label}
						</button>
					{/if}
				{/if}
			</div>
			<button
				type="button"
				onclick={() => toast.dismiss(t.id)}
				aria-label="Dismiss notification"
				class="shrink-0 rounded p-0.5 text-rig-500 transition-colors hover:text-rig-200"
			>
				<X size={15} />
			</button>
		</div>
	{/each}
</div>
