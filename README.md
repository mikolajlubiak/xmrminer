# xmrminer

A lightweight Monero (XMR) miner for Windows, built with Go.

## Build

Tested on Linux (cross-compile for Windows).

- Install necessary packages (different commands based on your distribution)
  - Fedora:
    - `sudo dnf install golang garble mingw64-gcc`
  - Arch:
    - `sudo pacman -S --needed go garble mingw-w64-gcc`
  - Ubuntu:
    - `sudo apt install golang garble gcc-mingw-w64`
- `git clone https://github.com/mikolajlubiak/xmrminer`
- `cd xmrminer`
- `./compilewin.sh`

## Usage

- Transfer the compiled `cortana.exe` binary to a Windows machine and run it
- The miner will start automatically in the background and add itself to autostart
- Logs are written to `cortanalog.txt` in the same directory
