package config

import (
	"fmt"
	"unicode/utf8"
)

type ModelConfig struct {
	Status   StatusConfig `yaml:"status"`
	Keywords []string     `yaml:"keywords"`
}

type StatusConfig struct {
	ContentListLimit int `yaml:"contentListLimit" json:"contentListLimit"`
	ContentLimit     int `yaml:"contentLimit" json:"contentLimit"`
	OverviewLimit    int `yaml:"overviewLimit" json:"overviewLimit"`
}

func (c *StatusConfig) RestrictContent(content string) error {
	count := utf8.RuneCountInString(content)
	if count > c.ContentLimit {
		return fmt.Errorf("maximum %d unicode characters per paragraph, %d",
			c.ContentLimit, count)
	}
	return nil
}

func (c *StatusConfig) RestrictContentList(contentListSize int) error {
	if contentListSize > Conf.Model.Status.ContentListLimit {
		return fmt.Errorf("maximum %d content blocks, %d",
			c.ContentListLimit, contentListSize)
	}
	return nil
}

func (c *StatusConfig) RestrictOverview(content string) error {
	count := utf8.RuneCountInString(content)
	if count > c.OverviewLimit {
		return fmt.Errorf("maximum %d unicode characters in status overview, %d",
			c.OverviewLimit, count)
	}
	return nil
}

func initModel() {
	if Conf.Model.Status.OverviewLimit == 0 {
		Conf.Model.Status.OverviewLimit = 256
	}

	if Conf.Model.Status.ContentLimit == 0 {
		Conf.Model.Status.ContentLimit = 4096
	}

	if Conf.Model.Status.ContentListLimit == 0 {
		Conf.Model.Status.ContentListLimit = 20
	}

	Conf.Model.Keywords =
		append(Conf.Model.Keywords,
			"explore",
			"messages",
			"bookmarks",
			"settings",
			"status",
			"search",
			"labels",
			"tags",
			"news",
			"probe",
			"verified",
		)
}
