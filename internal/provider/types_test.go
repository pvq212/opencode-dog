package provider

import (
	"testing"
)

// --- TriggerMode constants ---

func TestTriggerModeValues(t *testing.T) {
	if ModeAsk != "ask" {
		t.Errorf("ModeAsk = %q, want %q", ModeAsk, "ask")
	}
	if ModePlan != "plan" {
		t.Errorf("ModePlan = %q, want %q", ModePlan, "plan")
	}
	if ModeDo != "do" {
		t.Errorf("ModeDo = %q, want %q", ModeDo, "do")
	}
}

// --- ProviderType constants ---

func TestProviderTypeValues(t *testing.T) {
	if ProviderGitLab != "gitlab" {
		t.Errorf("ProviderGitLab = %q, want %q", ProviderGitLab, "gitlab")
	}
	if ProviderSlack != "slack" {
		t.Errorf("ProviderSlack = %q, want %q", ProviderSlack, "slack")
	}
	if ProviderTelegram != "telegram" {
		t.Errorf("ProviderTelegram = %q, want %q", ProviderTelegram, "telegram")
	}
}

// --- IncomingMessage fields ---

func TestIncomingMessage_Fields(t *testing.T) {
	msg := &IncomingMessage{
		Provider:       ProviderGitLab,
		ProviderCfgID:  "cfg-1",
		ProjectID:      "proj-1",
		ExternalRef:    "https://example.com/issue/1",
		Title:          "Test Issue",
		Body:           "body text",
		Author:         "user1",
		TriggerMode:    ModeAsk,
		TriggerKeyword: "@opencode",
		ReplyMeta:      map[string]int{"issue_iid": 42},
	}

	if msg.Provider != ProviderGitLab {
		t.Errorf("Provider = %q, want %q", msg.Provider, ProviderGitLab)
	}
	if msg.ProviderCfgID != "cfg-1" {
		t.Errorf("ProviderCfgID = %q, want %q", msg.ProviderCfgID, "cfg-1")
	}
	if msg.ProjectID != "proj-1" {
		t.Errorf("ProjectID = %q, want %q", msg.ProjectID, "proj-1")
	}
	if msg.ExternalRef != "https://example.com/issue/1" {
		t.Errorf("ExternalRef = %q", msg.ExternalRef)
	}
	if msg.Title != "Test Issue" {
		t.Errorf("Title = %q", msg.Title)
	}
	if msg.Body != "body text" {
		t.Errorf("Body = %q", msg.Body)
	}
	if msg.Author != "user1" {
		t.Errorf("Author = %q", msg.Author)
	}
	if msg.TriggerMode != ModeAsk {
		t.Errorf("TriggerMode = %q", msg.TriggerMode)
	}
	if msg.TriggerKeyword != "@opencode" {
		t.Errorf("TriggerKeyword = %q", msg.TriggerKeyword)
	}
	if msg.ReplyMeta == nil {
		t.Error("ReplyMeta is nil")
	}
}

// --- TriggerMode type conversion ---

func TestTriggerMode_StringConversion(t *testing.T) {
	var m TriggerMode = "ask"
	if string(m) != "ask" {
		t.Errorf("string(TriggerMode) = %q, want %q", string(m), "ask")
	}
}

// --- ProviderType type conversion ---

func TestProviderType_StringConversion(t *testing.T) {
	var p ProviderType = "gitlab"
	if string(p) != "gitlab" {
		t.Errorf("string(ProviderType) = %q, want %q", string(p), "gitlab")
	}
}
