package main

type PageSize string

const (
	PageSizeA4     PageSize = "a4"
	PageSizeLetter PageSize = "letter"
)

type Theme string

const (
	ThemeLight Theme = "light"
	ThemeDraft Theme = "draft"
)

type Config struct {
	PageSize   PageSize `json:"pageSize"`
	MarginMm   int      `json:"marginMm"`
	BaseFontPt int      `json:"baseFontPt"`
	Theme      Theme    `json:"theme"`
	BaseDir    string   `json:"baseDir"`
}

func DefaultConfig() Config {
	return Config{
		PageSize:   PageSizeA4,
		MarginMm:   25,
		BaseFontPt: 11,
		Theme:      ThemeLight,
	}
}
