package main

import (
	"github.com/rs/zerolog/log"

	"github.com/ssup2/kcchecker/pkg/cmd/kcchecker"
)

func main() {
	// Run command
	cmd, err := kcchecker.New()
	if err != nil {
		log.Panic().Err(err).Msg("Failed to get kcchecker error")
	}
	if err := cmd.Execute(); err != nil {
		log.Panic().Err(err).Msg("Failed to execute kcchecker error")
	}
}
