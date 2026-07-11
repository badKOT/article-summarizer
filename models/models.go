package models

type ModelPricing struct {
	Prompt     string `json:"prompt"`
	Completion string `json:"completion"`
}

type ModelArchitecture struct {
	Modality string `json:"modality"`
}

type SimplifiedModel struct {
	ID            string `json:"id"`
	ContextLength int    `json:"context_length"`
	Modality      string `json:"modality"`
	Pricing       string `json:"pricing"`
}

type FullModel struct {
	ID            string            `json:"id"`
	ContextLength int               `json:"context_length"`
	Architecture  ModelArchitecture `json:"architecture"`
	Pricing       ModelPricing      `json:"pricing"`
}

type ModelsResponse struct {
	Data []FullModel `json:"data"`
}
