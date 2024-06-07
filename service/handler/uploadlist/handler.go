package uploadlist

import (
    "discomfort/service"
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
    builder := new(strings.Builder)
    builder.WriteString("Available images: \n")
    for _, v := range ctx.Bot.GetImages() {
        builder.WriteString(v + "\n")
    }

    err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseChannelMessageWithSource,
        Data: &discordgo.InteractionResponseData{
            Content: builder.String(),
        },
    })

    if err != nil {
        log.Panicf("could not respond to interaction: %s", err)
    }
}
