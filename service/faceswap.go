package service

import (
    "fmt"
    "github.com/bwmarrin/discordgo"
    "github.com/tanis2000/comfy-client/workflow"
    "log"
    "os"
    "strings"
)

func (b *Bot) handleFaceSwap(s *discordgo.Session, i *discordgo.InteractionCreate, opts optionMap) {
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

        buf, err := os.ReadFile("workflows/faceswap.json")
        if err != nil {
            log.Printf("Cannot read workflow faceswap.json: %s", err)
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

        builder.WriteString("Queued prompt with ID: " + resp.PromptID + "\n")
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

    case discordgo.InteractionApplicationCommandAutocomplete:
        choices := make([]*discordgo.ApplicationCommandOptionChoice, 0)
        for _, image := range b.imageDB.Images {
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
        fmt.Printf("Choices: %v\n", choices[0].Name)
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
