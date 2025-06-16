package synchronizer

import (
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
	"github.com/rs/zerolog/pkgerrors"
	"os"
	"time"
)

var log zerolog.Logger

func init() {
	wr := diode.NewWriter(os.Stdout, 1000, 10*time.Millisecond, func(missed int) {
		fmt.Printf("Logger Dropped %d messages", missed)
	})
	zerolog.TimeFieldFormat = time.RFC1123
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	log = zerolog.New(wr).With().Timestamp().Caller().Logger()
}
