package service

import (
    "bytes"
    "fmt"
    "github.com/bwmarrin/discordgo"
    "github.com/tanis2000/comfy-client/client"
    "github.com/tanis2000/comfy-client/workflow"
    "log"
    "math/rand"
    "os"
    "strings"
)

func (b *Bot) initTxt2Img() (*client.Client, error) {
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
                if process, ok := b.processes[response.PromptID]; ok {
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
                                    Style: discordgo.SuccessButton},
                            },
                        },
                    }
                    if process, ok := b.processes[response.PromptID]; ok {
                        _, err := process.Session.FollowupMessageCreate(process.Interaction, true, &discordgo.WebhookParams{
                            Content:    "Image for prompt with ID: " + response.PromptID,
                            Files:      files,
                            Components: components,
                        })

                        if err != nil {
                            log.Printf("[txt2img] could not followup to interaction: %s", err)
                        }
                    }
                }
            }

        },
        OnProgress: func(c *client.Client, progress *client.WSStatusMessageDataProgress) {
            builder := strings.Builder{}
            process := b.GetProcessFromComfyClient(c)
            if process == nil {
                return
            }
            builder.WriteString(fmt.Sprintf("%d/%d", progress.Value, progress.Max))
            content := builder.String()
            _, err := process.Session.InteractionResponseEdit(process.Interaction, &discordgo.WebhookEdit{
                Content: &content,
            })
            if err != nil {
                log.Printf("could not edit: %s", err)
            }
        },
    }
    c, err := client.NewClient(b.comfyAddress, b.comfyPort, callbacks)
    if err != nil {
        return nil, err
    }
    return c, nil
}

func (b *Bot) handleTxt2Img(s *discordgo.Session, i *discordgo.InteractionCreate, opts optionMap) {
    comfyClient, err := b.initTxt2Img()
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
    b.processes[resp.PromptID] = Process{
        PromptID:    resp.PromptID,
        Interaction: i.Interaction,
        Session:     s,
        ComfyClient: comfyClient,
    }

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
