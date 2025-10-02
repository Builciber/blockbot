package main

import (
	"bytes"
	"fmt"
	"time"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

type PNLCardData struct {
	TokenPair      string
	PercentageGain string
	TradeDuration  string
	ReferralCode   string
	IsProfit       bool
}

const (
	cardWidth  = 800
	cardHeight = 600

	profitColor     = "#3CFE92"
	lossColor       = "#F54542"
	labelColor      = "#9F9F9F"
	backgroundColor = "#1a1a1a"
)

func generatePNLCard(data PNLCardData) (*bytes.Buffer, error) {
	dc := gg.NewContext(cardWidth, cardHeight)

	dc.SetHexColor(backgroundColor)
	dc.Clear()

	regularFont, err := truetype.Parse(goregular.TTF)
	if err != nil {
		return nil, fmt.Errorf("failed to parse regular font: %w", err)
	}

	boldFont, err := truetype.Parse(gobold.TTF)
	if err != nil {
		return nil, fmt.Errorf("failed to parse bold font: %w", err)
	}

	mainColor := profitColor
	if !data.IsProfit {
		mainColor = lossColor
	}

	tickerFace := truetype.NewFace(boldFont, &truetype.Options{Size: 48})
	dc.SetFontFace(tickerFace)
	dc.SetHexColor(mainColor)
	dc.DrawStringAnchored(data.TokenPair, cardWidth/2, 150, 0.5, 0.5)

	percentageFace := truetype.NewFace(boldFont, &truetype.Options{Size: 72})
	dc.SetFontFace(percentageFace)
	dc.SetHexColor(mainColor)
	dc.DrawStringAnchored(data.PercentageGain, cardWidth/2, 250, 0.5, 0.5)

	labelFace := truetype.NewFace(regularFont, &truetype.Options{Size: 20})
	dc.SetFontFace(labelFace)
	dc.SetHexColor(labelColor)
	dc.DrawStringAnchored("Unrealized PnL", cardWidth/2, 320, 0.5, 0.5)

	durationFace := truetype.NewFace(regularFont, &truetype.Options{Size: 16})
	dc.SetFontFace(durationFace)
	dc.SetHexColor(labelColor)
	dc.DrawStringAnchored(fmt.Sprintf("Trade Duration (%s)", data.TradeDuration), cardWidth/2, 370, 0.5, 0.5)

	refCodeFace := truetype.NewFace(boldFont, &truetype.Options{Size: 32})
	dc.SetFontFace(refCodeFace)
	dc.SetHexColor(mainColor)
	dc.DrawStringAnchored(fmt.Sprintf("Referral Code (%s)", data.ReferralCode), cardWidth/2, 480, 0.5, 0.5)

	refLabelFace := truetype.NewFace(regularFont, &truetype.Options{Size: 16})
	dc.SetFontFace(refLabelFace)
	dc.SetHexColor(labelColor)
	dc.DrawStringAnchored("Share your code and earn rewards!", cardWidth/2, 530, 0.5, 0.5)

	buf := new(bytes.Buffer)
	if err := dc.EncodePNG(buf); err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %w", err)
	}

	return buf, nil
}

func generateProfitCard(tokenPair string, percentageGain float64, tradeDuration time.Duration, referralCode string) (*bytes.Buffer, error) {
	durationStr := formatDuration(tradeDuration)
	percentStr := fmt.Sprintf("+%.2f%%", percentageGain)

	data := PNLCardData{
		TokenPair:      tokenPair,
		PercentageGain: percentStr,
		TradeDuration:  durationStr,
		ReferralCode:   referralCode,
		IsProfit:       true,
	}

	return generatePNLCard(data)
}

func generateLossCard(tokenPair string, percentageLoss float64, tradeDuration time.Duration, referralCode string) (*bytes.Buffer, error) {
	durationStr := formatDuration(tradeDuration)
	percentStr := fmt.Sprintf("%.2f%%", percentageLoss)

	data := PNLCardData{
		TokenPair:      tokenPair,
		PercentageGain: percentStr,
		TradeDuration:  durationStr,
		ReferralCode:   referralCode,
		IsProfit:       false,
	}

	return generatePNLCard(data)
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
