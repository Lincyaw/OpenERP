package printing

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"maps"
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/erp/backend/internal/domain/printing"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// TemplateEngine handles rendering HTML templates with business data.
// It uses Go's html/template package with custom functions for formatting.
type TemplateEngine struct {
	funcMap template.FuncMap
}

// TemplateEngineOption configures the template engine
type TemplateEngineOption func(*TemplateEngine)

// NewTemplateEngine creates a new template engine with default configuration
func NewTemplateEngine(opts ...TemplateEngineOption) *TemplateEngine {
	e := &TemplateEngine{}

	// Initialize template functions
	e.funcMap = template.FuncMap{
		// Money formatting
		"formatMoney":    formatMoney,
		"formatMoneyRaw": formatMoneyRaw,
		"moneyToChinese": moneyToChinese,

		// Date formatting
		"formatDate":     formatDate,
		"formatDateTime": formatDateTime,
		"formatTime":     formatTime,

		// Number formatting
		"formatDecimal": formatDecimal,
		"formatInt":     formatInt,
		"formatPercent": formatPercent,

		// String utilities
		"truncate":   truncate,
		"padLeft":    padLeft,
		"padRight":   padRight,
		"join":       strings.Join,
		"upper":      strings.ToUpper,
		"lower":      strings.ToLower,
		"title":      titleCase,
		"trim":       strings.TrimSpace,
		"replace":    strings.ReplaceAll,
		"split":      strings.Split,
		"contains":   strings.Contains,
		"hasPrefix":  strings.HasPrefix,
		"hasSuffix":  strings.HasSuffix,
		"trimPrefix": strings.TrimPrefix,
		"trimSuffix": strings.TrimSuffix,

		// Comparison and logic
		"eq": func(a, b interface{}) bool { return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b) },
		"ne": func(a, b interface{}) bool { return fmt.Sprintf("%v", a) != fmt.Sprintf("%v", b) },
		"lt": ltFunc,
		"le": leFunc,
		"gt": gtFunc,
		"ge": geFunc,

		// Arithmetic
		"add":      add,
		"sub":      sub,
		"mul":      mul,
		"div":      div,
		"mod":      mod,
		"abs":      absFunc,
		"round":    roundFunc,
		"roundUp":  roundUp,
		"roundDn":  roundDown,
		"max":      maxFunc,
		"min":      minFunc,
		"sum":      sum,
		"sumField": sumField,

		// Array/slice utilities
		"first":    first,
		"last":     last,
		"index":    indexFunc,
		"len":      length,
		"seq":      seq,
		"repeat":   strings.Repeat,
		"in":       inSlice,
		"empty":    empty,
		"notEmpty": notEmpty,

		// Conditional
		"default":  defaultFunc,
		"ternary":  ternary,
		"coalesce": coalesce,

		// Safe HTML
		"safeHTML": safeHTML,
		"safeCSS":  safeCSS,
		"safeJS":   safeJS,
		"safeURL":  safeURL,

		// UUID utilities
		"shortUUID": shortUUID,

		// Misc
		"now":        time.Now,
		"dict":       dict,
		"list":       list,
		"statusText": statusText,
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// RenderRequest represents a request to render a template
type RenderTemplateRequest struct {
	// Template is the print template to render
	Template *printing.PrintTemplate
	// Data is the business data to bind to the template
	Data interface{}
	// AdditionalFuncs are extra template functions (optional)
	AdditionalFuncs template.FuncMap
}

// RenderResult contains the rendered HTML output
type RenderTemplateResult struct {
	// HTML is the rendered HTML content
	HTML string
	// RenderDuration is how long the rendering took
	RenderDuration time.Duration
}

// Render renders a print template with the provided data
func (e *TemplateEngine) Render(ctx context.Context, req *RenderTemplateRequest) (*RenderTemplateResult, error) {
	if req == nil {
		return nil, NewRenderError(ErrCodeInvalidHTML, "render request is nil", nil)
	}
	if req.Template == nil {
		return nil, NewRenderError(ErrCodeInvalidHTML, "template is nil", nil)
	}
	if req.Template.Content == "" {
		return nil, NewRenderError(ErrCodeInvalidHTML, "template content is empty", nil)
	}

	startTime := time.Now()

	// Create template with functions
	funcMap := make(template.FuncMap)
	maps.Copy(funcMap, e.funcMap)
	if req.AdditionalFuncs != nil {
		maps.Copy(funcMap, req.AdditionalFuncs)
	}

	// Parse template
	tmpl, err := template.New(req.Template.ID.String()).Funcs(funcMap).Parse(req.Template.Content)
	if err != nil {
		return nil, NewRenderError(ErrCodeInvalidHTML, "failed to parse template", err)
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, req.Data); err != nil {
		return nil, NewRenderError(ErrCodeRenderFailed, "failed to execute template", err)
	}

	return &RenderTemplateResult{
		HTML:           buf.String(),
		RenderDuration: time.Since(startTime),
	}, nil
}

// RenderString renders a template string with the provided data
func (e *TemplateEngine) RenderString(ctx context.Context, name, content string, data interface{}) (string, error) {
	if content == "" {
		return "", NewRenderError(ErrCodeInvalidHTML, "template content is empty", nil)
	}

	// Parse template
	tmpl, err := template.New(name).Funcs(e.funcMap).Parse(content)
	if err != nil {
		return "", NewRenderError(ErrCodeInvalidHTML, "failed to parse template", err)
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", NewRenderError(ErrCodeRenderFailed, "failed to execute template", err)
	}

	return buf.String(), nil
}

// GetFuncMap returns a copy of the template function map
func (e *TemplateEngine) GetFuncMap() template.FuncMap {
	funcMap := make(template.FuncMap, len(e.funcMap))
	maps.Copy(funcMap, e.funcMap)
	return funcMap
}

// =============================================================================
// Template Functions - Money Formatting
// =============================================================================

// formatMoney formats a decimal value as currency with symbol
// Example: 1234.56 -> "¥1,234.56"
func formatMoney(v interface{}) string {
	d := toDecimal(v)
	return "¥" + formatMoneyRaw(d)
}

// formatMoneyRaw formats a decimal value as currency without symbol
// Example: 1234.56 -> "1,234.56"
func formatMoneyRaw(v interface{}) string {
	d := toDecimal(v)
	sign := ""
	if d.IsNegative() {
		sign = "-"
		d = d.Abs()
	}

	// Split into integer and decimal parts
	parts := strings.Split(d.StringFixed(2), ".")
	intPart := parts[0]
	decPart := "00"
	if len(parts) > 1 {
		decPart = parts[1]
	}

	// Add thousand separators
	var result strings.Builder
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(c)
	}

	return sign + result.String() + "." + decPart
}

// moneyToChinese converts a decimal money value to Chinese uppercase format
// Example: 1234.56 -> "壹仟贰佰叁拾肆元伍角陆分"
func moneyToChinese(v interface{}) string {
	d := toDecimal(v)
	if d.IsZero() {
		return "零元整"
	}

	sign := ""
	if d.IsNegative() {
		sign = "负"
		d = d.Abs()
	}

	// Chinese numerals
	chnNum := []string{"零", "壹", "贰", "叁", "肆", "伍", "陆", "柒", "捌", "玖"}
	chnUnit := []string{"", "拾", "佰", "仟"}
	chnBigUnit := []string{"", "万", "亿", "万亿"}

	// Separate integer and decimal parts
	// Multiply by 100 to get cents, then convert
	cents := d.Mul(decimal.NewFromInt(100)).IntPart()
	yuan := cents / 100
	jiao := (cents % 100) / 10
	fen := cents % 10

	var result strings.Builder
	result.WriteString(sign)

	// Convert yuan part
	if yuan > 0 {
		yuanStr := fmt.Sprintf("%d", yuan)
		length := len(yuanStr)
		zeroFlag := false
		lastBigUnitWritten := -1 // Track to avoid duplicate big units

		for i, c := range yuanStr {
			n := int(c - '0')
			pos := length - i - 1
			bigUnitPos := pos / 4
			unitPos := pos % 4

			if n == 0 {
				zeroFlag = true
				// Don't add big unit for zero digits - it will be added by non-zero digit or skipped
			} else {
				if zeroFlag && result.Len() > len(sign) {
					// Only add zero if there's already content (avoid leading zeros)
					result.WriteString(chnNum[0])
					zeroFlag = false
				}
				result.WriteString(chnNum[n])
				result.WriteString(chnUnit[unitPos])
				if unitPos == 0 && bigUnitPos > 0 && bigUnitPos != lastBigUnitWritten {
					result.WriteString(chnBigUnit[bigUnitPos])
					lastBigUnitWritten = bigUnitPos
				}
			}
		}
		result.WriteString("元")
	}

	// Convert jiao and fen
	if jiao == 0 && fen == 0 {
		result.WriteString("整")
	} else {
		if jiao == 0 && yuan > 0 {
			result.WriteString("零")
		} else if jiao > 0 {
			result.WriteString(chnNum[jiao])
			result.WriteString("角")
		}
		if fen > 0 {
			if jiao == 0 && yuan > 0 {
				// Zero already added
			}
			result.WriteString(chnNum[fen])
			result.WriteString("分")
		}
	}

	return result.String()
}

// =============================================================================
// Template Functions - Date Formatting
// =============================================================================

// formatDate formats a time value as date string
// Example: time.Now() -> "2024-01-15"
func formatDate(v interface{}) string {
	t := toTime(v)
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

// formatDateTime formats a time value as datetime string
// Example: time.Now() -> "2024-01-15 14:30:00"
func formatDateTime(v interface{}) string {
	t := toTime(v)
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

// formatTime formats a time value as time string
// Example: time.Now() -> "14:30:00"
func formatTime(v interface{}) string {
	t := toTime(v)
	if t.IsZero() {
		return ""
	}
	return t.Format("15:04:05")
}

// =============================================================================
// Template Functions - Number Formatting
// =============================================================================

// formatDecimal formats a decimal with specified precision
func formatDecimal(v interface{}, precision int) string {
	d := toDecimal(v)
	return d.StringFixed(int32(precision))
}

// formatInt formats as integer
func formatInt(v interface{}) string {
	d := toDecimal(v)
	return d.Round(0).String()
}

// formatPercent formats as percentage
// Example: 0.15 -> "15%"
func formatPercent(v interface{}, precision int) string {
	d := toDecimal(v)
	percent := d.Mul(decimal.NewFromInt(100))
	return percent.StringFixed(int32(precision)) + "%"
}

// =============================================================================
// Template Functions - String Utilities
// =============================================================================

// truncate truncates a string to max runes with optional suffix
// Uses rune count for proper UTF-8 handling
func truncate(s string, max int, suffix ...string) string {
	suf := "..."
	if len(suffix) > 0 {
		suf = suffix[0]
	}
	runes := []rune(s)
	sufRunes := []rune(suf)
	if len(runes) <= max {
		return s
	}
	if max <= len(sufRunes) {
		return suf[:max]
	}
	return string(runes[:max-len(sufRunes)]) + suf
}

// padLeft pads string on the left to reach desired length
func padLeft(s string, length int, pad string) string {
	if len(s) >= length || pad == "" {
		return s
	}
	padLen := length - len(s)
	padding := strings.Repeat(pad, (padLen/len(pad))+1)
	return padding[:padLen] + s
}

// padRight pads string on the right to reach desired length
func padRight(s string, length int, pad string) string {
	if len(s) >= length || pad == "" {
		return s
	}
	padLen := length - len(s)
	padding := strings.Repeat(pad, (padLen/len(pad))+1)
	return s + padding[:padLen]
}

// titleCase converts string to title case using proper Unicode handling
func titleCase(s string) string {
	caser := cases.Title(language.English)
	return caser.String(s)
}

// =============================================================================
// Template Functions - Comparison
// =============================================================================

func ltFunc(a, b interface{}) bool {
	return toDecimal(a).LessThan(toDecimal(b))
}

func leFunc(a, b interface{}) bool {
	return toDecimal(a).LessThanOrEqual(toDecimal(b))
}

func gtFunc(a, b interface{}) bool {
	return toDecimal(a).GreaterThan(toDecimal(b))
}

func geFunc(a, b interface{}) bool {
	return toDecimal(a).GreaterThanOrEqual(toDecimal(b))
}

// =============================================================================
// Template Functions - Arithmetic
// =============================================================================

func add(a, b interface{}) decimal.Decimal {
	return toDecimal(a).Add(toDecimal(b))
}

func sub(a, b interface{}) decimal.Decimal {
	return toDecimal(a).Sub(toDecimal(b))
}

func mul(a, b interface{}) decimal.Decimal {
	return toDecimal(a).Mul(toDecimal(b))
}

func div(a, b interface{}) decimal.Decimal {
	bDec := toDecimal(b)
	if bDec.IsZero() {
		return decimal.Zero
	}
	return toDecimal(a).Div(bDec)
}

func mod(a, b interface{}) decimal.Decimal {
	bDec := toDecimal(b)
	if bDec.IsZero() {
		return decimal.Zero
	}
	return toDecimal(a).Mod(bDec)
}

func absFunc(v interface{}) decimal.Decimal {
	return toDecimal(v).Abs()
}

func roundFunc(v interface{}, places int) decimal.Decimal {
	return toDecimal(v).Round(int32(places))
}

func roundUp(v interface{}, places int) decimal.Decimal {
	d := toDecimal(v)
	mult := decimal.NewFromFloat(math.Pow(10, float64(places)))
	return d.Mul(mult).Ceil().Div(mult)
}

func roundDown(v interface{}, places int) decimal.Decimal {
	d := toDecimal(v)
	mult := decimal.NewFromFloat(math.Pow(10, float64(places)))
	return d.Mul(mult).Floor().Div(mult)
}

func maxFunc(vals ...interface{}) decimal.Decimal {
	if len(vals) == 0 {
		return decimal.Zero
	}
	result := toDecimal(vals[0])
	for _, v := range vals[1:] {
		d := toDecimal(v)
		if d.GreaterThan(result) {
			result = d
		}
	}
	return result
}

func minFunc(vals ...interface{}) decimal.Decimal {
	if len(vals) == 0 {
		return decimal.Zero
	}
	result := toDecimal(vals[0])
	for _, v := range vals[1:] {
		d := toDecimal(v)
		if d.LessThan(result) {
			result = d
		}
	}
	return result
}

func sum(vals ...interface{}) decimal.Decimal {
	result := decimal.Zero
	for _, v := range vals {
		result = result.Add(toDecimal(v))
	}
	return result
}

// sumField sums a field from a slice of structs/maps
// Usage in template: {{ sumField .Items "Amount" }}
func sumField(slice interface{}, field string) decimal.Decimal {
	result := decimal.Zero
	rv := reflect.ValueOf(slice)
	if rv.Kind() != reflect.Slice {
		return result
	}
	for i := 0; i < rv.Len(); i++ {
		elem := rv.Index(i)
		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}
		var fieldVal reflect.Value
		switch elem.Kind() {
		case reflect.Struct:
			fieldVal = elem.FieldByName(field)
		case reflect.Map:
			fieldVal = elem.MapIndex(reflect.ValueOf(field))
		}
		if fieldVal.IsValid() {
			result = result.Add(toDecimal(fieldVal.Interface()))
		}
	}
	return result
}

// =============================================================================
// Template Functions - Array/Slice
// =============================================================================

func first(v interface{}) interface{} {
	switch val := v.(type) {
	case []interface{}:
		if len(val) > 0 {
			return val[0]
		}
	case []string:
		if len(val) > 0 {
			return val[0]
		}
	}
	return nil
}

func last(v interface{}) interface{} {
	switch val := v.(type) {
	case []interface{}:
		if len(val) > 0 {
			return val[len(val)-1]
		}
	case []string:
		if len(val) > 0 {
			return val[len(val)-1]
		}
	}
	return nil
}

func indexFunc(v interface{}, i int) interface{} {
	switch val := v.(type) {
	case []interface{}:
		if i >= 0 && i < len(val) {
			return val[i]
		}
	case []string:
		if i >= 0 && i < len(val) {
			return val[i]
		}
	}
	return nil
}

func length(v interface{}) int {
	switch val := v.(type) {
	case string:
		return len(val)
	case []interface{}:
		return len(val)
	case []string:
		return len(val)
	case map[string]interface{}:
		return len(val)
	default:
		return 0
	}
}

// seq generates a sequence of integers from 0 to n-1
func seq(n int) []int {
	if n <= 0 {
		return []int{}
	}
	result := make([]int, n)
	for i := 0; i < n; i++ {
		result[i] = i
	}
	return result
}

func inSlice(needle interface{}, haystack []interface{}) bool {
	needleStr := fmt.Sprintf("%v", needle)
	for _, v := range haystack {
		if fmt.Sprintf("%v", v) == needleStr {
			return true
		}
	}
	return false
}

func empty(v interface{}) bool {
	if v == nil {
		return true
	}
	switch val := v.(type) {
	case string:
		return val == ""
	case []interface{}:
		return len(val) == 0
	case []string:
		return len(val) == 0
	case map[string]interface{}:
		return len(val) == 0
	case int:
		return val == 0
	case int64:
		return val == 0
	case float64:
		return val == 0
	case bool:
		return !val
	}
	return false
}

func notEmpty(v interface{}) bool {
	return !empty(v)
}

// =============================================================================
// Template Functions - Conditional
// =============================================================================

func defaultFunc(val, def interface{}) interface{} {
	if empty(val) {
		return def
	}
	return val
}

func ternary(condition bool, trueVal, falseVal interface{}) interface{} {
	if condition {
		return trueVal
	}
	return falseVal
}

func coalesce(vals ...interface{}) interface{} {
	for _, v := range vals {
		if !empty(v) {
			return v
		}
	}
	return nil
}

// =============================================================================
// Template Functions - Safe HTML
// =============================================================================
// SECURITY WARNING: The following functions bypass Go's built-in HTML escaping.
// ONLY use these functions with trusted content that is NOT user-generated.
// Using these functions with user-controlled input may create XSS vulnerabilities.
//
// Safe usage:
//   - Static HTML snippets defined in templates
//   - Pre-validated and sanitized content from trusted sources
//   - System-generated content that cannot be manipulated by users
//
// NEVER use with:
//   - Customer names, remarks, or any user-provided text
//   - Content from external APIs without sanitization
//   - Any field that could be modified by end users
// =============================================================================

// safeHTML marks a string as safe HTML, bypassing automatic escaping.
// SECURITY: Only use with trusted, non-user-generated content.
func safeHTML(s string) template.HTML {
	return template.HTML(s)
}

// safeCSS marks a string as safe CSS, bypassing automatic escaping.
// SECURITY: Only use with trusted, non-user-generated content.
func safeCSS(s string) template.CSS {
	return template.CSS(s)
}

// safeJS marks a string as safe JavaScript, bypassing automatic escaping.
// SECURITY: Only use with trusted, non-user-generated content.
func safeJS(s string) template.JS {
	return template.JS(s)
}

// safeURL marks a string as safe URL, bypassing automatic escaping.
// SECURITY: Only use with trusted, non-user-generated content.
func safeURL(s string) template.URL {
	return template.URL(s)
}

// =============================================================================
// Template Functions - UUID
// =============================================================================

func shortUUID(id uuid.UUID) string {
	s := id.String()
	if len(s) >= 8 {
		return s[:8]
	}
	return s
}

// =============================================================================
// Template Functions - Dict and List
// =============================================================================

// dict creates a map from key-value pairs
func dict(pairs ...interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for i := 0; i < len(pairs)-1; i += 2 {
		key, ok := pairs[i].(string)
		if ok {
			result[key] = pairs[i+1]
		}
	}
	return result
}

// list creates a slice from values
func list(vals ...interface{}) []interface{} {
	return vals
}

// =============================================================================
// Template Functions - Status Text
// =============================================================================

// statusText converts status codes to display text
func statusText(status string) string {
	statusMap := map[string]string{
		// Order statuses
		"DRAFT":     "草稿",
		"CONFIRMED": "已确认",
		"SHIPPED":   "已发货",
		"COMPLETED": "已完成",
		"CANCELLED": "已取消",
		// Voucher statuses
		"ALLOCATED": "已核销",
		// Return statuses
		"PENDING":  "待处理",
		"APPROVED": "已审批",
		"REJECTED": "已拒绝",
		"RECEIVED": "已收货",
		// Payment methods
		"CASH":          "现金",
		"BANK_TRANSFER": "银行转账",
		"WECHAT":        "微信支付",
		"ALIPAY":        "支付宝",
		"CHECK":         "支票",
		"BALANCE":       "余额抵扣",
		"OTHER":         "其他",
	}
	if text, ok := statusMap[status]; ok {
		return text
	}
	return status
}

// =============================================================================
// Helper Functions
// =============================================================================

// toDecimal converts various types to decimal.Decimal
func toDecimal(v interface{}) decimal.Decimal {
	switch val := v.(type) {
	case decimal.Decimal:
		return val
	case *decimal.Decimal:
		if val == nil {
			return decimal.Zero
		}
		return *val
	case int:
		return decimal.NewFromInt(int64(val))
	case int32:
		return decimal.NewFromInt(int64(val))
	case int64:
		return decimal.NewFromInt(val)
	case float32:
		return decimal.NewFromFloat(float64(val))
	case float64:
		return decimal.NewFromFloat(val)
	case string:
		d, err := decimal.NewFromString(val)
		if err != nil {
			return decimal.Zero
		}
		return d
	default:
		return decimal.Zero
	}
}

// toTime converts various types to time.Time
func toTime(v interface{}) time.Time {
	switch val := v.(type) {
	case time.Time:
		return val
	case *time.Time:
		if val == nil {
			return time.Time{}
		}
		return *val
	case string:
		// Try common formats
		formats := []string{
			time.RFC3339,
			"2006-01-02T15:04:05Z",
			"2006-01-02 15:04:05",
			"2006-01-02",
		}
		for _, f := range formats {
			if t, err := time.Parse(f, val); err == nil {
				return t
			}
		}
		return time.Time{}
	case int64:
		return time.Unix(val, 0)
	default:
		return time.Time{}
	}
}
