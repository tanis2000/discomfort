package uploadlist

import (
    "discomfort/internal/service"
    "github.com/bwmarrin/discordgo"
    "log"
    "strings"
)

type Handler struct {
}

func (h Handler) GetName() string {
    return "uploadlist"
}

func (h Handler) GetApplicationCommand() *discordgo.ApplicationCommand {
    return &discordgo.ApplicationCommand{
        Name:        "uploadlist",
        Description: "List the images available",
    }
}
func (h Handler) HandleCommand(ctx service.Context, s *discordgo.Session, i *discordgo.InteractionCreate, opts map[string]*discordgo.ApplicationCommandInteractionDataOption) {
    userId := service.GetDiscordUserId(i)
    builder := new(strings.Builder)
    images, err := ctx.Bot.GetDatabase().FindImagesByUserId(userId, true)
    if err != nil {
        log.Printf("cannot load images for user %s: %s", userId, err.Error())
        builder.WriteString("Cannot retrieve the list of images")
        err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
            Type: discordgo.InteractionResponseChannelMessageWithSource,
            Data: &discordgo.InteractionResponseData{
                Content: builder.String(),
            },
        })
        if err != nil {
            log.Printf("could not respond to interaction: %s", err)
        }
        return
    }
    builder.WriteString("Available images: \n")
    for _, v := range images {
        builder.WriteString(v.Filename)
        if v.FriendlyName != "" {
            builder.WriteString(" (" + v.FriendlyName + ")")
        }
        builder.WriteString("\n")
    }

    err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseChannelMessageWithSource,
        Data: &discordgo.InteractionResponseData{
            Content: builder.String(),
        },
    })

    if err != nil {
        log.Printf("could not respond to interaction: %s", err)
    }
}
