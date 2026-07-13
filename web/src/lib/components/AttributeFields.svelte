<script module lang="ts">
	/** One dynamic field, as declared by a species (cultivar attributes) or an
	 *  inventory category (columns). Covers both schemas' field types. */
	export interface FieldDef {
		key: string;
		label: string;
		type: 'text' | 'number' | 'percent' | 'enum' | 'date';
		options?: string[];
		unit?: string;
	}
</script>

<script lang="ts">
	import { Select, DatePicker, fieldClass } from '$lib/components/ui';
	import { titleCase } from '$lib/format';

	interface Props {
		schema: FieldDef[];
		/** Bindable map of key -> string value. */
		values?: Record<string, string>;
		/** Extra classes for the grid wrapper. */
		class?: string;
	}

	let { schema, values = $bindable({}), class: className = 'grid gap-3 sm:grid-cols-2' }: Props =
		$props();
</script>

<div class={className}>
	{#each schema as field (field.key)}
		<label class="block">
			<span class="text-xs text-rig-400">
				{field.label}{#if field.unit}<span class="text-rig-600"> ({field.unit})</span>{/if}
			</span>
			{#if field.type === 'enum'}
				<Select
					class="mt-1"
					value={values[field.key] ?? ''}
					placeholder="—"
					items={(field.options ?? []).map((opt) => ({ value: opt, label: titleCase(opt) }))}
					onValueChange={(v) => (values[field.key] = v)}
				/>
			{:else if field.type === 'date'}
				<DatePicker class="mt-1" bind:value={values[field.key]} />
			{:else if field.type === 'number' || field.type === 'percent'}
				<input
					type="number"
					inputmode="decimal"
					step="any"
					min="0"
					bind:value={values[field.key]}
					placeholder={field.type === 'percent' ? '%' : ''}
					class="{fieldClass} mt-1"
				/>
			{:else}
				<input bind:value={values[field.key]} class="{fieldClass} mt-1" />
			{/if}
		</label>
	{/each}
</div>
