import type { Measurement } from './types';

export type Tone = 'good' | 'warn' | 'danger' | 'muted';

export const measurementUnit: Record<Measurement, string> = {
	temperature: '°C',
	humidity: '%',
	co2: 'ppm'
};

export const measurementLabel: Record<Measurement, string> = {
	temperature: 'Temperature',
	humidity: 'Humidity',
	co2: 'CO₂'
};

export function formatValue(measurement: Measurement, value: number): string {
	if (measurement === 'co2') return Math.round(value).toString();
	if (measurement === 'humidity') return value.toFixed(0);
	return value.toFixed(1);
}

// VPD growth zones (kPa), a widely used rule of thumb.
export function vpdZone(vpd: number): { label: string; tone: Tone } {
	if (vpd < 0.4) return { label: 'Too humid', tone: 'danger' };
	if (vpd < 0.8) return { label: 'Propagation', tone: 'warn' };
	if (vpd < 1.2) return { label: 'Vegetative', tone: 'good' };
	if (vpd < 1.6) return { label: 'Flowering', tone: 'good' };
	return { label: 'Too dry', tone: 'danger' };
}

export const toneClass: Record<Tone, string> = {
	good: 'text-leaf',
	warn: 'text-warn',
	danger: 'text-danger',
	muted: 'text-rig-400'
};

// Tent air volume in m³ from centimetre dimensions; 0 if any dimension is unset.
export function volumeM3(widthCm: number, depthCm: number, heightCm: number): number {
	if (widthCm <= 0 || depthCm <= 0 || heightCm <= 0) return 0;
	return (widthCm * depthCm * heightCm) / 1_000_000;
}

// "120 × 120 × 200 cm" from centimetre dimensions, or '' if any is unset.
export function formatDimensions(widthCm: number, depthCm: number, heightCm: number): string {
	if (widthCm <= 0 || depthCm <= 0 || heightCm <= 0) return '';
	return `${widthCm} × ${depthCm} × ${heightCm} cm`;
}

export function climateTone(tempC: number, target: number, emergency: number): Tone {
	if (emergency > 0 && tempC >= emergency) return 'danger';
	if (tempC - target >= 2) return 'warn';
	return 'good';
}
