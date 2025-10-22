package main

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/fogleman/gg"
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
	img, err := gg.LoadJPG(data.BackgroundPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load background: %w", err)
	}
	dc := gg.NewContextForImage(img)
	tickerFontSize := 48.0
	labelFontSize := 20.0
	referralCodeFontSize := 32.0
	percentageFontSize := 72.0
	tickerFont, err := gg.LoadFontFace("./fonts/TacticSansExtExd-Blk.ttf", tickerFontSize)
	if err != nil {
		return nil, fmt.Errorf("failed to load ticker font: %w", err)
	}
	labelFont, err := gg.LoadFontFace("./fonts/TTFirsNeue-Regular.ttf", labelFontSize)
	if err != nil {
		return nil, fmt.Errorf("failed to load text font: %w", err)
	}
	percentageFont, err := gg.LoadFontFace("./fonts/TacticSansExtExd-Blk.ttf", percentageFontSize)
	if err != nil {
		return nil, fmt.Errorf("failed to load percentage font: %w", err)
	}
	referralCodeFont, err := gg.LoadFontFace("./fonts/TTFirsNeue-DemiBold.ttf", referralCodeFontSize)
	if err != nil {
		return nil, fmt.Errorf("failed to load referral code font: %w", err)
	}
	mainColor := profitColor
	if !data.IsProfit {
		mainColor = lossColor
	}

	tokenPair := strings.ToUpper(data.TokenPair)
	xLeft := 30.0
	y := float64(dc.Height()) / 3.0
	imgMidPointX := float64(dc.Width()) / 2.0
	dc.SetFontFace(tickerFont)
	dc.SetHexColor(textWhite)
	tokenPairFontWidth, fontHeight := dc.MeasureString(tokenPair)
	for tokenPairFontWidth+xLeft > imgMidPointX {
		tickerFontSize -= 3
		dc.LoadFontFace("./fonts/TacticSansExtExd-Blk.ttf", tickerFontSize)
		tokenPairFontWidth, fontHeight = dc.MeasureString(tokenPair)
	}
	dc.DrawString(tokenPair, xLeft, y)

	pnlLabel := "Unrealized PnL"
	y += fontHeight + 40
	dc.SetFontFace(labelFont)
	dc.SetHexColor(labelColor)
	_, fontHeight = dc.MeasureString(pnlLabel)
	dc.DrawStringAnchored(pnlLabel, xLeft, y, 0, 0.5)

	y += fontHeight + 30
	dc.SetFontFace(percentageFont)
	dc.SetHexColor(mainColor)
	percentageFontWidth, fontHeight := dc.MeasureString(data.PercentageGain)
	for percentageFontWidth+xLeft > imgMidPointX {
		percentageFontSize -= 5
		dc.LoadFontFace("./fonts/TacticSansExtExd-Blk.ttf", percentageFontSize)
		percentageFontWidth, fontHeight = dc.MeasureString(data.PercentageGain)
	}
	dc.DrawStringAnchored(data.PercentageGain, xLeft, y, 0.0, 0.5)

	duration := fmt.Sprintf("Duration: %s", data.TradeDuration)
	y += fontHeight + 20
	dc.SetFontFace(labelFont)
	dc.SetHexColor(labelColor)
	dc.DrawString(duration, xLeft, y)

	referralLabelY := 4.5 * (float64(dc.Height()) / 6.0)
	referralLabel := "Referral Link"
	dc.SetFontFace(labelFont)
	dc.SetHexColor(labelColor)
	_, fontHeight = dc.MeasureString(referralLabel)
	dc.DrawStringAnchored(referralLabel, xLeft, referralLabelY, 0, 0.5)

	referralCodeY := referralLabelY + fontHeight + 20
	dc.SetFontFace(referralCodeFont)
	dc.SetHexColor(mainColor)
	referralCodeFontWidth, _ := dc.MeasureString(data.ReferralCode)
	for referralCodeFontWidth+xLeft > imgMidPointX {
		referralCodeFontSize -= 3
		dc.LoadFontFace("./fonts/TTFirsNeue-DemiBold.ttf", referralCodeFontSize)
		referralCodeFontWidth, _ = dc.MeasureString(data.ReferralCode)
	}
	dc.DrawStringAnchored(data.ReferralCode, xLeft, referralCodeY, 0, 0.5)

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
