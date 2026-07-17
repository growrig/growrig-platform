// Central lucide icon mappings so binding kinds and measurements render one
// consistent icon set across the app. Import the named lucide components
// directly elsewhere for one-off icons.
import Thermometer from '@lucide/svelte/icons/thermometer';
import Fan from '@lucide/svelte/icons/fan';
import Lightbulb from '@lucide/svelte/icons/lightbulb';
import Camera from '@lucide/svelte/icons/camera';
import Plug from '@lucide/svelte/icons/plug';
import Droplets from '@lucide/svelte/icons/droplets';
import Wind from '@lucide/svelte/icons/wind';
import Zap from '@lucide/svelte/icons/zap';
import Cpu from '@lucide/svelte/icons/cpu';
import type { BindingKind, Measurement } from './types';

// All lucide icons share one component shape; key every map off that type.
export type IconComponent = typeof Thermometer;

export const bindingKindIcon: Record<BindingKind, IconComponent> = {
	sensor: Thermometer,
	fan: Fan,
	controller: Cpu,
	light: Lightbulb,
	power: Zap,
	camera: Camera,
	irrigation: Droplets
};

/** Fallback for kinds without a dedicated icon (e.g. generic devices/plugs). */
export const fallbackIcon: IconComponent = Plug;

export const measurementIcon: Record<Measurement, IconComponent> = {
	temperature: Thermometer,
	humidity: Droplets,
	co2: Wind,
	power: Zap
};
