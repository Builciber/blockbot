package main

import (
	"os"
	"testing"
	"time"
)

func TestGenerateProfitCard(t *testing.T) {
	data := PNLCardData{
		TokenPair:      "CHOGSTARRRRRRRRRRRRRR/MON",
		PercentageGain: "+455555.67%",
		TradeDuration:  formatDuration(259260 * time.Second),
		ReferralCode:   "BBBBBBB",
		IsProfit:       true,
		BackgroundPath: profitBackground[2],
	}

	buf, err := generatePNLCard(data)
	if err != nil {
		t.Fatalf("Failed to generate profit card: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("Generated profit card buffer is empty")
	}

	err = os.WriteFile("test_profit_card.png", buf.Bytes(), 0644)
	if err != nil {
		t.Logf("Warning: Could not save profit card test file: %v", err)
	} else {
		t.Log("✅ Profit card saved to test_profit_card.png")
	}
}

func TestGenerateLossCard(t *testing.T) {
	data := PNLCardData{
		TokenPair:      "HARRYPOTTEROBAMASONIC10INU/MON",
		PercentageGain: "-23.45%",
		TradeDuration:  formatDuration(1*time.Hour + 30*time.Minute),
		ReferralCode:   "BBBBBBB",
		IsProfit:       false,
		BackgroundPath: lossBackgrounds[0],
	}

	buf, err := generatePNLCard(data)
	if err != nil {
		t.Fatalf("Failed to generate loss card: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("Generated loss card buffer is empty")
	}

	err = os.WriteFile("test_loss_card.png", buf.Bytes(), 0644)
	if err != nil {
		t.Logf("Warning: Could not save loss card test file: %v", err)
	} else {
		t.Log("✅ Loss card saved to test_loss_card.png")
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{20*time.Minute + 15*time.Second, "20m 15s"},
		{1*time.Hour + 30*time.Minute + 45*time.Second, "1h 30m 45s"},
		{2*24*time.Hour + 3*time.Hour + 15*time.Minute, "2d 3h 15m"},
		{45 * time.Second, "45s"},
	}

	for _, tt := range tests {
		result := formatDuration(tt.duration)
		if result != tt.expected {
			t.Errorf("formatDuration(%v) = %s; want %s", tt.duration, result, tt.expected)
		}
	}
}
