package reputation

import (
	"regexp"

	"github.com/go-telegram/bot/models"
)

var trigger = regexp.MustCompile(`(?i)^([+]+|[-]+)(rep|реп)(?:$|\s|[[:punct:]])`)

type Trigger struct {
	Delta int
}

var (
	plusRepStickerIDs = map[string]string{
		"AgADlYwAArYY6Uo": "MR P.K.",
		"AgADUKMAAnEM8Uo": "Kopatich",
	}
	minusRepStickerIDs = map[string]string{
		"AgADTpAAAiB38Eo": "MR P.K.",
		"AgADWJcAAvBI-Uo": "Losyash",
	}
)

func Parse(msg *models.Message) *Trigger {
	if msg.Sticker != nil {
		return parseSticker(msg.Sticker)
	}
	return parseText(msg.Text)
}

func parseSticker(sticker *models.Sticker) *Trigger {
	if _, ok := plusRepStickerIDs[sticker.FileUniqueID]; ok {
		return &Trigger{Delta: 1}
	}
	if _, ok := minusRepStickerIDs[sticker.FileUniqueID]; ok {
		return &Trigger{Delta: -1}
	}
	return nil
}

func parseText(text string) *Trigger {
	m := trigger.FindStringSubmatch(text)
	if m == nil {
		return nil
	}
	signs := m[1]
	n := len(signs)
	if signs[0] == '-' {
		n = -n
	}
	return &Trigger{Delta: n}
}
