# fileViews
a simple hierarchical file and directory viewer

## Building

Requires Go 1.26+, a C compiler, and the OpenGL/X11 development headers used by [Fyne](https://docs.fyne.io/started/).

Debian / Ubuntu:

    sudo apt install gcc libgl1-mesa-dev xorg-dev

Fedora:

    sudo dnf install gcc libXcursor-devel libXrandr-devel mesa-libGL-devel libXi-devel libXinerama-devel libXxf86vm-devel

Then:

    make build   # produces ./fileviews
    make run
    make test
