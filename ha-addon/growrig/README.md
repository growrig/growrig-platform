# GrowRig add-on — Grow Core for Home Assistant OS

Runs [Grow Core](../../growcore/) (control engine + embedded Grow App Web
dashboard) as a **local** Home Assistant OS add-on. Grow Core reaches Home
Assistant through the Supervisor proxy, so it needs no long-lived token.

This is the **manual install** path: you copy this folder onto your HAOS host
and install it from *Local add-ons*. For a full disk image with GrowRig
preinstalled, see the HAOS image build.

## Build the binaries

The add-on ships prebuilt, arch-matched binaries under `bin/`. Produce them
from the repo root:

```bash
make addon        # cross-compiles bin/growcore.{aarch64,amd64,armv7}
```

Each binary has the web UI and device catalogue embedded (`go:embed`), so the
on-device install is just a thin copy onto the Home Assistant base image.

## Install

1. Copy this `growrig/` folder (with `bin/` populated) to `/addons/growrig/` on
   your Home Assistant OS host.
2. **Settings → Add-ons → Add-on Store → ⋮ → Check for updates**, then open
   **GrowRig — Grow Core** under *Local add-ons* and click **Install → Start**.
3. Click **OPEN WEB UI** (default host port `8099`).

See [DOCS.md](DOCS.md) for options and details.

## Layout

```
ha-addon/growrig/
├── config.yaml     # add-on manifest (arch, ports, options, Supervisor API)
├── build.yaml      # HA base images per architecture
├── Dockerfile      # thin image: copies the arch-matched binary in
├── rootfs/
│   └── run.sh       # entrypoint: writes Supervisor-mode config, execs binary
├── bin/            # prebuilt binaries (git-ignored; `make addon`)
├── README.md
└── DOCS.md         # shown on the add-on's Documentation tab
```
