<script module lang="ts">
	// Universal breadcrumb: a "/"-separated trail of links. The current page's
	// own title is rendered as the page <h1>, not repeated here — so every crumb
	// is a link to an ancestor. Used by the environment and grow profile headers.
	export type Crumb = { label: string; href: string };
</script>

<script lang="ts">
	interface Props {
		items: Crumb[];
	}
	let { items }: Props = $props();
</script>

<nav class="flex items-center gap-1 text-xs text-rig-500">
	{#each items as crumb, i (crumb.href + crumb.label)}
		{#if i > 0}<span class="text-rig-700">/</span>{/if}
		<a href={crumb.href} class="hover:text-rig-300">{crumb.label}</a>
	{/each}
</nav>
