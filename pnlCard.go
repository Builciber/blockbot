package main

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"strings"
	"time"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
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
	cardWidth  = 1351
	cardHeight = 901

	profitColor = "#3CFE92"
	lossColor   = "#F54542"
	labelColor  = "#9F9F9F"
	textWhite   = "#FFFFFF"
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

	tickerFontBytes, err := os.ReadFile("./fonts/TacticSansExtExd-Blk.ttf")
	if err != nil {
		return nil, fmt.Errorf("failed to read ticker font: %w", err)
	}
	tickerFont, err := truetype.Parse(tickerFontBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ticker font: %w", err)
	}

	textFontBytes, err := os.ReadFile("./fonts/TTFirsNeue-DemiBold.ttf")
	if err != nil {
		return nil, fmt.Errorf("failed to read text font: %w", err)
	}
	textFont, err := truetype.Parse(textFontBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse text font: %w", err)
	}

	mainColor := profitColor
	if !data.IsProfit {
		mainColor = lossColor
	}

	xLeft := 50.0

	maxWidth := 320.0
	tokenPair := strings.ToUpper(data.TokenPair)

	fontSize := 54.0
	if len(data.TokenPair) > 10 {
		dc.SetFontFace(truetype.NewFace(tickerFont, &truetype.Options{Size: 46}))
	}

	dc.SetFontFace(truetype.NewFace(tickerFont, &truetype.Options{Size: fontSize}))
	dc.SetHexColor(textWhite)

	if len(data.TokenPair) > 10 {
		dc.SetFontFace(truetype.NewFace(tickerFont, &truetype.Options{Size: 46}))
	}

	y := 240.0
	lineHeight := 1.2
	dc.DrawStringWrapped(tokenPair, xLeft, y, 0, 0.5, maxWidth, lineHeight, gg.AlignLeft)
	dc.DrawStringAnchored(data.TokenPair, xLeft, y, 0, 0.5)

	y += 80
	dc.SetFontFace(truetype.NewFace(textFont, &truetype.Options{Size: 22}))
	dc.SetHexColor(labelColor)
	dc.DrawStringAnchored("Unrealized PnL", xLeft, y, 0, 0.5)

	y += 60
	dc.SetFontFace(truetype.NewFace(tickerFont, &truetype.Options{Size: 68}))
	dc.SetHexColor(mainColor)
	dc.DrawStringAnchored(data.PercentageGain, xLeft, y, 0, 0.5)

	y += 50
	dc.SetFontFace(truetype.NewFace(textFont, &truetype.Options{Size: 18}))
	dc.SetHexColor(labelColor)
	dc.DrawStringAnchored(fmt.Sprintf("duration %s", data.TradeDuration), xLeft, y, 0, 0.5)

	referralLabelY := 640.0
	codeY := referralLabelY + 28

	dc.SetFontFace(truetype.NewFace(textFont, &truetype.Options{Size: 16}))
	dc.SetHexColor(labelColor)
	dc.DrawStringAnchored("Referral Code", xLeft, referralLabelY, 0, 0.5)

	dc.SetFontFace(truetype.NewFace(tickerFont, &truetype.Options{Size: 28}))
	dc.SetHexColor(mainColor)
	dc.DrawStringAnchored(data.ReferralCode, xLeft, codeY, 0, 0.5)

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
