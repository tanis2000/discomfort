package txt2img

import (
    "bytes"
    service2 "discomfort/internal/service"
    "fmt"
    "github.com/bwmarrin/discordgo"
    "github.com/tanis2000/comfy-client/client"
    "github.com/tanis2000/comfy-client/workflow"
    "log"
    "math/rand"
    "os"
    "strings"
)

type Handler struct {
}

func (h Handler) GetName() string {
    return "txt2img"
}

func (h Handler) GetApplicationCommand() *discordgo.ApplicationCommand {
    return &discordgo.ApplicationCommand{
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
    }
}

func (h Handler) HandleCommand(ctx service2.Context, s *discordgo.Session, i *discordgo.InteractionCreate, opts map[string]*discordgo.ApplicationCommandInteractionDataOption) {
    comfyClient, err := h.setup(ctx)
    if err != nil {
        log.Printf("Cannot initialize the ComfyUI client: %s", err)
        return
    }
    builder := new(strings.Builder)

    positive := opts["positive"].Value.(string)
    negative := opts["negative"].Value.(string)
    seed := rand.Uint64()
    if val, ok := opts["seed"]; ok {
        seed = val.UintValue()
    }

    buf, err := os.ReadFile("workflows/txt2img/txt2img_api.json")
    if err != nil {
        log.Printf("Cannot read workflow workflows/txt2img/txt2img_api.json: %s", err)
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

    sampler1Node := wf.NodeByID("9")
    if sampler1Node != nil {
        sampler1Node.Inputs.Set("noise_seed", seed)
    }

    sampler2Node := wf.NodeByID("11")
    if sampler2Node != nil {
        sampler2Node.Inputs.Set("noise_seed", seed)
    }

    resp, err := comfyClient.QueuePrompt(-1, wf)
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
    ctx.Bot.AddProcess(resp.PromptID, service2.Process{
        PromptID:          resp.PromptID,
        InteractionCreate: i,
        Session:           s,
        ComfyClient:       comfyClient,
        PositivePrompt:    positive,
        NegativePrompt:    negative,
        Seed:              seed,
        NodesCount:        len(wf.Map),
    })

    builder.WriteString("Queued prompt with ID: " + resp.PromptID + "\n")
    builder.WriteString("Positive prompt: " + positive + "\n")
    builder.WriteString("Negative prompt: " + negative + "\n")
    builder.WriteString(fmt.Sprintf("Seed: %d\n", seed))

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

func (h Handler) setup(ctx service2.Context) (*client.Client, error) {
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
                process := ctx.Bot.GetProcessByPromptID(response.PromptID)
                if process != nil {
                    _, err := process.Session.FollowupMessageCreate(process.InteractionCreate.Interaction, true, &discordgo.WebhookParams{
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
                    buf, err := c.GetView(image.Filename, image.Subfolder, image.Type)
                    if err != nil {
                        log.Printf("could not get view: %s", err)
                    }
                    files := make([]*discordgo.File, 0)
                    file := &discordgo.File{Name: image.Filename,
                        ContentType: "image/jpeg",
                        Reader:      bytes.NewReader(buf),
                    }
                    files = append(files, file)

                    components := []discordgo.MessageComponent{
                        discordgo.ActionsRow{
                            Components: []discordgo.MessageComponent{
                                discordgo.Button{
                                    CustomID: "txt2img-reseed",
                                    Label:    "Reseed with a random value",
                                    Emoji: &discordgo.ComponentEmoji{
                                        Name: "🎲",
                                    },
                                    Style: discordgo.SuccessButton,
                                },
                            },
                        },
                    }

                    embed := service2.EmbedTemplate()
                    embed.Image = &discordgo.MessageEmbedImage{
                        URL: fmt.Sprintf("attachment://%s", files[0].Name),
                    }
                    embed.Fields = []*discordgo.MessageEmbedField{
                        {
                            Name:  "Prompt",
                            Value: process.PositivePrompt,
                        },
                        {
                            Name:  "Negative prompt",
                            Value: process.NegativePrompt,
                        },
                        {
                            Name:  "Seed",
                            Value: fmt.Sprintf("%d", process.Seed),
                        },
                        {
                            Name:   "User",
                            Value:  fmt.Sprintf("<@%s>", service2.GetDiscordUserId(process.InteractionCreate)),
                            Inline: true,
                        },
                    }

                    embeds := make([]*discordgo.MessageEmbed, 0)
                    embeds = append(embeds, embed)

                    process := ctx.Bot.GetProcessByPromptID(response.PromptID)
                    if process != nil {
                        content := "Image for prompt with ID: " + response.PromptID
                        _, err := process.Session.InteractionResponseEdit(process.InteractionCreate.Interaction, &discordgo.WebhookEdit{
                            Content:    &content,
                            Embeds:     &embeds,
                            Files:      files,
                            Components: &components,
                        })

                        if err != nil {
                            log.Printf("[txt2img] could not followup to interaction: %s", err)
                        }
                    }
                }
            } else {
                process := ctx.Bot.GetProcessByPromptID(response.PromptID)
                if process == nil {
                    return
                }
                process.CurrentNode = node
                content := fmt.Sprintf("Running node %s/%d...", node, process.NodesCount)
                _, err := process.Session.InteractionResponseEdit(process.InteractionCreate.Interaction, &discordgo.WebhookEdit{
                    Content: &content,
                })

                if err != nil {
                    log.Printf("[txt2img] could not followup to interaction: %s", err)
                }
            }

        },
        OnProgress: func(c *client.Client, progress *client.WSStatusMessageDataProgress) {
            builder := strings.Builder{}
            process := ctx.Bot.GetProcessFromComfyClient(c)
            if process == nil {
                return
            }
            builder.WriteString(fmt.Sprintf("Running node %s/%d\nStep %d/%d", process.CurrentNode, process.NodesCount, progress.Value, progress.Max))
            content := builder.String()
            _, err := process.Session.InteractionResponseEdit(process.InteractionCreate.Interaction, &discordgo.WebhookEdit{
                Content: &content,
            })
            if err != nil {
                log.Printf("could not edit: %s", err)
            }
        },
    }
    c, err := client.NewClient(ctx.Bot.GetComfyAddress(), ctx.Bot.GetComfyPort(), callbacks)
    if err != nil {
        return nil, err
    }
    return c, nil
}

func consumeMessages(resp *client.QueuePromptResponse) {
    for
    {
        <-resp.Messages
    }
}
