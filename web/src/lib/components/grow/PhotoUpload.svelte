<script lang="ts">
	import { uploadGrowPhoto } from '$lib/api';
	import { errMsg } from '$lib/errors';
	import type { Snippet } from 'svelte';
	import Camera from '@lucide/svelte/icons/camera';

	interface Props {
		growId: string;
		plantUnitId?: string;
		onUploaded?: () => void;
		/** Custom trigger content; defaults to a labelled button. */
		trigger?: Snippet<[{ busy: boolean }]>;
		triggerClass?: string;
		label?: string;
	}
	let {
		growId,
		plantUnitId,
		onUploaded,
		trigger,
		triggerClass = 'inline-flex items-center gap-1.5 rounded-md border border-rig-700 px-3 py-1.5 text-sm text-rig-300 transition-colors hover:border-leaf hover:text-rig-100',
		label = 'Add photo'
	}: Props = $props();

	let input = $state<HTMLInputElement>();
	let busy = $state(false);
	let error = $state('');

	function pick() {
		error = '';
		input?.click();
	}

	function readAsDataURL(file: File): Promise<string> {
		return new Promise((resolve, reject) => {
			const reader = new FileReader();
			reader.onload = () => resolve(reader.result as string);
			reader.onerror = () => reject(reader.error);
			reader.readAsDataURL(file);
		});
	}

	async function onChange(e: Event) {
		const file = (e.target as HTMLInputElement).files?.[0];
		if (!file) return;
		busy = true;
		error = '';
		try {
			const image = await readAsDataURL(file);
			await uploadGrowPhoto(growId, { image, plantUnitId });
			onUploaded?.();
		} catch (err) {
			error = errMsg(err, 'Upload failed');
		} finally {
			busy = false;
			if (input) input.value = '';
		}
	}
</script>

<input bind:this={input} type="file" accept="image/*" class="hidden" onchange={onChange} />
<button onclick={pick} disabled={busy} class={triggerClass}>
	{#if trigger}
		{@render trigger({ busy })}
	{:else}
		<Camera size={15} /> {busy ? 'Uploading…' : label}
	{/if}
</button>
{#if error}<span class="ml-2 text-xs text-danger">{error}</span>{/if}
