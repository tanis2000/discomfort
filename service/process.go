package service

import "github.com/bwmarrin/discordgo"

type Process struct {
    PromptID    string
    Interaction *discordgo.Interaction
    Session     *discordgo.Session
}
