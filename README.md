# fileViews
a simple hierarchical file and directory viewer

## Requirements

fileViews is a graphical Linux app built with [Fyne](https://docs.fyne.io/started/).
It works in both X11 and Wayland desktop sessions, so it needs the libraries for both.
Most desktop installs already have the runtime packages; they matter mainly on minimal installs.

The package names for Ubuntu were checked on Ubuntu 26.04. Debian uses the same names.
The Fedora names follow Fedora's naming, but were not tested on a Fedora system.

### To run

You need these packages to run a ready-built `fileviews` binary:

| Purpose | Ubuntu / Debian | Fedora |
|---------|-----------------|--------|
| OpenGL (graphics) | `libgl1 libglx-mesa0 libgl1-mesa-dri libegl1` | `mesa-libGL mesa-libEGL mesa-dri-drivers` |
| X11 sessions | `libx11-6 libxcursor1 libxrandr2 libxinerama1 libxi6 libxxf86vm1` | `libX11 libXcursor libXrandr libXinerama libXi libXxf86vm` |
| Wayland sessions | `libwayland-client0 libwayland-cursor0 libwayland-egl1 libxkbcommon0` | `libwayland-client libwayland-cursor libwayland-egl libxkbcommon` |
| Opening files with their default app | `xdg-utils` | `xdg-utils` |

Ubuntu / Debian:

    sudo apt install libgl1 libglx-mesa0 libgl1-mesa-dri libegl1 \
        libx11-6 libxcursor1 libxrandr2 libxinerama1 libxi6 libxxf86vm1 \
        libwayland-client0 libwayland-cursor0 libwayland-egl1 libxkbcommon0 \
        xdg-utils

Fedora:

    sudo dnf install mesa-libGL mesa-libEGL mesa-dri-drivers \
        libX11 libXcursor libXrandr libXinerama libXi libXxf86vm \
        libwayland-client libwayland-cursor libwayland-egl libxkbcommon \
        xdg-utils

Several of the X11 and Wayland libraries are loaded only when the app starts,
so `ldd fileviews` does not list them all. A missing one shows up as an error at startup.

### To build

Building needs everything under [To run](#to-run), plus:

- **Go 1.26 or newer.** Distribution packages are often older; if so, install Go from <https://go.dev/dl/>.
- **A C compiler and the development headers** for the libraries above:

Ubuntu / Debian:

    sudo apt install gcc libc6-dev libgl-dev \
        libx11-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev libxxf86vm-dev \
        libwayland-dev libxkbcommon-dev

Fedora:

    sudo dnf install gcc mesa-libGL-devel \
        libX11-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel libXxf86vm-devel \
        wayland-devel libxkbcommon-devel

If the build stops with `cannot find -lXxxx`, the development package for library `Xxxx` is missing.
For example, `-lXxf86vm` needs `libxxf86vm-dev` (Ubuntu/Debian) or `libXxf86vm-devel` (Fedora).

## Building

    make build   # produces ./fileviews
    make run
    make test
