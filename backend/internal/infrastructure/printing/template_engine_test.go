package printing

import (
	"context"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/printing"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTemplateEngine(t *testing.T) {
	engine := NewTemplateEngine()
	assert.NotNil(t, engine)
	assert.NotNil(t, engine.funcMap)
}

func TestTemplateEngine_GetFuncMap(t *testing.T) {
	engine := NewTemplateEngine()
	funcMap := engine.GetFuncMap()

	// Check essential functions exist
	assert.NotNil(t, funcMap["formatMoney"])
	assert.NotNil(t, funcMap["formatDate"])
	assert.NotNil(t, funcMap["moneyToChinese"])
	assert.NotNil(t, funcMap["formatDecimal"])
	assert.NotNil(t, funcMap["add"])
	assert.NotNil(t, funcMap["sub"])
	assert.NotNil(t, funcMap["mul"])
	assert.NotNil(t, funcMap["div"])
}

func TestTemplateEngine_Render_Simple(t *testing.T) {
	engine := NewTemplateEngine()
	ctx := context.Background()

	template := &printing.PrintTemplate{}
	template.ID = uuid.New()
	template.Content = `<html><body>Hello, {{.Name}}!</body></html>`

	data := map[string]interface{}{
		"Name": "World",
	}

	result, err := engine.Render(ctx, &RenderTemplateRequest{
		Template: template,
		Data:     data,
	})

	require.NoError(t, err)
	assert.Contains(t, result.HTML, "Hello, World!")
	assert.True(t, result.RenderDuration > 0)
}

func TestTemplateEngine_Render_NilRequest(t *testing.T) {
	engine := NewTemplateEngine()
	ctx := context.Background()

	_, err := engine.Render(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "render request is nil")
}

func TestTemplateEngine_Render_NilTemplate(t *testing.T) {
	engine := NewTemplateEngine()
	ctx := context.Background()

	_, err := engine.Render(ctx, &RenderTemplateRequest{
		Template: nil,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template is nil")
}

func TestTemplateEngine_Render_EmptyContent(t *testing.T) {
	engine := NewTemplateEngine()
	ctx := context.Background()

	template := &printing.PrintTemplate{}
	template.ID = uuid.New()
	template.Content = ""

	_, err := engine.Render(ctx, &RenderTemplateRequest{
		Template: template,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template content is empty")
}

func TestTemplateEngine_Render_InvalidTemplate(t *testing.T) {
	engine := NewTemplateEngine()
	ctx := context.Background()

	template := &printing.PrintTemplate{}
	template.ID = uuid.New()
	template.Content = `{{.Name` // Missing closing braces

	_, err := engine.Render(ctx, &RenderTemplateRequest{
		Template: template,
		Data:     map[string]interface{}{},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse template")
}

func TestTemplateEngine_Render_WithLoop(t *testing.T) {
	engine := NewTemplateEngine()
	ctx := context.Background()

	template := &printing.PrintTemplate{}
	template.ID = uuid.New()
	template.Content = `<ul>{{range .Items}}<li>{{.}}</li>{{end}}</ul>`

	data := map[string]interface{}{
		"Items": []string{"Apple", "Banana", "Cherry"},
	}

	result, err := engine.Render(ctx, &RenderTemplateRequest{
		Template: template,
		Data:     data,
	})

	require.NoError(t, err)
	assert.Contains(t, result.HTML, "<li>Apple</li>")
	assert.Contains(t, result.HTML, "<li>Banana</li>")
	assert.Contains(t, result.HTML, "<li>Cherry</li>")
}

func TestTemplateEngine_Render_WithConditional(t *testing.T) {
	engine := NewTemplateEngine()
	ctx := context.Background()

	template := &printing.PrintTemplate{}
	template.ID = uuid.New()
	template.Content = `{{if .ShowPrice}}Price: {{.Price}}{{else}}Price hidden{{end}}`

	tests := []struct {
		name      string
		showPrice bool
		expected  string
	}{
		{"show price", true, "Price: 100"},
		{"hide price", false, "Price hidden"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := map[string]interface{}{
				"ShowPrice": tt.showPrice,
				"Price":     100,
			}

			result, err := engine.Render(ctx, &RenderTemplateRequest{
				Template: template,
				Data:     data,
			})

			require.NoError(t, err)
			assert.Contains(t, result.HTML, tt.expected)
		})
	}
}

func TestTemplateEngine_Render_WithCustomFunctions(t *testing.T) {
	engine := NewTemplateEngine()
	ctx := context.Background()

	template := &printing.PrintTemplate{}
	template.ID = uuid.New()
	template.Content = `Total: {{formatMoney .Amount}}`

	data := map[string]interface{}{
		"Amount": decimal.NewFromFloat(1234.56),
	}

	result, err := engine.Render(ctx, &RenderTemplateRequest{
		Template: template,
		Data:     data,
	})

	require.NoError(t, err)
	assert.Contains(t, result.HTML, "¥1,234.56")
}

func TestTemplateEngine_RenderString(t *testing.T) {
	engine := NewTemplateEngine()
	ctx := context.Background()

	content := `Hello, {{.Name}}!`
	data := map[string]interface{}{
		"Name": "Test",
	}

	result, err := engine.RenderString(ctx, "test", content, data)
	require.NoError(t, err)
	assert.Equal(t, "Hello, Test!", result)
}

func TestTemplateEngine_RenderString_Empty(t *testing.T) {
	engine := NewTemplateEngine()
	ctx := context.Background()

	_, err := engine.RenderString(ctx, "test", "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template content is empty")
}

// =============================================================================
// Template Function Tests - Money Formatting
// =============================================================================

func TestFormatMoney(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{decimal.NewFromFloat(1234.56), "¥1,234.56"},
		{decimal.NewFromFloat(0), "¥0.00"},
		{decimal.NewFromFloat(-1234.56), "¥-1,234.56"},
		{decimal.NewFromFloat(1000000), "¥1,000,000.00"},
		{1234.56, "¥1,234.56"},
		{1234, "¥1,234.00"},
		{"1234.56", "¥1,234.56"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatMoney(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatMoneyRaw(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{decimal.NewFromFloat(1234.56), "1,234.56"},
		{decimal.NewFromFloat(0), "0.00"},
		{decimal.NewFromFloat(-1234.56), "-1,234.56"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatMoneyRaw(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMoneyToChinese(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{decimal.NewFromFloat(0), "零元整"},
		{decimal.NewFromFloat(1), "壹元整"},
		{decimal.NewFromFloat(10), "壹拾元整"},
		{decimal.NewFromFloat(11), "壹拾壹元整"},
		{decimal.NewFromFloat(100), "壹佰元整"},
		{decimal.NewFromFloat(101), "壹佰零壹元整"},
		{decimal.NewFromFloat(110), "壹佰壹拾元整"},
		{decimal.NewFromFloat(1000), "壹仟元整"},
		{decimal.NewFromFloat(10000), "壹万元整"},
		{decimal.NewFromFloat(1234.56), "壹仟贰佰叁拾肆元伍角陆分"},
		{decimal.NewFromFloat(100000000), "壹亿元整"},
		{decimal.NewFromFloat(0.01), "壹分"},
		{decimal.NewFromFloat(0.10), "壹角"},
		{decimal.NewFromFloat(0.11), "壹角壹分"},
		{decimal.NewFromFloat(-100), "负壹佰元整"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := moneyToChinese(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Template Function Tests - Date Formatting
// =============================================================================

func TestFormatDate(t *testing.T) {
	testTime := time.Date(2024, 1, 15, 14, 30, 45, 0, time.UTC)

	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"time.Time", testTime, "2024-01-15"},
		{"*time.Time", &testTime, "2024-01-15"},
		{"zero time", time.Time{}, ""},
		{"nil *time.Time", (*time.Time)(nil), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDate(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatDateTime(t *testing.T) {
	testTime := time.Date(2024, 1, 15, 14, 30, 45, 0, time.UTC)
	result := formatDateTime(testTime)
	assert.Equal(t, "2024-01-15 14:30:45", result)
}

func TestFormatTime(t *testing.T) {
	testTime := time.Date(2024, 1, 15, 14, 30, 45, 0, time.UTC)
	result := formatTime(testTime)
	assert.Equal(t, "14:30:45", result)
}

// =============================================================================
// Template Function Tests - Number Formatting
// =============================================================================

func TestFormatDecimal(t *testing.T) {
	tests := []struct {
		value     interface{}
		precision int
		expected  string
	}{
		{decimal.NewFromFloat(1234.5678), 2, "1234.57"},
		{decimal.NewFromFloat(1234.5678), 4, "1234.5678"},
		{decimal.NewFromFloat(1234.5678), 0, "1235"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatDecimal(tt.value, tt.precision)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatPercent(t *testing.T) {
	tests := []struct {
		value     interface{}
		precision int
		expected  string
	}{
		{decimal.NewFromFloat(0.15), 0, "15%"},
		{decimal.NewFromFloat(0.155), 1, "15.5%"},
		{decimal.NewFromFloat(1.5), 0, "150%"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatPercent(tt.value, tt.precision)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Template Function Tests - String Utilities
// =============================================================================

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		max      int
		suffix   []string
		expected string
	}{
		{"Hello World", 20, nil, "Hello World"},
		{"Hello World", 8, nil, "Hello..."},
		{"Hello World", 8, []string{"~"}, "Hello W~"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := truncate(tt.input, tt.max, tt.suffix...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPadLeft(t *testing.T) {
	tests := []struct {
		input    string
		length   int
		pad      string
		expected string
	}{
		{"123", 5, "0", "00123"},
		{"123", 3, "0", "123"},
		{"123", 6, "0", "000123"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := padLeft(tt.input, tt.length, tt.pad)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		input    string
		length   int
		pad      string
		expected string
	}{
		{"123", 5, "0", "12300"},
		{"123", 3, "0", "123"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := padRight(tt.input, tt.length, tt.pad)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTitleCase(t *testing.T) {
	result := titleCase("hello world")
	assert.Equal(t, "Hello World", result)
}

// =============================================================================
// Template Function Tests - Comparison
// =============================================================================

func TestComparisonFunctions(t *testing.T) {
	a := decimal.NewFromInt(10)
	b := decimal.NewFromInt(20)
	c := decimal.NewFromInt(10)

	assert.True(t, ltFunc(a, b))
	assert.False(t, ltFunc(b, a))
	assert.False(t, ltFunc(a, c))

	assert.True(t, leFunc(a, b))
	assert.True(t, leFunc(a, c))
	assert.False(t, leFunc(b, a))

	assert.True(t, gtFunc(b, a))
	assert.False(t, gtFunc(a, b))
	assert.False(t, gtFunc(a, c))

	assert.True(t, geFunc(b, a))
	assert.True(t, geFunc(a, c))
	assert.False(t, geFunc(a, b))
}

// =============================================================================
// Template Function Tests - Arithmetic
// =============================================================================

func TestArithmeticFunctions(t *testing.T) {
	a := decimal.NewFromInt(10)
	b := decimal.NewFromInt(3)

	assert.True(t, add(a, b).Equal(decimal.NewFromInt(13)))
	assert.True(t, sub(a, b).Equal(decimal.NewFromInt(7)))
	assert.True(t, mul(a, b).Equal(decimal.NewFromInt(30)))

	// Division - just check the value is approximately correct
	result := div(a, b)
	expected := decimal.NewFromFloat(3.3333333333)
	assert.True(t, result.Sub(expected).Abs().LessThan(decimal.NewFromFloat(0.0001)),
		"division result should be approximately 3.3333")

	// Division by zero
	assert.True(t, div(a, decimal.Zero).Equal(decimal.Zero))

	// Mod
	assert.True(t, mod(a, b).Equal(decimal.NewFromInt(1)))
	assert.True(t, mod(a, decimal.Zero).Equal(decimal.Zero))

	// Abs
	assert.True(t, absFunc(decimal.NewFromInt(-5)).Equal(decimal.NewFromInt(5)))
}

func TestRoundFunctions(t *testing.T) {
	v := decimal.NewFromFloat(1.555)

	// Round
	assert.Equal(t, "1.56", roundFunc(v, 2).String())
	assert.Equal(t, "1.6", roundFunc(v, 1).String())
	assert.Equal(t, "2", roundFunc(v, 0).String())

	// Round up
	assert.Equal(t, "1.56", roundUp(decimal.NewFromFloat(1.551), 2).String())

	// Round down
	assert.Equal(t, "1.55", roundDown(decimal.NewFromFloat(1.559), 2).String())
}

func TestMinMaxFunctions(t *testing.T) {
	vals := []interface{}{
		decimal.NewFromInt(5),
		decimal.NewFromInt(10),
		decimal.NewFromInt(3),
		decimal.NewFromInt(8),
	}

	assert.True(t, maxFunc(vals...).Equal(decimal.NewFromInt(10)))
	assert.True(t, minFunc(vals...).Equal(decimal.NewFromInt(3)))

	// Empty
	assert.True(t, maxFunc().Equal(decimal.Zero))
	assert.True(t, minFunc().Equal(decimal.Zero))
}

func TestSum(t *testing.T) {
	vals := []interface{}{
		decimal.NewFromInt(1),
		decimal.NewFromInt(2),
		decimal.NewFromInt(3),
	}
	assert.True(t, sum(vals...).Equal(decimal.NewFromInt(6)))
}

func TestSumField(t *testing.T) {
	type Item struct {
		Name   string
		Amount decimal.Decimal
		Price  float64
	}

	items := []Item{
		{Name: "Apple", Amount: decimal.NewFromInt(10), Price: 5.0},
		{Name: "Banana", Amount: decimal.NewFromInt(20), Price: 3.0},
		{Name: "Orange", Amount: decimal.NewFromInt(15), Price: 4.0},
	}

	// Test summing a decimal field
	result := sumField(items, "Amount")
	assert.True(t, result.Equal(decimal.NewFromInt(45)))

	// Test summing a float field
	resultPrice := sumField(items, "Price")
	assert.True(t, resultPrice.Equal(decimal.NewFromFloat(12.0)))

	// Test with map slice
	mapItems := []map[string]interface{}{
		{"Amount": decimal.NewFromInt(5)},
		{"Amount": decimal.NewFromInt(10)},
	}
	resultMap := sumField(mapItems, "Amount")
	assert.True(t, resultMap.Equal(decimal.NewFromInt(15)))

	// Test with non-slice returns zero
	assert.True(t, sumField("not a slice", "field").Equal(decimal.Zero))

	// Test with non-existent field returns zero
	assert.True(t, sumField(items, "NonExistent").Equal(decimal.Zero))
}

// =============================================================================
// Template Function Tests - Array/Slice
// =============================================================================

func TestFirstLast(t *testing.T) {
	slice := []interface{}{"a", "b", "c"}
	assert.Equal(t, "a", first(slice))
	assert.Equal(t, "c", last(slice))

	// Empty slice
	assert.Nil(t, first([]interface{}{}))
	assert.Nil(t, last([]interface{}{}))
}

func TestIndexFunc(t *testing.T) {
	slice := []interface{}{"a", "b", "c"}
	assert.Equal(t, "b", indexFunc(slice, 1))
	assert.Nil(t, indexFunc(slice, 10)) // Out of bounds
}

func TestLength(t *testing.T) {
	assert.Equal(t, 5, length("hello"))
	assert.Equal(t, 3, length([]interface{}{1, 2, 3}))
	assert.Equal(t, 0, length(123)) // Non-collection
}

func TestSeq(t *testing.T) {
	assert.Equal(t, []int{0, 1, 2, 3, 4}, seq(5))
	assert.Equal(t, []int{}, seq(0))
	assert.Equal(t, []int{}, seq(-1))
}

func TestEmpty(t *testing.T) {
	assert.True(t, empty(nil))
	assert.True(t, empty(""))
	assert.True(t, empty([]interface{}{}))
	assert.True(t, empty(0))
	assert.True(t, empty(false))

	assert.False(t, empty("hello"))
	assert.False(t, empty([]interface{}{1}))
	assert.False(t, empty(1))
	assert.False(t, empty(true))
}

// =============================================================================
// Template Function Tests - Conditional
// =============================================================================

func TestDefaultFunc(t *testing.T) {
	assert.Equal(t, "default", defaultFunc("", "default"))
	assert.Equal(t, "value", defaultFunc("value", "default"))
	assert.Equal(t, "default", defaultFunc(nil, "default"))
}

func TestTernary(t *testing.T) {
	assert.Equal(t, "yes", ternary(true, "yes", "no"))
	assert.Equal(t, "no", ternary(false, "yes", "no"))
}

func TestCoalesce(t *testing.T) {
	assert.Equal(t, "first", coalesce("first", "second"))
	assert.Equal(t, "second", coalesce("", "second"))
	assert.Equal(t, "second", coalesce(nil, "second"))
	assert.Nil(t, coalesce(nil, "", 0))
}

// =============================================================================
// Template Function Tests - UUID
// =============================================================================

func TestShortUUID(t *testing.T) {
	id := uuid.MustParse("12345678-1234-1234-1234-123456789012")
	assert.Equal(t, "12345678", shortUUID(id))
}

// =============================================================================
// Template Function Tests - Dict and List
// =============================================================================

func TestDict(t *testing.T) {
	d := dict("name", "John", "age", 30)
	assert.Equal(t, "John", d["name"])
	assert.Equal(t, 30, d["age"])
}

func TestList(t *testing.T) {
	l := list(1, 2, 3)
	assert.Equal(t, []interface{}{1, 2, 3}, l)
}

// =============================================================================
// Template Function Tests - Status Text
// =============================================================================

func TestStatusText(t *testing.T) {
	tests := []struct {
		status   string
		expected string
	}{
		{"DRAFT", "草稿"},
		{"CONFIRMED", "已确认"},
		{"SHIPPED", "已发货"},
		{"COMPLETED", "已完成"},
		{"CANCELLED", "已取消"},
		{"CASH", "现金"},
		{"WECHAT", "微信支付"},
		{"UNKNOWN", "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := statusText(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Template Function Tests - toDecimal helper
// =============================================================================

func TestToDecimal(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected decimal.Decimal
	}{
		{"decimal.Decimal", decimal.NewFromInt(10), decimal.NewFromInt(10)},
		{"*decimal.Decimal", func() *decimal.Decimal { d := decimal.NewFromInt(20); return &d }(), decimal.NewFromInt(20)},
		{"nil *decimal.Decimal", (*decimal.Decimal)(nil), decimal.Zero},
		{"int", 30, decimal.NewFromInt(30)},
		{"int32", int32(40), decimal.NewFromInt(40)},
		{"int64", int64(50), decimal.NewFromInt(50)},
		{"float32", float32(60.5), decimal.NewFromFloat(60.5)},
		{"float64", float64(70.5), decimal.NewFromFloat(70.5)},
		{"string", "80.5", decimal.NewFromFloat(80.5)},
		{"invalid string", "not a number", decimal.Zero},
		{"nil", nil, decimal.Zero},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toDecimal(tt.input)
			assert.True(t, result.Equal(tt.expected), "expected %s, got %s", tt.expected, result)
		})
	}
}

// =============================================================================
// Template Function Tests - toTime helper
// =============================================================================

func TestToTime(t *testing.T) {
	testTime := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)

	tests := []struct {
		name     string
		input    interface{}
		expected time.Time
	}{
		{"time.Time", testTime, testTime},
		{"*time.Time", &testTime, testTime},
		{"nil *time.Time", (*time.Time)(nil), time.Time{}},
		{"RFC3339 string", "2024-01-15T14:30:00Z", testTime},
		{"date string", "2024-01-15", time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)},
		{"invalid string", "not a date", time.Time{}},
		{"unix timestamp", int64(1705330200), time.Unix(1705330200, 0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toTime(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Integration Test - Full Document Rendering
// =============================================================================

func TestTemplateEngine_FullDocumentRender(t *testing.T) {
	engine := NewTemplateEngine()
	ctx := context.Background()

	templateContent := `<!DOCTYPE html>
<html>
<head>
    <title>销售订单 - {{.OrderNumber}}</title>
</head>
<body>
    <h1>销售订单</h1>
    <p>订单号: {{.OrderNumber}}</p>
    <p>客户: {{.CustomerName}}</p>
    <p>日期: {{formatDate .OrderDate}}</p>

    <table>
        <thead>
            <tr>
                <th>序号</th>
                <th>商品</th>
                <th>数量</th>
                <th>单价</th>
                <th>金额</th>
            </tr>
        </thead>
        <tbody>
            {{range $index, $item := .Items}}
            <tr>
                <td>{{add $index 1}}</td>
                <td>{{$item.ProductName}}</td>
                <td>{{formatDecimal $item.Quantity 2}}</td>
                <td>{{formatMoney $item.UnitPrice}}</td>
                <td>{{formatMoney $item.Amount}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>

    <p>合计: {{formatMoney .TotalAmount}}</p>
    <p>大写: {{moneyToChinese .TotalAmount}}</p>

    {{if gt .DiscountAmount 0}}
    <p>折扣: {{formatMoney .DiscountAmount}}</p>
    {{end}}

    <p>应付: {{formatMoney .PayableAmount}}</p>
    <p>应付大写: {{moneyToChinese .PayableAmount}}</p>
</body>
</html>`

	template := &printing.PrintTemplate{}
	template.ID = uuid.New()
	template.Content = templateContent

	orderDate := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	data := map[string]interface{}{
		"OrderNumber":  "SO-2024-0001",
		"CustomerName": "张三",
		"OrderDate":    orderDate,
		"Items": []map[string]interface{}{
			{
				"ProductName": "苹果",
				"Quantity":    decimal.NewFromInt(10),
				"UnitPrice":   decimal.NewFromFloat(5.50),
				"Amount":      decimal.NewFromFloat(55.00),
			},
			{
				"ProductName": "香蕉",
				"Quantity":    decimal.NewFromInt(20),
				"UnitPrice":   decimal.NewFromFloat(3.00),
				"Amount":      decimal.NewFromFloat(60.00),
			},
		},
		"TotalAmount":    decimal.NewFromFloat(115.00),
		"DiscountAmount": decimal.NewFromFloat(10.00),
		"PayableAmount":  decimal.NewFromFloat(105.00),
	}

	result, err := engine.Render(ctx, &RenderTemplateRequest{
		Template: template,
		Data:     data,
	})

	require.NoError(t, err)

	// Verify rendered content
	assert.Contains(t, result.HTML, "SO-2024-0001")
	assert.Contains(t, result.HTML, "张三")
	assert.Contains(t, result.HTML, "2024-01-15")
	assert.Contains(t, result.HTML, "苹果")
	assert.Contains(t, result.HTML, "香蕉")
	assert.Contains(t, result.HTML, "¥55.00")
	assert.Contains(t, result.HTML, "¥60.00")
	assert.Contains(t, result.HTML, "¥115.00")
	assert.Contains(t, result.HTML, "壹佰零伍元整") // 105 in Chinese
	assert.Contains(t, result.HTML, "¥10.00") // Discount
}
