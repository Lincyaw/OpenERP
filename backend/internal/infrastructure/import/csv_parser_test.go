package csvimport

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCSVParser(t *testing.T) {
	t.Run("Valid UTF-8 CSV", func(t *testing.T) {
		csv := "name,age,city\nAlice,30,New York\nBob,25,Boston"
		parser, err := NewCSVParser(strings.NewReader(csv))

		require.NoError(t, err)
		require.NotNil(t, parser)
	})

	t.Run("UTF-8 BOM is stripped", func(t *testing.T) {
		// UTF-8 BOM: 0xEF, 0xBB, 0xBF
		csv := "\xEF\xBB\xBFname,age\nAlice,30"
		parser, err := NewCSVParser(strings.NewReader(csv))

		require.NoError(t, err)
		require.NotNil(t, parser)

		err = parser.ParseHeader()
		require.NoError(t, err)

		// Header should not include BOM
		headers := parser.Headers()
		assert.Equal(t, "name", headers[0])
	})

	t.Run("Empty file returns error", func(t *testing.T) {
		parser, err := NewCSVParser(strings.NewReader(""))

		assert.Error(t, err)
		assert.Nil(t, parser)
		assert.ErrorIs(t, err, ErrEmptyFile)
	})

	t.Run("Custom delimiter", func(t *testing.T) {
		csv := "name;age;city\nAlice;30;NYC"
		parser, err := NewCSVParser(strings.NewReader(csv), WithDelimiter(';'))

		require.NoError(t, err)
		require.NoError(t, parser.ParseHeader())

		headers := parser.Headers()
		assert.Equal(t, []string{"name", "age", "city"}, headers)
	})
}

func TestParseHeader(t *testing.T) {
	t.Run("Valid header", func(t *testing.T) {
		csv := "code,name,price\n001,Widget,10.00"
		parser, _ := NewCSVParser(strings.NewReader(csv))

		err := parser.ParseHeader()

		require.NoError(t, err)
		assert.Equal(t, []string{"code", "name", "price"}, parser.Headers())
		assert.Equal(t, map[string]int{"code": 0, "name": 1, "price": 2}, parser.HeaderMap())
	})

	t.Run("Header with spaces trimmed", func(t *testing.T) {
		csv := "  code  ,  name  ,  price  \n001,Widget,10.00"
		parser, _ := NewCSVParser(strings.NewReader(csv))

		err := parser.ParseHeader()

		require.NoError(t, err)
		assert.Equal(t, []string{"code", "name", "price"}, parser.Headers())
	})

	t.Run("HasHeader check", func(t *testing.T) {
		csv := "code,name,price\n001,Widget,10.00"
		parser, _ := NewCSVParser(strings.NewReader(csv))
		parser.ParseHeader()

		assert.True(t, parser.HasHeader("code"))
		assert.True(t, parser.HasHeader("name"))
		assert.False(t, parser.HasHeader("description"))
	})

	t.Run("ValidateHeaders finds missing", func(t *testing.T) {
		csv := "code,name\n001,Widget"
		parser, _ := NewCSVParser(strings.NewReader(csv))
		parser.ParseHeader()

		missing := parser.ValidateHeaders([]string{"code", "name", "price", "category"})
		assert.ElementsMatch(t, []string{"price", "category"}, missing)
	})
}

func TestReadRow(t *testing.T) {
	t.Run("Read single row", func(t *testing.T) {
		csv := "code,name,price\n001,Widget,10.00"
		parser, _ := NewCSVParser(strings.NewReader(csv))
		parser.ParseHeader()

		row, err := parser.ReadRow()

		require.NoError(t, err)
		assert.Equal(t, 2, row.LineNumber)
		assert.Equal(t, "001", row.Get("code"))
		assert.Equal(t, "Widget", row.Get("name"))
		assert.Equal(t, "10.00", row.Get("price"))
	})

	t.Run("Row with missing columns", func(t *testing.T) {
		csv := "code,name,price,category\n001,Widget"
		parser, _ := NewCSVParser(strings.NewReader(csv))
		parser.ParseHeader()

		row, err := parser.ReadRow()

		require.NoError(t, err)
		assert.Equal(t, "001", row.Get("code"))
		assert.Equal(t, "Widget", row.Get("name"))
		assert.Equal(t, "", row.Get("price"))
		assert.Equal(t, "", row.Get("category"))
	})

	t.Run("GetOrDefault", func(t *testing.T) {
		csv := "code,name,price\n001,Widget,"
		parser, _ := NewCSVParser(strings.NewReader(csv))
		parser.ParseHeader()

		row, _ := parser.ReadRow()

		assert.Equal(t, "001", row.GetOrDefault("code", "default"))
		assert.Equal(t, "N/A", row.GetOrDefault("price", "N/A"))
		assert.Equal(t, "none", row.GetOrDefault("missing", "none"))
	})

	t.Run("IsEmpty row", func(t *testing.T) {
		csv := "code,name\n,,\n001,Widget"
		parser, _ := NewCSVParser(strings.NewReader(csv))
		parser.ParseHeader()

		row1, _ := parser.ReadRow()
		assert.True(t, row1.IsEmpty())

		row2, _ := parser.ReadRow()
		assert.False(t, row2.IsEmpty())
	})

	t.Run("EOF after last row", func(t *testing.T) {
		csv := "code,name\n001,Widget"
		parser, _ := NewCSVParser(strings.NewReader(csv))
		parser.ParseHeader()

		_, err := parser.ReadRow()
		require.NoError(t, err)

		_, err = parser.ReadRow()
		assert.Equal(t, io.EOF, err)
	})
}

func TestReadAllRows(t *testing.T) {
	t.Run("Read all rows", func(t *testing.T) {
		csv := "code,name\n001,Widget\n002,Gadget\n003,Gizmo"
		parser, _ := NewCSVParser(strings.NewReader(csv))
		parser.ParseHeader()

		rows, err := parser.ReadAllRows()

		require.NoError(t, err)
		assert.Len(t, rows, 3)
		assert.Equal(t, "001", rows[0].Get("code"))
		assert.Equal(t, "002", rows[1].Get("code"))
		assert.Equal(t, "003", rows[2].Get("code"))
	})

	t.Run("Skip empty rows", func(t *testing.T) {
		csv := "code,name\n001,Widget\n,,\n,,\n002,Gadget"
		parser, _ := NewCSVParser(strings.NewReader(csv))
		parser.ParseHeader()

		rows, err := parser.ReadAllRows()

		require.NoError(t, err)
		assert.Len(t, rows, 2)
	})

	t.Run("TotalRows count", func(t *testing.T) {
		csv := "code,name\n001,Widget\n002,Gadget\n003,Gizmo"
		parser, _ := NewCSVParser(strings.NewReader(csv))
		parser.ParseHeader()

		parser.ReadAllRows()

		assert.Equal(t, 3, parser.TotalRows())
	})
}

func TestParseFromBytes(t *testing.T) {
	t.Run("Parse from byte slice", func(t *testing.T) {
		data := []byte("code,name\n001,Widget")
		parser, err := ParseFromBytes(data)

		require.NoError(t, err)
		require.NoError(t, parser.ParseHeader())

		row, _ := parser.ReadRow()
		assert.Equal(t, "001", row.Get("code"))
	})
}

func TestQuotedFields(t *testing.T) {
	t.Run("Fields with quotes", func(t *testing.T) {
		csv := `code,name,description
001,"Widget","A fancy widget"
002,"Gadget","Contains, comma"
003,"Item ""Quoted""","With ""quotes"""
`
		parser, _ := NewCSVParser(strings.NewReader(csv))
		parser.ParseHeader()

		row1, _ := parser.ReadRow()
		assert.Equal(t, "Widget", row1.Get("name"))
		assert.Equal(t, "A fancy widget", row1.Get("description"))

		row2, _ := parser.ReadRow()
		assert.Equal(t, "Contains, comma", row2.Get("description"))

		row3, _ := parser.ReadRow()
		assert.Equal(t, `Item "Quoted"`, row3.Get("name"))
		assert.Equal(t, `With "quotes"`, row3.Get("description"))
	})
}

func TestMultilineFields(t *testing.T) {
	t.Run("Fields with newlines", func(t *testing.T) {
		csv := "code,name,description\n001,Widget,\"Line 1\nLine 2\nLine 3\""
		parser, _ := NewCSVParser(strings.NewReader(csv))
		parser.ParseHeader()

		row, _ := parser.ReadRow()
		assert.Equal(t, "Line 1\nLine 2\nLine 3", row.Get("description"))
	})
}

func TestGetColumnIndex(t *testing.T) {
	csv := "code,name,price\n001,Widget,10.00"
	parser, _ := NewCSVParser(strings.NewReader(csv))
	parser.ParseHeader()

	idx, ok := parser.GetColumnIndex("name")
	assert.True(t, ok)
	assert.Equal(t, 1, idx)

	_, ok = parser.GetColumnIndex("missing")
	assert.False(t, ok)
}
