# Discomfort, a Discord bot for ComfyUI

<div align="center"><img src="docs/logo/logo-64.png"  alt="Discomfort logo"/></div>

[![Windows](https://github.com/tanis2000/discomfort/actions/workflows/windows.yml/badge.svg)](https://github.com/tanis2000/discomfort/actions/workflows/windows.yml)
[![macOS](https://github.com/tanis2000/discomfort/actions/workflows/macos.yml/badge.svg)](https://github.com/tanis2000/discomfort/actions/workflows/macos.yml)
[![Linux](https://github.com/tanis2000/discomfort/actions/workflows/linux.yml/badge.svg)](https://github.com/tanis2000/discomfort/actions/workflows/linux.yml)

Discomfort is in an early development stage. Please feel free to play around with it and to contribute back.

The main aim of Discomfort is to provide an easy-to-use interface for ComfyUI over Discord. 

# Supported commands

* **/txt2img** to create images based on text

* **/faceswap** to apply a face to a generated image

* **/uploadimage** to upload an image that can be used in /faceswap

* **/uploadlist** to obtain the list of all the uploaded images

# Docker images

Docker images for x64 and arm64 Linux are available on [Docker Hub](https://hub.docker.com/r/tanis2000/discomfort)

# How to run locally

Download the latest binary for your OS from the [Releases](https://github.com/tanis2000/discomfort/releases) page.

Unzip and run the bot with the following command:

```shell
discomfort --token <your discord bot token goes here> --address 127.0.0.1 --port 8818
```

The supported parameters are the following:

| Parameter | Description                                                                                                                 |
|-----------|-----------------------------------------------------------------------------------------------------------------------------|
| --token   | The Discord bot token that can be obtained from the [Discord Developer Portal](https://discord.com/developers/applications) |
| --address | The IP address or hostname of the ComfyUI server                                                                            | 
| --port    | The port number of the ComfyUI server                                                                                       |

# Documentation

Further documentation is available in the [docs folder](docs/), starting from the [main page](docs/index.md)