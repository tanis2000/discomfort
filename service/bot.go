package service

import (
    "bytes"
    "github.com/bwmarrin/discordgo"
    "github.com/tanis2000/comfy-client/client"
    "github.com/tanis2000/comfy-client/workflow"
    "io"
    "log"
    "net/http"
    "os"
    "strings"
)

type Bot struct {
    session     *discordgo.Session
    token       string
    imageDB     *ImageDB
    comfyClient *client.Client
    wsc         *client.WebSocketClient
    processes   map[string]Process
}

type optionMap = map[string]*discordgo.ApplicationCommandInteractionDataOption

func parseOptions(options []*discordgo.ApplicationCommandInteractionDataOption) (om optionMap) {
    om = make(optionMap)
    for _, opt := range options {
        om[opt.Name] = opt
    }
    return
}

func NewBot(token string) *Bot {
    res := &Bot{
        token:     token,
        imageDB:   NewImageDB(),
        processes: map[string]Process{},
    }
    res.imageDB.Load()
    callbacks := &client.Callbacks{
        OnStatus: func(c *client.Client, queuedCount int) {
            log.Printf("Queue size: %d", queuedCount)
        },
        OnExecutionStart: func(c *client.Client, response *client.QueuePromptResponse) {
            log.Printf("Execution started for prompt with ID: %s", response.PromptID)
        },
        OnExecuted: func(c *client.Client, response *client.QueuePromptResponse) {
            log.Printf("Execution finished for prompt with ID: %s", response.PromptID)
        },
        OnExecuting: func(c *client.Client, response *client.QueuePromptResponse, node string) {
            log.Printf("Executing node %s with prompt with ID: %s", node, response.PromptID)
            if node == "" {
                if process, ok := res.processes[response.PromptID]; ok {
                    _, err := process.Session.FollowupMessageCreate(process.Interaction, true, &discordgo.WebhookParams{
                        Content: "Execution completed for prompt with ID: " + response.PromptID,
                    })

                    if err != nil {
                        log.Printf("could not respond to interaction: %s", err)
                    }
                }

                history, err := c.GetHistoryByPromptID(response.PromptID, 200)
                if err != nil {
                    log.Printf("could not get history: %s", err)
                }

                historyContent := (*history)[response.PromptID]
                images := historyContent.GetImagesByType("output")
                if len(images) == 0 {
                    log.Printf("No images found with type output. Falling back to type temp")
                    images = historyContent.GetImagesByType("temp")
                    if len(images) == 0 {
                        log.Printf("No images found with type temp")
                    }
                }
                for _, image := range images {
                    log.Printf("%s %s %s", image.Filename, image.Subfolder, image.Type)
                    b, err := c.GetView(image.Filename, image.Subfolder, image.Type)
                    if err != nil {
                        log.Printf("could not get view: %s", err)
                    }
                    files := make([]*discordgo.File, 0)
                    file := &discordgo.File{Name: image.Filename,
                        ContentType: "image/jpeg",
                        Reader:      bytes.NewReader(b),
                    }
                    files = append(files, file)
                    if process, ok := res.processes[response.PromptID]; ok {
                        _, err := process.Session.FollowupMessageCreate(process.Interaction, true, &discordgo.WebhookParams{
                            Content: "Image for prompt with ID: " + response.PromptID,
                            Files:   files,
                        })

                        if err != nil {
                            log.Printf("could not respond to interaction: %s", err)
                        }
                    }
                }
            }

        },
    }
    res.comfyClient = client.NewClient("127.0.0.1", 8188, callbacks)
    res.wsc = client.NewWebSocketClient(res.comfyClient)
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

func (b *Bot) handleUploadImage(s *discordgo.Session, i *discordgo.InteractionCreate, opts optionMap) {
    index := opts["image"].Value.(string)
    attachment := i.Data.(discordgo.ApplicationCommandInteractionData).Resolved.Attachments[index]
    imageURL := attachment.URL
    log.Println("Image URL: " + imageURL)

    resp, err := http.Get(imageURL)
    if err != nil {
        log.Panicf("could not download image: %s", err)
    }
    defer resp.Body.Close()

    upload, err := b.comfyClient.Upload(io.Reader(resp.Body), attachment.Filename, true, client.InputImageType, "")
    if err != nil {
        log.Panicf("could not upload image: %s", err)
    }

    b.imageDB.Add(upload)
    b.imageDB.Save()

    builder := new(strings.Builder)
    builder.WriteString(upload + " uploaded.")

    err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseChannelMessageWithSource,
        Data: &discordgo.InteractionResponseData{
            Content: builder.String(),
        },
    })

    if err != nil {
        log.Panicf("could not respond to interaction: %s", err)
    }
}

func (b *Bot) handleTxt2Img(s *discordgo.Session, i *discordgo.InteractionCreate, opts optionMap) {
    builder := new(strings.Builder)

    positive := opts["positive"].Value.(string)
    negative := opts["negative"].Value.(string)

    buf, err := os.ReadFile("workflows/txt2img.json")
    if err != nil {
        log.Printf("Cannot read workflow txt2img.json: %s", err)
        return
    }
    wf, err := workflow.NewWorkflow(string(buf))
    if err != nil {
        log.Printf("Cannot create workflow: %s", err)
        return
    }

    positiveNode := wf.NodeByID("3")
    if positiveNode != nil {
        positiveNode.Inputs.Set("text_g", positive)
        positiveNode.Inputs.Set("text_l", positive)
    }

    positiveNode = wf.NodeByID("7")
    if positiveNode != nil {
        positiveNode.Inputs.Set("text", positive)
    }

    negativeNode := wf.NodeByID("5")
    if negativeNode != nil {
        negativeNode.Inputs.Set("text_g", negative)
        negativeNode.Inputs.Set("text_l", negative)
    }

    negativeNode = wf.NodeByID("8")
    if negativeNode != nil {
        negativeNode.Inputs.Set("text", negative)
    }

    resp, err := b.comfyClient.QueuePrompt(-1, wf)
    if err != nil {
        builder.WriteString("could not queue prompt: " + err.Error())
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

    go consumeMessages(resp)
    b.processes[resp.PromptID] = Process{
        PromptID:    resp.PromptID,
        Interaction: i.Interaction,
        Session:     s,
    }

    builder.WriteString("Queued prompt with ID: " + resp.PromptID)
    builder.WriteString("Positive prompt: " + positive + "\n")
    builder.WriteString("Negative prompt: " + negative + "\n")

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
        log.Printf("%v", m.Message)
        log.Printf("%v", m.Content)
    })
    bot.session.AddHandler(func(c *discordgo.Session, m *discordgo.InteractionCreate) {
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

    err = bot.StartComfyWebSocket()
    if err != nil {
        return err
    }

    return nil
}

func (bot *Bot) StartComfyWebSocket() error {
    err := bot.wsc.Connect("127.0.0.1", 8188, bot.comfyClient.ClientId())
    if err != nil {
        return err
    }

    go func() {
        println("Handling messages")
        bot.wsc.HandleMessages()
    }()

    /*
       println("Starting the websocket loop")
       for continueLoop := true; continueLoop; {
           msg := <-res.Messages
           println(msg)
       }
    */
    return nil
}

func (bot *Bot) Stop() error {
    err := bot.session.Close()
    if err != nil {
        return err
    }

    err = bot.wsc.Close()
    if err != nil {
        return err
    }

    return nil
}
