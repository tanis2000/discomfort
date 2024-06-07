package service

import (
    "github.com/bwmarrin/discordgo"
)

type Handler interface {
    GetName() string
    GetApplicationCommand() *discordgo.ApplicationCommand
    HandleCommand(ctx Context, s *discordgo.Session, i *discordgo.InteractionCreate, opts map[string]*discordgo.ApplicationCommandInteractionDataOption)
}

func GetDiscordUserId(i *discordgo.InteractionCreate) string {
    if i.GuildID == "" {
        return i.Interaction.User.ID
    } else {
        return i.Interaction.Member.User.ID
    }
}
