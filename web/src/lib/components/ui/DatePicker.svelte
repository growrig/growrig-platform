<script lang="ts">
	import { DatePicker } from 'bits-ui';
	import { CalendarDate, type DateValue } from '@internationalized/date';
	import CalendarIcon from '@lucide/svelte/icons/calendar';
	import ChevronLeft from '@lucide/svelte/icons/chevron-left';
	import ChevronRight from '@lucide/svelte/icons/chevron-right';
	import { cn } from './utils';

	interface Props {
		/** Bindable value as an ISO date string (`YYYY-MM-DD`), or '' when empty. */
		value?: string;
		disabled?: boolean;
		/** Extra classes for the input field. */
		class?: string;
		onValueChange?: (value: string) => void;
	}

	let {
		value = $bindable(''),
		disabled = false,
		class: className,
		onValueChange
	}: Props = $props();

	/** Parse `YYYY-MM-DD` into a CalendarDate; undefined when blank/invalid. */
	function toDateValue(iso: string): DateValue | undefined {
		const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(iso);
		if (!m) return undefined;
		return new CalendarDate(Number(m[1]), Number(m[2]), Number(m[3]));
	}

	// Bridge the ISO-string prop to bits-ui's DateValue model, keeping both in
	// sync without feedback loops.
	let dateValue = $state<DateValue | undefined>(toDateValue(value));
	$effect(() => {
		const next = toDateValue(value);
		if ((next?.toString() ?? '') !== (dateValue?.toString() ?? '')) dateValue = next;
	});

	const segClass =
		'rounded px-0.5 tabular-nums focus:bg-rig-700 focus:text-rig-50 focus:outline-none data-[placeholder]:text-rig-500';
</script>

<!-- One-way `value` (not `bind:`): an empty field is `undefined`, and Svelte
     forbids `bind:value={undefined}` on a prop that has a fallback. bits-ui stays
     in sync through `onValueChange` below, which drives our ISO-string prop. -->
<DatePicker.Root
	value={dateValue}
	weekdayFormat="short"
	fixedWeeks
	{disabled}
	onValueChange={(v) => {
		value = v ? v.toString() : '';
		onValueChange?.(value);
	}}
>
	<DatePicker.Input
		class={cn(
			'flex h-9 w-full items-center gap-0.5 rounded-md border border-rig-700 bg-rig-950 px-3 text-sm text-rig-100 focus-within:border-leaf',
			'data-[disabled]:cursor-not-allowed data-[disabled]:opacity-50',
			className
		)}
	>
		{#snippet children({ segments })}
			{#each segments as { part, value: segValue }, i (i)}
				<DatePicker.Segment {part} class={segClass}>{segValue}</DatePicker.Segment>
			{/each}
			<DatePicker.Trigger
				class="ml-auto inline-flex items-center rounded p-1 text-rig-400 hover:text-rig-100 focus:outline-none"
				aria-label="Open calendar"
			>
				<CalendarIcon size={16} />
			</DatePicker.Trigger>
		{/snippet}
	</DatePicker.Input>

	<DatePicker.Content sideOffset={6} class="z-50">
		<DatePicker.Calendar
			class="rounded-lg border border-rig-700 bg-rig-900 p-3 shadow-xl"
		>
			{#snippet children({ months, weekdays })}
				<DatePicker.Header class="mb-2 flex items-center justify-between">
					<DatePicker.PrevButton
						class="inline-flex h-7 w-7 items-center justify-center rounded-md text-rig-300 hover:bg-rig-800 hover:text-rig-50"
					>
						<ChevronLeft size={16} />
					</DatePicker.PrevButton>
					<DatePicker.Heading class="text-sm font-medium text-rig-100" />
					<DatePicker.NextButton
						class="inline-flex h-7 w-7 items-center justify-center rounded-md text-rig-300 hover:bg-rig-800 hover:text-rig-50"
					>
						<ChevronRight size={16} />
					</DatePicker.NextButton>
				</DatePicker.Header>

				{#each months as month (month.value)}
					<DatePicker.Grid class="w-full border-collapse select-none">
						<DatePicker.GridHead>
							<DatePicker.GridRow class="flex">
								{#each weekdays as day (day)}
									<DatePicker.HeadCell
										class="w-8 text-center text-[11px] font-normal text-rig-500"
									>
										{day.slice(0, 2)}
									</DatePicker.HeadCell>
								{/each}
							</DatePicker.GridRow>
						</DatePicker.GridHead>
						<DatePicker.GridBody>
							{#each month.weeks as weekDates (weekDates)}
								<DatePicker.GridRow class="flex w-full">
									{#each weekDates as date (date)}
										<DatePicker.Cell {date} month={month.value} class="p-0">
											<DatePicker.Day
												class="inline-flex h-8 w-8 items-center justify-center rounded-md text-sm text-rig-200 hover:bg-rig-800 data-[disabled]:text-rig-700 data-[outside-month]:text-rig-700 data-[selected]:bg-leaf data-[selected]:font-medium data-[selected]:text-rig-950 data-[unavailable]:text-rig-700 data-[unavailable]:line-through"
											>
												{date.day}
											</DatePicker.Day>
										</DatePicker.Cell>
									{/each}
								</DatePicker.GridRow>
							{/each}
						</DatePicker.GridBody>
					</DatePicker.Grid>
				{/each}
			{/snippet}
		</DatePicker.Calendar>
	</DatePicker.Content>
</DatePicker.Root>
