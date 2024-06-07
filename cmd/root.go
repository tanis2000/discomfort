package cmd

import (
    "discomfort/service"
    "discomfort/service/handler/faceswap"
    "discomfort/service/handler/txt2img"
    "discomfort/service/handler/uploadimage"
    "discomfort/service/handler/uploadlist"
    "fmt"
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
    "log"
    "os"
    "os/signal"
    "syscall"
)

var discordToken string
var comfyAddress string
var comfyPort int

var rootCmd = &cobra.Command{
    Use:   "discomfort",
    Short: "discomfort is a Discord bot to control ComfyUI",
    Long:  `Your Discord bot for ComfyUI management`,
    Run: func(cmd *cobra.Command, args []string) {
        var desiredHandlers = []service.Handler{
            txt2img.Handler{},
            faceswap.Handler{},
            uploadimage.Handler{},
            uploadlist.Handler{},
        }
        bot := service.NewBot(discordToken, comfyAddress, comfyPort, desiredHandlers)
        err := bot.Start()
        if err != nil {
            log.Fatal(err.Error())
        }
        sc := make(chan os.Signal, 1)
        signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
        log.Println("Press Ctrl+C to exit")
        <-sc
        log.Println("Gracefully shutting down.")
        err = bot.Stop()
        if err != nil {
            log.Fatal(err.Error())
        }
    },
}

func init() {
    cobra.OnInitialize(initConfig)
    rootCmd.PersistentFlags().StringVar(&discordToken, "token", "", "Discord API token")
    rootCmd.PersistentFlags().StringVar(&comfyAddress, "address", "127.0.0.1", "ComfyUI address (IP or hostname)")
    rootCmd.PersistentFlags().IntVar(&comfyPort, "port", 8188, "ComfyUI port")
    err := viper.BindPFlag("token", rootCmd.PersistentFlags().Lookup("token"))
    if err != nil {
        log.Fatal(err)
    }
    err = viper.BindPFlag("address", rootCmd.PersistentFlags().Lookup("address"))
    if err != nil {
        log.Fatal(err)
    }
    err = viper.BindPFlag("port", rootCmd.PersistentFlags().Lookup("port"))
    if err != nil {
        log.Fatal(err)
    }
}

func initConfig() {
    viper.AutomaticEnv()
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        log.Println(err.Error())
        fmt.Fprintln(os.Stderr, err)
        os.Exit(2)
    }
}
