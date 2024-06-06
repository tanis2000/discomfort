package service

import (
    "github.com/bwmarrin/discordgo"
    "github.com/tanis2000/comfy-client/client"
)

type Process struct {
    PromptID    string
    Interaction *discordgo.Interaction
    Session     *discordgo.Session
    ComfyClient *client.Client
}
