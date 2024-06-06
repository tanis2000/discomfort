package service

import (
    "github.com/bwmarrin/discordgo"
    "github.com/tanis2000/comfy-client/client"
    "log"
    "strings"
)

type Bot struct {
    session      *discordgo.Session
    token        string
    comfyAddress string
    comfyPort    int
    imageDB      *ImageDB
    processes    map[string]Process
}

type optionMap = map[string]*discordgo.ApplicationCommandInteractionDataOption

func parseOptions(options []*discordgo.ApplicationCommandInteractionDataOption) (om optionMap) {
    om = make(optionMap)
    for _, opt := range options {
        om[opt.Name] = opt
    }
    return
}

func NewBot(token string, comfyAddress string, comfyPort int) *Bot {
    res := &Bot{
        token:        token,
        comfyAddress: comfyAddress,
        comfyPort:    comfyPort,
        imageDB:      NewImageDB(),
        processes:    map[string]Process{},
    }
    res.imageDB.Load()
    return res
}

func interactionAuthor(i *discordgo.Interaction) *discordgo.User {
    if i.Member != nil {
        return i.Member.User
    }
    return i.User
}

func (b *Bot) handleUploadList(s *discordgo.Session, i *discordgo.InteractionCreate, opts optionMap) {
    builder := new(strings.Builder)
    builder.WriteString("Available images: \n")
    for _, v := range b.imageDB.Images {
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

func consumeMessages(resp *client.QueuePromptResponse) {
    for
    {
        <-resp.Messages
    }
}

var commands = []*discordgo.ApplicationCommand{
    {
        Name:        "uploadlist",
        Description: "List the images available",
    },
    {
        Name:        "uploadimage",
        Description: "Say something through a bot",
        Options: []*discordgo.ApplicationCommandOption{
            {
                Name:        "image",
                Description: "Image to upload",
                Type:        discordgo.ApplicationCommandOptionAttachment,
                Required:    true,
            },
        },
    },
    {
        Name:        "txt2img",
        Description: "Generate an image from text",
        Options: []*discordgo.ApplicationCommandOption{
            {
                Name:        "positive",
                Description: "Positive prompt",
                Type:        discordgo.ApplicationCommandOptionString,
                Required:    true,
            },
            {
                Name:        "negative",
                Description: "Positive prompt",
                Type:        discordgo.ApplicationCommandOptionString,
                Required:    true,
            },
            {
                Name:        "seed",
                Description: "Seed",
                Type:        discordgo.ApplicationCommandOptionInteger,
                Required:    false,
            },
        },
    },
    {
        Name:        "faceswap",
        Description: "Generate an image from text",
        Options: []*discordgo.ApplicationCommandOption{
            {
                Name:        "positive",
                Description: "Positive prompt",
                Type:        discordgo.ApplicationCommandOptionString,
                Required:    true,
            },
            {
                Name:        "negative",
                Description: "Positive prompt",
                Type:        discordgo.ApplicationCommandOptionString,
                Required:    true,
            },
            {
                Name:         "image",
                Description:  "Face image",
                Type:         discordgo.ApplicationCommandOptionString,
                Required:     true,
                Autocomplete: true,
            },
            {
                Name:        "seed",
                Description: "Seed",
                Type:        discordgo.ApplicationCommandOptionInteger,
                Required:    false,
            },
        },
    },
}

func (bot *Bot) Start() error {
    var err error
    bot.session, err = discordgo.New("Bot " + bot.token)
    if err != nil {
        return err
    }
    bot.session.LogLevel = discordgo.LogWarning
    bot.session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
        log.Println("Bot up and running - id: " + r.User.ID)
    })
    bot.session.AddHandler(func(c *discordgo.Session, m *discordgo.MessageCreate) {
        log.Printf("MessageCreate.Message: %v", m.Message)
        log.Printf("MessageCreate.Content: %v", m.Content)
    })
    bot.session.AddHandler(func(c *discordgo.Session, m *discordgo.InteractionCreate) {
        switch m.Type {
        case discordgo.InteractionApplicationCommand:
            data := m.ApplicationCommandData()

            switch data.Name {
            case "uploadlist":
                bot.handleUploadList(c, m, parseOptions(data.Options))
            case "uploadimage":
                bot.handleUploadImage(c, m, parseOptions(data.Options))
            case "txt2img":
                bot.handleTxt2Img(c, m, parseOptions(data.Options))
            case "faceswap":
                bot.handleFaceSwap(c, m, parseOptions(data.Options))
            }
        case discordgo.InteractionMessageComponent:
            data := m.MessageComponentData()
            switch data.CustomID {
            case "txt2img-reseed":
                log.Println("reseed")
            }
        }

    })

    bot.session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuilds
    err = bot.session.Open()
    if err != nil {
        return err
    }

    log.Println("Adding commands...")
    registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
    for i, v := range commands {
        log.Println("Registering command " + v.Name)
        cmd, err := bot.session.ApplicationCommandCreate(bot.session.State.User.ID, "", v)
        if err != nil {
            log.Panicf("Cannot create '%v' command: %v", v.Name, err)
        }
        registeredCommands[i] = cmd
    }

    return nil
}

func (bot *Bot) Stop() error {
    err := bot.session.Close()
    if err != nil {
        return err
    }

    return nil
}

func (bot *Bot) GetProcessFromComfyClient(comfyClient *client.Client) *Process {
    for _, process := range bot.processes {
        if process.ComfyClient == comfyClient {
            return &process
        }
    }
    return nil
}
