package faceswap

import (
    "bytes"
    "discomfort/service"
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
    return "faceswap"
}

func (h Handler) GetApplicationCommand() *discordgo.ApplicationCommand {
    return &discordgo.ApplicationCommand{
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
    }
}

func (h Handler) HandleCommand(ctx service.Context, s *discordgo.Session, i *discordgo.InteractionCreate, opts map[string]*discordgo.ApplicationCommandInteractionDataOption) {
    comfyClient, err := h.setup(ctx)
    if err != nil {
        log.Printf("Cannot initialize the ComfyUI client: %s", err)
        return
    }
    //err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
    //    Type: discordgo.InteractionResponseDeferredMessageUpdate,
    //})
    //if err != nil {
    //    log.Printf("could not respond to interaction: %s", err)
    //}

    switch i.Type {
    case discordgo.InteractionApplicationCommand:
        data := i.ApplicationCommandData()
        log.Printf("%v", data)
        builder := new(strings.Builder)

        positive := opts["positive"].Value.(string)
        negative := opts["negative"].Value.(string)
        image := opts["image"].Value.(string)
        seed := rand.Uint64()
        if val, ok := opts["seed"]; ok {
            seed = val.UintValue()
        }

        buf, err := os.ReadFile("workflows/faceswap/faceswap_api.json")
        if err != nil {
            log.Printf("Cannot read workflow workflows/faceswap/faceswap_api.json: %s", err)
            return
        }
        wf, err := workflow.NewWorkflow(string(buf))
        if err != nil {
            log.Printf("Cannot create workflow: %s", err)
            return
        }

        positiveNode := wf.NodeByID("7")
        if positiveNode != nil {
            positiveNode.Inputs.Set("text", positive)
        }

        negativeNode := wf.NodeByID("8")
        if negativeNode != nil {
            negativeNode.Inputs.Set("text", negative)
        }

        imageNode := wf.NodeByID("1")
        if imageNode != nil {
            imageNode.Inputs.Set("image", image)
        }

        samplerNode := wf.NodeByID("9")
        if samplerNode != nil {
            samplerNode.Inputs.Set("seed", seed)
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
        ctx.Bot.AddProcess(resp.PromptID, service.Process{
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

    case discordgo.InteractionApplicationCommandAutocomplete:
        choices := make([]*discordgo.ApplicationCommandOptionChoice, 0)
        for _, image := range ctx.Bot.GetImages() {
            choice := &discordgo.ApplicationCommandOptionChoice{
                Name:  image,
                Value: image,
            }
            if opts["image"].StringValue() != "" {
                if strings.Contains(image, opts["image"].StringValue()) {
                    choices = append(choices, choice)
                }
            } else {
                choices = append(choices, choice)
            }
        }
        err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
            Type: discordgo.InteractionApplicationCommandAutocompleteResult,
            Data: &discordgo.InteractionResponseData{
                Choices: choices,
            },
        })
        if err != nil {
            log.Printf("could not respond to interaction: %s", err)
        }
    }
}

func (h Handler) setup(ctx service.Context) (*client.Client, error) {
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
                    embed := service.EmbedTemplate()
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
                            Value:  fmt.Sprintf("<@%s>", service.GetDiscordUserId(process.InteractionCreate)),
                            Inline: true,
                        },
                    }

                    embeds := make([]*discordgo.MessageEmbed, 0)
                    embeds = append(embeds, embed)
                    process := ctx.Bot.GetProcessByPromptID(response.PromptID)
                    if process != nil {
                        content := "Image for prompt with ID: " + response.PromptID
                        _, err := process.Session.InteractionResponseEdit(process.InteractionCreate.Interaction, &discordgo.WebhookEdit{
                            Content: &content,
                            Embeds:  &embeds,
                            Files:   files,
                        })

                        if err != nil {
                            log.Printf("could not respond to interaction: %s", err)
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
