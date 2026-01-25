package resources

import "github.com/gdamore/tcell/v2"

type TableCell struct {
	Text  string
	Color tcell.Color
}

type ResourceView struct {
	Headers []string
	Rows    [][]TableCell
}
