import * as XLSX from 'xlsx'
import { t } from '../i18n'
// Legacy .xls codepage support (e.g. GBK)
import * as cptable from 'xlsx/dist/cpexcel.full.mjs'
import type { ExcelSheet } from './filePreview'

XLSX.set_cptable(cptable)

function formatCell(cell: XLSX.CellObject | undefined): string {
  if (!cell) return ''
  if (cell.w != null && cell.w !== '') return String(cell.w)
  const v = cell.v
  if (v == null) return ''
  if (v instanceof Date) return v.toLocaleString()
  return String(v)
}

function normalizeRows(rows: string[][]): string[][] {
  if (!rows.length) return [['']]
  const maxCols = Math.max(...rows.map((row) => row.length), 1)
  return rows.map((row) => {
    const next = row.map((cell) => (cell == null ? '' : String(cell)))
    while (next.length < maxCols) next.push('')
    return next
  })
}

function worksheetToData(sheet: XLSX.WorkSheet): string[][] {
  if (sheet['!ref']) {
    const range = XLSX.utils.decode_range(sheet['!ref'])
    const data: string[][] = []
    for (let r = range.s.r; r <= range.e.r; r++) {
      const row: string[] = []
      for (let c = range.s.c; c <= range.e.c; c++) {
        const addr = XLSX.utils.encode_cell({ r, c })
        row.push(formatCell(sheet[addr]))
      }
      data.push(row)
    }
    if (data.length) return normalizeRows(data)
  }

  const rows = XLSX.utils.sheet_to_json<(string | number | boolean | Date | null)[]>(sheet, {
    header: 1,
    defval: '',
    raw: false,
    blankrows: true,
  })
  const data = rows.map((row) =>
    (Array.isArray(row) ? row : []).map((cell) => {
      if (cell == null) return ''
      if (cell instanceof Date) return cell.toLocaleString()
      return String(cell)
    }),
  )
  return normalizeRows(data)
}

export function parseExcel(buffer: ArrayBuffer): ExcelSheet[] {
  const workbook = XLSX.read(buffer, {
    type: 'array',
    cellDates: true,
    cellNF: true,
    cellText: true,
  })
  if (!workbook.SheetNames.length) {
    throw new Error(t('excel.noSheets'))
  }
  return workbook.SheetNames.map((name) => ({
    name,
    data: worksheetToData(workbook.Sheets[name]),
  }))
}
