package service

import (
    "bytes"
    "fmt"
    "github.com/bwmarrin/discordgo"
    "github.com/tanis2000/comfy-client/client"
    "io"
    "log"
    "net/http"
    "strings"
)

func (b *Bot) initUploadImage() (*client.Client, error) {
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
                    if process, ok := b.processes[response.PromptID]; ok {
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

func (b *Bot) handleUploadImage(s *discordgo.Session, i *discordgo.InteractionCreate, opts optionMap) {
    comfyClient, err := b.initUploadImage()
    if err != nil {
        log.Printf("Cannot initialize the ComfyUI client: %s", err)
        return
    }

    index := opts["image"].Value.(string)
    attachment := i.Data.(discordgo.ApplicationCommandInteractionData).Resolved.Attachments[index]
    imageURL := attachment.URL
    log.Println("Image URL: " + imageURL)

    resp, err := http.Get(imageURL)
    if err != nil {
        log.Panicf("could not download image: %s", err)
    }
    defer resp.Body.Close()

    upload, err := comfyClient.Upload(io.Reader(resp.Body), attachment.Filename, true, client.InputImageType, "")
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
