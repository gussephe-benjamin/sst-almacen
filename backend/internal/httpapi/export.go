package httpapi

import (
	"fmt"
	"net/http"

	"github.com/go-pdf/fpdf"
	"github.com/xuri/excelize/v2"
)

var exportCols = []struct{ title, key string }{
	{"Código", "codigo"}, {"Fecha", "created_at"}, {"Descripción", "descripcion"}, {"Tipo", "tipo"}, {"Ubicación", "ubicacion"},
	{"Área responsable", "area_responsable"}, {"Riesgo", "nivel_riesgo"}, {"Prioridad", "prioridad"}, {"Estado", "estado"},
}

func cell(row map[string]any, key string) string {
	v := fmt.Sprint(row[key])
	if key == "created_at" && len(v) >= 10 {
		return v[:10]
	}
	if v == "<nil>" {
		return ""
	}
	return v
}

func (a *API) exportXLSX(w http.ResponseWriter, r *http.Request) {
	rows, err := a.S.HallazgosRows(r.Context(), filtro(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	f := excelize.NewFile()
	sh := "Hallazgos"
	f.SetSheetName("Sheet1", sh)
	for i, c := range exportCols {
		name, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sh, name, c.title)
	}
	for j, row := range rows {
		for i, c := range exportCols {
			name, _ := excelize.CoordinatesToCellName(i+1, j+2)
			f.SetCellValue(sh, name, cell(row, c.key))
		}
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="hallazgos.xlsx"`)
	f.Write(w)
}

func (a *API) exportPDF(w http.ResponseWriter, r *http.Request) {
	rows, err := a.S.HallazgosRows(r.Context(), filtro(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	pdf := fpdf.New("L", "mm", "A4", "")
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 14)
	pdf.Cell(0, 10, tr("Reporte de hallazgos"))
	pdf.Ln(12)
	widths := []float64{18, 22, 70, 40, 30, 32, 18, 20, 25}
	pdf.SetFont("Helvetica", "B", 8)
	for i, c := range exportCols {
		pdf.CellFormat(widths[i], 7, tr(c.title), "1", 0, "", false, 0, "")
	}
	pdf.Ln(-1)
	pdf.SetFont("Helvetica", "", 8)
	for _, row := range rows {
		for i, c := range exportCols {
			v := cell(row, c.key)
			if len([]rune(v)) > 48 {
				v = string([]rune(v)[:46]) + "…"
			}
			pdf.CellFormat(widths[i], 6, tr(v), "1", 0, "", false, 0, "")
		}
		pdf.Ln(-1)
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="hallazgos.pdf"`)
	pdf.Output(w)
}
