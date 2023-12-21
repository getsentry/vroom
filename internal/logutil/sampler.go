package logutil

import (
	"github.com/rs/zerolog"
)

type LevelSampler struct {
	Level zerolog.Level
}

func (l LevelSampler) Sample(lvl zerolog.Level) bool {
	return lvl >= l.Level
}
