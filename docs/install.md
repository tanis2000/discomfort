# OS

Discomfort is expected to work on all major versions of Linux, macOS and Windows.

# ComfyUI

Due to ComfyUI not following a strict release versioning, it is hard to provide a compatibility matrix. We try to make sure that Discomfort works with the latest version from master but we cannot guarantee 100% compatibility at all times.

# ComfyUI extensions

Discomfort assumes that some extensions have been installed. The list of extensions follows:

- [ComfyUI_InstantID](https://github.com/cubiq/ComfyUI_InstantID)

# Models

Discomfort uses a predefined set of checkpoints. This will change in the future as we are planning to let the you choose the checkpoints you like the most. For the time being, these checkpoints must be installed in ComfyUI.

The list of checkpoints follows:

- [dreamshaperXL_v21TurboDPMSDE](https://civitai.com/models/112902/dreamshaper-xl)
- [epicrealismXL_v7FinalDestination](https://civitai.com/models/277058/epicrealism-xl)
- [sd_xl_refiner_1.0_0.9vae](https://huggingface.co/stabilityai/stable-diffusion-xl-refiner-1.0/blob/main/sd_xl_refiner_1.0_0.9vae.safetensors)

# Discord application

Create a new Discord application for the bot as detailed in [Building your first Discord app](https://discord.com/developers/docs/getting-started)

# Running Discomfort

The easiest way to run Discomfort is to run it with Docker.

We provide a [docker-compose](../deploy/docker/docker-compose.yml) file to get you started.

Three environment variables must be set in the docker-compose file with the following data:

- TOKEN -> The Discord Application token that you got when you created the Discord application.
- COMFY_ADDRESS -> The IP address or the hostname of your ComfyUI server. If you are running it locally, you can use 127.0.0.1
- COMFY_PORT -> The port number the ComfyUI server is using. By default, it is 8188