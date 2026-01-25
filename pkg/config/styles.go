package config

import (
	"os"
	"path/filepath"

	"github.com/gdamore/tcell/v2"
	"gopkg.in/yaml.v3"
)

// Color represents a color that can be specified as hex string or name
type Color string

// ToTcellColor converts a Color to tcell.Color
func (c Color) ToTcellColor() tcell.Color {
	if c == "" {
		return tcell.ColorDefault
	}
	// Handle hex colors
	if len(c) == 7 && c[0] == '#' {
		return tcell.GetColor(string(c))
	}
	// Handle named colors
	return tcell.GetColor(string(c))
}

// StyleConfig represents a complete theme configuration
type StyleConfig struct {
	K13s K13sStyles `yaml:"k13d"`
}

// K13sStyles contains all application-level style definitions
type K13sStyles struct {
	Body      BodyStyle      `yaml:"body"`
	Frame     FrameStyle     `yaml:"frame"`
	Views     ViewStyles     `yaml:"views"`
	Dialog    DialogStyle    `yaml:"dialog"`
	StatusBar StatusBarStyle `yaml:"statusBar"`
}

// BodyStyle defines the main application background
type BodyStyle struct {
	FgColor Color `yaml:"fgColor"`
	BgColor Color `yaml:"bgColor"`
}

// FrameStyle defines border and title styles
type FrameStyle struct {
	BorderColor      Color `yaml:"borderColor"`
	FocusBorderColor Color `yaml:"focusBorderColor"`
	TitleColor       Color `yaml:"titleColor"`
	FocusTitleColor  Color `yaml:"focusTitleColor"`
}

// ViewStyles contains styles for different view types
type ViewStyles struct {
	Table  TableStyle `yaml:"table"`
	Log    LogStyle   `yaml:"log"`
	Charts ChartStyle `yaml:"charts"`
}

// TableStyle defines table/list view colors
type TableStyle struct {
	Header      CellStyle `yaml:"header"`
	RowOdd      CellStyle `yaml:"rowOdd"`
	RowEven     CellStyle `yaml:"rowEven"`
	RowSelected CellStyle `yaml:"rowSelected"`
	RowHover    CellStyle `yaml:"rowHover"`
}

// CellStyle defines a table cell's appearance
type CellStyle struct {
	FgColor Color `yaml:"fgColor"`
	BgColor Color `yaml:"bgColor"`
	Bold    bool  `yaml:"bold"`
}

// LogStyle defines log viewer colors
type LogStyle struct {
	FgColor      Color `yaml:"fgColor"`
	BgColor      Color `yaml:"bgColor"`
	ErrorColor   Color `yaml:"errorColor"`
	WarningColor Color `yaml:"warningColor"`
	InfoColor    Color `yaml:"infoColor"`
}

// ChartStyle defines chart/graph colors
type ChartStyle struct {
	Default Color `yaml:"default"`
	CPU     Color `yaml:"cpu"`
	Memory  Color `yaml:"memory"`
	Network Color `yaml:"network"`
}

// DialogStyle defines modal/dialog colors
type DialogStyle struct {
	FgColor       Color `yaml:"fgColor"`
	BgColor       Color `yaml:"bgColor"`
	ButtonFgColor Color `yaml:"buttonFgColor"`
	ButtonBgColor Color `yaml:"buttonBgColor"`
	ButtonFocusFg Color `yaml:"buttonFocusFgColor"`
	ButtonFocusBg Color `yaml:"buttonFocusBgColor"`
}

// StatusBarStyle defines status bar colors
type StatusBarStyle struct {
	FgColor    Color `yaml:"fgColor"`
	BgColor    Color `yaml:"bgColor"`
	ErrorColor Color `yaml:"errorColor"`
}

// StatusColorConfig defines colors for resource status
type StatusColorConfig struct {
	Running    Color `yaml:"running"`
	Pending    Color `yaml:"pending"`
	Succeeded  Color `yaml:"succeeded"`
	Failed     Color `yaml:"failed"`
	Unknown    Color `yaml:"unknown"`
	Terminated Color `yaml:"terminated"`
}

// DefaultStyles returns the default Dracula-inspired theme
func DefaultStyles() *StyleConfig {
	return &StyleConfig{
		K13s: K13sStyles{
			Body: BodyStyle{
				FgColor: "#f8f8f2",
				BgColor: "#282a36",
			},
			Frame: FrameStyle{
				BorderColor:      "#6272a4",
				FocusBorderColor: "#bd93f9",
				TitleColor:       "#f8f8f2",
				FocusTitleColor:  "#50fa7b",
			},
			Views: ViewStyles{
				Table: TableStyle{
					Header: CellStyle{
						FgColor: "#bd93f9",
						BgColor: "#282a36",
						Bold:    true,
					},
					RowOdd: CellStyle{
						FgColor: "#f8f8f2",
						BgColor: "#282a36",
					},
					RowEven: CellStyle{
						FgColor: "#f8f8f2",
						BgColor: "#343746",
					},
					RowSelected: CellStyle{
						FgColor: "#282a36",
						BgColor: "#8be9fd",
					},
					RowHover: CellStyle{
						FgColor: "#f8f8f2",
						BgColor: "#44475a",
					},
				},
				Log: LogStyle{
					FgColor:      "#f8f8f2",
					BgColor:      "#282a36",
					ErrorColor:   "#ff5555",
					WarningColor: "#ffb86c",
					InfoColor:    "#8be9fd",
				},
				Charts: ChartStyle{
					Default: "#bd93f9",
					CPU:     "#8be9fd",
					Memory:  "#ff79c6",
					Network: "#50fa7b",
				},
			},
			Dialog: DialogStyle{
				FgColor:       "#f8f8f2",
				BgColor:       "#44475a",
				ButtonFgColor: "#f8f8f2",
				ButtonBgColor: "#6272a4",
				ButtonFocusFg: "#282a36",
				ButtonFocusBg: "#50fa7b",
			},
			StatusBar: StatusBarStyle{
				FgColor:    "#f8f8f2",
				BgColor:    "#6272a4",
				ErrorColor: "#ff5555",
			},
		},
	}
}

// LoadStyles loads style configuration from a skin file
func LoadStyles(skinName string) (*StyleConfig, error) {
	if skinName == "" {
		skinName = "default"
	}

	configDir, err := GetConfigDir()
	if err != nil {
		return DefaultStyles(), nil
	}

	skinPath := filepath.Join(configDir, "skins", skinName+".yaml")
	data, err := os.ReadFile(skinPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultStyles(), nil
		}
		return nil, err
	}

	var styles StyleConfig
	if err := yaml.Unmarshal(data, &styles); err != nil {
		return DefaultStyles(), nil
	}

	return &styles, nil
}

// SaveStyles saves style configuration to a skin file
func SaveStyles(skinName string, styles *StyleConfig) error {
	if skinName == "" {
		skinName = "default"
	}

	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	skinDir := filepath.Join(configDir, "skins")
	if err := os.MkdirAll(skinDir, 0755); err != nil {
		return err
	}

	skinPath := filepath.Join(skinDir, skinName+".yaml")
	data, err := yaml.Marshal(styles)
	if err != nil {
		return err
	}

	return os.WriteFile(skinPath, data, 0644)
}

// ListSkins returns available skin names
func ListSkins() ([]string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return []string{"default"}, nil
	}

	skinDir := filepath.Join(configDir, "skins")
	entries, err := os.ReadDir(skinDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{"default"}, nil
		}
		return nil, err
	}

	var skins []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".yaml" {
			name := entry.Name()
			skins = append(skins, name[:len(name)-5])
		}
	}

	if len(skins) == 0 {
		skins = []string{"default"}
	}

	return skins, nil
}
