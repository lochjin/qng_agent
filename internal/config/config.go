package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	LLM      LLMConfig      `mapstructure:"llm"`
	MCP      MCPConfig      `mapstructure:"mcp"`
	QNG      QNGConfig      `mapstructure:"qng"`
	MetaMask MetaMaskConfig `mapstructure:"metamask"`
	UI       UIConfig       `mapstructure:"ui"`
}

type LLMConfig struct {
	Provider string            `mapstructure:"provider"`
	Configs  map[string]string `mapstructure:"configs"`
}

type MCPConfig struct {
	Timeout int `mapstructure:"timeout"`
}

type QNGConfig struct {
	ChainRPC     string `mapstructure:"chain_rpc"`
	GraphNodes   int    `mapstructure:"graph_nodes"`
	PollInterval int    `mapstructure:"poll_interval"`
}

type MetaMaskConfig struct {
	Network string `mapstructure:"network"`
}

type UIConfig struct {
	Port   int    `mapstructure:"port"`
	Static string `mapstructure:"static"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./configs")

	// 设置默认值
	viper.SetDefault("llm.provider", "openai")
	viper.SetDefault("llm.configs.openai_base_url", "https://api.openai.com/v1")
	viper.SetDefault("llm.configs.ollama_base_url", "http://localhost:11434")
	viper.SetDefault("llm.configs.gemini_base_url", "https://generativelanguage.googleapis.com/v1beta")
	viper.SetDefault("llm.configs.model", "gpt-4")
	viper.SetDefault("mcp.timeout", 30)
	viper.SetDefault("qng.poll_interval", 1000)
	viper.SetDefault("qng.graph_nodes", 5)
	viper.SetDefault("ui.port", 9090)
	viper.SetDefault("ui.static", "./web/dist")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
