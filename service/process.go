package service

import (
    "github.com/bwmarrin/discordgo"
    "github.com/tanis2000/comfy-client/client"
)

type Process struct {
    PromptID          string
    InteractionCreate *discordgo.InteractionCreate
    Session           *discordgo.Session
    ComfyClient       *client.Client
    PositivePrompt    string
    NegativePrompt    string
    Seed              uint64
}
