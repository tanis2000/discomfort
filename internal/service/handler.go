package service

import (
    "discomfort/internal/build"
    "discomfort/internal/database"
    "fmt"
    "github.com/bwmarrin/discordgo"
    "log"
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

func EmbedTemplate() *discordgo.MessageEmbed {
    embed := &discordgo.MessageEmbed{
        Author: &discordgo.MessageEmbedAuthor{
            Name:    "discomfort",
            IconURL: "https://github.com/tanis2000/discomfort/blob/master/docs/logo/logo-64.png?raw=true",
        },
        Footer: &discordgo.MessageEmbedFooter{
            Text:    fmt.Sprintf("Powered by discomfort (%s)", build.Version),
            IconURL: "https://github.com/tanis2000/discomfort/blob/master/docs/logo/logo-64.png?raw=true",
        },
    }
    return embed
}

func UpdateDiscordUser(db *database.Database, userID string) error {
    log.Printf("Updating Discord User id:%s", userID)
    return nil
}
