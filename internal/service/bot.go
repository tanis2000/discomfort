package service

import (
    "discomfort/internal/database"
    "discomfort/internal/imagedb"
    "errors"
    "github.com/bwmarrin/discordgo"
    "github.com/tanis2000/comfy-client/client"
    "log"
)

type Bot struct {
    session            *discordgo.Session
    token              string
    comfyAddress       string
    comfyPort          int
    imageDB            *imagedb.ImageDB
    processes          map[string]Process
    desiredHandlers    []Handler
    registeredHandlers map[string]Handler
    // availableCommands contains the list of commands that the bot wants to register
    availableCommands []*discordgo.ApplicationCommand
    // registeredCommands contains the list of the commands that the bot has successfully registered with Discord
    registeredCommands []*discordgo.ApplicationCommand
    // existingCommands contains the list of the commands that have been already previously registered with Discord. This is used to remove commands that are no longer available
    existingCommands []*discordgo.ApplicationCommand
    context          Context
    db               *database.Database
}

type optionMap = map[string]*discordgo.ApplicationCommandInteractionDataOption

func parseOptions(options []*discordgo.ApplicationCommandInteractionDataOption) (om optionMap) {
    om = make(optionMap)
    for _, opt := range options {
        om[opt.Name] = opt
    }
    return
}

func NewBot(token string, comfyAddress string, comfyPort int, desiredHandlers []Handler, db *database.Database) *Bot {
    res := &Bot{
        token:              token,
        comfyAddress:       comfyAddress,
        comfyPort:          comfyPort,
        imageDB:            imagedb.NewImageDB(),
        processes:          map[string]Process{},
        availableCommands:  make([]*discordgo.ApplicationCommand, 0),
        registeredCommands: make([]*discordgo.ApplicationCommand, 0),
        existingCommands:   make([]*discordgo.ApplicationCommand, 0),
        desiredHandlers:    desiredHandlers,
        registeredHandlers: make(map[string]Handler),
        context:            Context{},
        db:                 db,
    }
    res.context.Bot = res
    res.registerHandlers()
    res.addAvailableCommands()
    res.imageDB.Load()
    return res
}

func (b *Bot) registerHandlers() {
    for _, h := range b.desiredHandlers {
        if err := b.registerHandler(h); err != nil {
            log.Fatalf("Failed to register handler '%s': %v", h.GetName(), err)
        }
    }
}

func (b *Bot) registerHandler(handler Handler) error {
    if handler == nil {
        return errors.New("cannot register nil handler")
    }
    log.Printf("Registering handler '%s'", handler.GetName())
    if _, ok := b.registeredHandlers[handler.GetName()]; ok {
        log.Panicf("Handler '%s' already registered", handler.GetName())
    }
    b.registeredHandlers[handler.GetName()] = handler
    return nil
}

func (b *Bot) addAvailableCommands() {
    for _, h := range b.registeredHandlers {
        b.availableCommands = append(b.availableCommands, h.GetApplicationCommand())
    }
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

            for _, h := range bot.registeredHandlers {
                if h.GetName() == data.Name {
                    h.HandleCommand(bot.context, c, m, parseOptions(data.Options))
                }
            }
        case discordgo.InteractionApplicationCommandAutocomplete:
            data := m.ApplicationCommandData()

            for _, h := range bot.registeredHandlers {
                if h.GetName() == data.Name {
                    h.HandleCommand(bot.context, c, m, parseOptions(data.Options))
                }
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

    bot.SyncCommands()

    return nil
}

func (bot *Bot) SyncCommands() {
    commands, err := bot.session.ApplicationCommands(bot.session.State.User.ID, "")
    bot.registeredCommands = commands
    if err != nil {
        log.Panicf("Cannot get commands: %v", err)
    }
    if len(bot.registeredCommands) > 0 {
        for _, v := range bot.registeredCommands {
            if bot.IsCommandAvailable(v.Name) {
                log.Printf("'%s' is in the available commands, skipping it...", v.Name)
                continue
            }
            log.Printf("Removing obsolete command '%s'...", v.Name)
            err := bot.session.ApplicationCommandDelete(bot.session.State.User.ID, "", v.ID)
            if err != nil {
                log.Panicf("Cannot remove obsolete command '%s': %s", v.Name, err)
            }
        }
    }

    log.Println("Registering commands with Discord...")
    for _, v := range bot.availableCommands {
        log.Printf("Checking if command '%s' needs to be updated", v.Name)
        if !bot.IsCommandUpdateNeeded(v) {
            log.Printf("Skipping command '%s'", v.Name)
            continue
        }

        for _, r := range bot.registeredCommands {
            if r.Name == v.Name {
                log.Printf("Removing old command '%s' id '%s", v.Name, r.ID)
                err := bot.session.ApplicationCommandDelete(bot.session.State.User.ID, "", r.ID)
                if err != nil {
                    log.Panicf("Cannot remove old command '%s': %s", v.Name, err)
                }
            }
        }

        log.Println("Registering command " + v.Name)
        cmd, err := bot.session.ApplicationCommandCreate(bot.session.State.User.ID, "", v)
        if err != nil {
            log.Panicf("Cannot create command '%s': %s", v.Name, err)
        }
        log.Printf("Registered command '%s' with ID '%s'", v.Name, cmd.ID)
    }
    updatedCommands, err := bot.session.ApplicationCommands(bot.session.State.User.ID, "")
    if err != nil {
        log.Panicf("Cannot get commands: %v", err)
    }
    bot.registeredCommands = updatedCommands

}

func (bot *Bot) IsCommandAvailable(name string) bool {
    for _, v := range bot.availableCommands {
        if v.Name == name {
            return true
        }
    }
    return false
}

func (bot *Bot) IsCommandUpdateNeeded(command *discordgo.ApplicationCommand) bool {
    for _, registeredCommand := range bot.registeredCommands {
        if registeredCommand.Name == command.Name {
            if registeredCommand.Description != command.Description {
                return true
            }
            if len(registeredCommand.Options) != len(command.Options) {
                return true
            }
            for i, option := range command.Options {
                if option.Description != registeredCommand.Options[i].Description {
                    // log.Println("Registered description '", registeredCommand.Options[i].Description, "' is different from command description '", option.Description, "' for command", command.Name)
                    return true
                }
                //if option.Autocomplete {
                //    return true
                //}
                for k, choice := range option.Choices {
                    if len(registeredCommand.Options[i].Choices) != len(option.Choices) {
                        // log.Println("Length of choices is different for command", command.Name)
                        // log.Println("Registered command:", registeredCommand.Options[i].Choices)
                        // // print all the choices names and their description
                        // for _, v := range command.Options[i].Choices {
                        // 	log.Println("Command", v.Name, v.Value)
                        // }
                        // for _, v := range registeredCommand.Options[i].Choices {
                        // 	log.Println("RegisteredCommand", v.Name, v.Value)
                        // }
                        return true
                    }
                    if choice.Name == registeredCommand.Options[i].Choices[k].Name {
                        if choice.Value != registeredCommand.Options[i].Choices[k].Value {
                            continue
                        }
                    } else {
                        return false
                    }
                }
                if option.MinValue != registeredCommand.Options[i].MinValue || option.MaxValue != registeredCommand.Options[i].MaxValue {
                    return true
                }
            }
            return false
        }
    }
    return true
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

func (bot *Bot) GetProcessByPromptID(promptID string) *Process {
    for _, process := range bot.processes {
        if process.PromptID == promptID {
            return &process
        }
    }
    return nil
}

func (bot *Bot) GetComfyAddress() string {
    return bot.comfyAddress
}

func (bot *Bot) GetComfyPort() int {
    return bot.comfyPort
}

func (b *Bot) AddProcess(promptID string, process Process) {
    b.processes[promptID] = process
}

func (bot *Bot) GetImages() []string {
    return bot.imageDB.Images
}

func (b *Bot) AddImage(image string) {
    b.imageDB.Add(image)
}

func (b *Bot) SaveImageDB() {
    b.imageDB.Save()
}

func (b *Bot) GetDatabase() *database.Database {
    return b.db
}
