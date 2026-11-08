package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Cakem1x/fin_man/internal/workspace"
)

func writeConfig(t *testing.T, content string) *workspace.Manager {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "fin.toml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return workspace.NewManager(dir)
}

func TestLoadConfigWithoutLLMSectionUsesDefaults(t *testing.T) {
	cfg, err := writeConfig(t, "store_path = \"../cipher\"\n").LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	want := workspace.Config{StorePath: "../cipher", LLM: workspace.DefaultLLMConfig()}
	if *cfg != want {
		t.Errorf("cfg = %+v, want %+v", *cfg, want)
	}
	if cfg.LLM.Enabled {
		t.Error("LLM must be disabled by default")
	}
	if cfg.LLM.ChatModel != "qwen3:1.7b" || cfg.LLM.EmbedModel != "qwen3-embedding:0.6b" {
		t.Errorf("unexpected default models: %+v", cfg.LLM)
	}
}

func TestLoadConfigPartialLLMSectionKeepsOtherDefaults(t *testing.T) {
	cfg, err := writeConfig(t, "store_path = \"x\"\n[llm]\nenabled = true\nchat_model = \"qwen3:4b\"\n").LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	want := workspace.DefaultLLMConfig()
	want.Enabled = true
	want.ChatModel = "qwen3:4b"
	if cfg.LLM != want {
		t.Errorf("LLM = %+v, want %+v", cfg.LLM, want)
	}
}
