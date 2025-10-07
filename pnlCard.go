package main

import (
	"bytes"
	"fmt"
	"image"
	"os"
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
	BackgroundPath string
}

const (
	cardWidth  = 1151
	cardHeight = 768

	profitColor = "#3CFE92"
	lossColor   = "#F54542"
	labelColor  = "#9F9F9F"
)

var (
	profitBackground = []string{
		"./template/PNL Card Testnet - Profit 1 (Empty).jpg",
		"./template/PNL Card Testnet - Profit 8 (Empty).jpg",
		"./template/PNL Card Testnet - Profit 4 (Empty).jpg",
	}
	lossBackgrounds = []string{
		"./template/PNL Card Testnet - Loss 1 (Empty).jpg",
		"./template/PNL Card Testnet - Loss 3 (Empty).jpg",
		"./template/PNL Card Testnet - Loss 5 (Empty).jpg",
	}
)

func generatePNLCard(data PNLCardData) (*bytes.Buffer, error) {
	dc := gg.NewContext(cardWidth, cardHeight)

	bgFile, err := os.Open(data.BackgroundPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open background: %w", err)
	}
	defer bgFile.Close()

	img, _, err := image.Decode(bgFile)
	if err != nil {
		return nil, fmt.Errorf("failed to decode background: %w", err)
	}
	dc.DrawImageAnchored(img, cardWidth/2, cardHeight/2, 0.5, 0.5)

	// 🎨 Load fonts
	regularFont, _ := truetype.Parse(goregular.TTF)
	boldFont, _ := truetype.Parse(gobold.TTF)

	mainColor := profitColor
	if !data.IsProfit {
		mainColor = lossColor
	}

	// Token Pair
	dc.SetFontFace(truetype.NewFace(boldFont, &truetype.Options{Size: 48}))
	dc.SetHexColor(mainColor)
	dc.DrawStringAnchored(data.TokenPair, 300, 250, 0.5, 0.5)

	// Percentage Gain / Loss
	dc.SetFontFace(truetype.NewFace(boldFont, &truetype.Options{Size: 72}))
	dc.DrawStringAnchored(data.PercentageGain, 300, 360, 0.5, 0.5)

	//  Unrealized PnL
	dc.SetFontFace(truetype.NewFace(regularFont, &truetype.Options{Size: 20}))
	dc.SetHexColor(labelColor)
	dc.DrawStringAnchored("Unrealized PnL", 300, 420, 0.5, 0.5)

	//  Trade Duration
	dc.SetFontFace(truetype.NewFace(regularFont, &truetype.Options{Size: 16}))
	dc.DrawStringAnchored(fmt.Sprintf("Trade Duration (%s)", data.TradeDuration), 300, 460, 0.5, 0.5)

	// Referral Code
	dc.SetFontFace(truetype.NewFace(boldFont, &truetype.Options{Size: 32}))
	dc.SetHexColor(mainColor)
	dc.DrawStringAnchored(fmt.Sprintf("Referral Code (%s)", data.ReferralCode), 300, 530, 0.5, 0.5)

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
