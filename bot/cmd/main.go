package main

import (
	"fmt"
	"nelchanbot"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	nelchan, err := nelchanbot.NewNelchan()
	if err != nil {
		fmt.Println("ねるちゃんの起動に失敗しました:", err)
		return
	}

	err = nelchan.Start()
	if err != nil {
		fmt.Println("ねるちゃんの起動に失敗しました:", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("ねるちゃんが起動しました。 CTRL-C で停止します。")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	err = nelchan.Close()
	if err != nil {
		fmt.Println("ねるちゃんの停止に失敗しました:", err)
		return
	}
}

// This function will be called (due to AddHandler above) every time a new
