# GrowRig — Grow Core

Grow Core is the control engine for [GrowRig](https://github.com/growrig/growrig-platform):
a Go service with a built-in SvelteKit dashboard (**Grow App Web**) that turns
climate targets into fan commands via Home Assistant. This add-on runs it
inside Home Assistant OS and talks to Home Assistant through the Supervisor
proxy — no long-lived token required.

## Installation

1. Copy the `growrig` folder into the `addons` share on your Home Assistant OS
   host (via Samba, the SSH/Terminal add-on, or the Studio Code Server add-on).
   The result should be `/addons/growrig/`.
2. In Home Assistant go to **Settings → Add-ons → Add-on Store**, open the
   **⋮** menu, and choose **Check for updates** (or reload the page). GrowRig
   appears under **Local add-ons**.
3. Open it and click **Install**. The Supervisor builds a small image around the
   prebuilt binary — this takes under a minute.
4. Click **Start**, then **OPEN WEB UI** to reach the dashboard.

> The `growrig` folder must contain arch-matched binaries under `bin/`. If you
> are building from source, run `make addon` at the repo root first (see the
> repository README), which cross-compiles them.

## Configuration

| Option | Default | Description |
| --- | --- | --- |
| `control_interval` | `2s` | How often Grow Core runs a reconciliation tick (Go duration, e.g. `1s`, `5s`). |

Everything else — environments, devices, channel roles, and which Home
Assistant entities each sensor/fan binds to — is owned by Grow Core and edited
in the web UI (Setup), then stored in the add-on's persistent `/data` volume.

## Network

The dashboard and API are served on container port **8080**, mapped to host
port **8099** by default. Change the host port under the add-on's **Network**
tab. Point ESPHome/Home Assistant fan and climate entities at the roles you
assign in Setup.

## Support

Issues and questions: https://github.com/growrig/growrig-platform/issues
