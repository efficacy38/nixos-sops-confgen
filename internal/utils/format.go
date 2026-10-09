package utils

import (
	"os"

	"github.com/olekukonko/tablewriter"
)

func RenderTable(header []string, data [][]string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header(header)
	table.Bulk(data)
	table.Render()
}
