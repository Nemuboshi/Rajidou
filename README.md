# Rajidou CLI

Rajidou is a CLI for downloading Radiko timeshift audio.

This project reimplements core ideas and workflows from the original Rajiko project:
- Rajiko: https://github.com/jackyzy823/rajiko

Core logic in this repository (auth flow, playlist/chunklist handling, and AAC segment processing) is derived from Rajiko behavior and adapted for a standalone CLI.

## Run

If `--config` is omitted, `config.yaml` is used.

See `config.example.yaml` for config format.
