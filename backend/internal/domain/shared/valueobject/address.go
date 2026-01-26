package valueobject

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
)

// Address is a value object representing a physical address
// It is immutable - all operations return new Address instances
// Fields: Province (省), City (市), District (区/县), Detail (详细地址), PostalCode (邮编)
type Address struct {
	province   string
	city       string
	district   string
	detail     string
	postalCode string
	country    string
}

// AddressOption is a functional option for configuring Address
type AddressOption func(*Address)

// WithPostalCode sets the postal code for the address
func WithPostalCode(postalCode string) AddressOption {
	return func(a *Address) {
		a.postalCode = strings.TrimSpace(postalCode)
	}
}

// WithCountry sets the country for the address
func WithCountry(country string) AddressOption {
	return func(a *Address) {
		a.country = strings.TrimSpace(country)
	}
}

// NewAddress creates a new Address with the required fields
// Province, city, and detail are required; district and postal code are optional
func NewAddress(province, city, district, detail string, opts ...AddressOption) (Address, error) {
	province = strings.TrimSpace(province)
	city = strings.TrimSpace(city)
	district = strings.TrimSpace(district)
	detail = strings.TrimSpace(detail)

	if err := validateProvince(province); err != nil {
		return Address{}, err
	}
	if err := validateCity(city); err != nil {
		return Address{}, err
	}
	if err := validateDistrict(district); err != nil {
		return Address{}, err
	}
	if err := validateDetail(detail); err != nil {
		return Address{}, err
	}

	addr := Address{
		province: province,
		city:     city,
		district: district,
		detail:   detail,
		country:  "中国", // Default country
	}

	for _, opt := range opts {
		opt(&addr)
	}

	// Validate postal code if set
	if addr.postalCode != "" {
		if err := validatePostalCode(addr.postalCode); err != nil {
			return Address{}, err
		}
	}

	// Validate country if set
	if addr.country != "" && len(addr.country) > 100 {
		return Address{}, fmt.Errorf("country cannot exceed 100 characters")
	}

	return addr, nil
}

// NewAddressWithPostalCode creates a new Address including postal code
func NewAddressWithPostalCode(province, city, district, detail, postalCode string) (Address, error) {
	return NewAddress(province, city, district, detail, WithPostalCode(postalCode))
}

// NewAddressFull creates a new Address with all fields including country
func NewAddressFull(province, city, district, detail, postalCode, country string) (Address, error) {
	return NewAddress(province, city, district, detail, WithPostalCode(postalCode), WithCountry(country))
}

// MustNewAddress creates a new Address, panics on error
func MustNewAddress(province, city, district, detail string, opts ...AddressOption) Address {
	addr, err := NewAddress(province, city, district, detail, opts...)
	if err != nil {
		panic(err)
	}
	return addr
}

// EmptyAddress returns an empty address (for optional address fields)
func EmptyAddress() Address {
	return Address{}
}

// Province returns the province
func (a Address) Province() string {
	return a.province
}

// City returns the city
func (a Address) City() string {
	return a.city
}

// District returns the district
func (a Address) District() string {
	return a.district
}

// Detail returns the detailed address
func (a Address) Detail() string {
	return a.detail
}

// PostalCode returns the postal code
func (a Address) PostalCode() string {
	return a.postalCode
}

// Country returns the country
func (a Address) Country() string {
	return a.country
}

// IsEmpty returns true if the address is empty (all fields are blank)
func (a Address) IsEmpty() bool {
	return a.province == "" && a.city == "" && a.district == "" && a.detail == ""
}

// FullAddress returns the complete formatted address string
// Format: Country Province City District Detail PostalCode
func (a Address) FullAddress() string {
	if a.IsEmpty() {
		return ""
	}

	parts := make([]string, 0, 6)
	if a.country != "" {
		parts = append(parts, a.country)
	}
	if a.province != "" {
		parts = append(parts, a.province)
	}
	if a.city != "" {
		parts = append(parts, a.city)
	}
	if a.district != "" {
		parts = append(parts, a.district)
	}
	if a.detail != "" {
		parts = append(parts, a.detail)
	}
	if a.postalCode != "" {
		parts = append(parts, a.postalCode)
	}
	return strings.Join(parts, " ")
}

// ShortAddress returns a shortened address (City + District + Detail)
func (a Address) ShortAddress() string {
	if a.IsEmpty() {
		return ""
	}

	parts := make([]string, 0, 3)
	if a.city != "" {
		parts = append(parts, a.city)
	}
	if a.district != "" {
		parts = append(parts, a.district)
	}
	if a.detail != "" {
		parts = append(parts, a.detail)
	}
	return strings.Join(parts, " ")
}

// RegionAddress returns region-level address (Province + City + District)
func (a Address) RegionAddress() string {
	if a.IsEmpty() {
		return ""
	}

	parts := make([]string, 0, 3)
	if a.province != "" {
		parts = append(parts, a.province)
	}
	if a.city != "" {
		parts = append(parts, a.city)
	}
	if a.district != "" {
		parts = append(parts, a.district)
	}
	return strings.Join(parts, " ")
}

// ChineseFullAddress returns address in Chinese format without spaces
// Format: 省市区详细地址
func (a Address) ChineseFullAddress() string {
	if a.IsEmpty() {
		return ""
	}

	var sb strings.Builder
	if a.province != "" {
		sb.WriteString(a.province)
	}
	if a.city != "" {
		sb.WriteString(a.city)
	}
	if a.district != "" {
		sb.WriteString(a.district)
	}
	if a.detail != "" {
		sb.WriteString(a.detail)
	}
	return sb.String()
}

// String returns a string representation of the address
func (a Address) String() string {
	return a.FullAddress()
}

// Equals returns true if both addresses are equal
func (a Address) Equals(other Address) bool {
	return a.province == other.province &&
		a.city == other.city &&
		a.district == other.district &&
		a.detail == other.detail &&
		a.postalCode == other.postalCode &&
		a.country == other.country
}

// SameRegion returns true if both addresses are in the same region (province, city, district)
func (a Address) SameRegion(other Address) bool {
	return a.province == other.province &&
		a.city == other.city &&
		a.district == other.district
}

// SameCity returns true if both addresses are in the same city
func (a Address) SameCity(other Address) bool {
	return a.province == other.province && a.city == other.city
}

// SameProvince returns true if both addresses are in the same province
func (a Address) SameProvince(other Address) bool {
	return a.province == other.province
}

// WithProvince returns a new Address with the updated province
func (a Address) WithProvince(province string) (Address, error) {
	return NewAddress(province, a.city, a.district, a.detail,
		WithPostalCode(a.postalCode), WithCountry(a.country))
}

// WithCity returns a new Address with the updated city
func (a Address) WithCity(city string) (Address, error) {
	return NewAddress(a.province, city, a.district, a.detail,
		WithPostalCode(a.postalCode), WithCountry(a.country))
}

// WithDistrict returns a new Address with the updated district
func (a Address) WithDistrict(district string) (Address, error) {
	return NewAddress(a.province, a.city, district, a.detail,
		WithPostalCode(a.postalCode), WithCountry(a.country))
}

// WithDetail returns a new Address with the updated detail
func (a Address) WithDetail(detail string) (Address, error) {
	return NewAddress(a.province, a.city, a.district, detail,
		WithPostalCode(a.postalCode), WithCountry(a.country))
}

// WithUpdatedPostalCode returns a new Address with the updated postal code
func (a Address) WithUpdatedPostalCode(postalCode string) (Address, error) {
	return NewAddress(a.province, a.city, a.district, a.detail,
		WithPostalCode(postalCode), WithCountry(a.country))
}

// WithUpdatedCountry returns a new Address with the updated country
func (a Address) WithUpdatedCountry(country string) (Address, error) {
	return NewAddress(a.province, a.city, a.district, a.detail,
		WithPostalCode(a.postalCode), WithCountry(country))
}

// addressJSON is used for JSON marshaling/unmarshaling
type addressJSON struct {
	Province   string `json:"province"`
	City       string `json:"city"`
	District   string `json:"district"`
	Detail     string `json:"detail"`
	PostalCode string `json:"postalCode,omitempty"`
	Country    string `json:"country,omitempty"`
}

// MarshalJSON implements json.Marshaler
func (a Address) MarshalJSON() ([]byte, error) {
	return json.Marshal(addressJSON{
		Province:   a.province,
		City:       a.city,
		District:   a.district,
		Detail:     a.detail,
		PostalCode: a.postalCode,
		Country:    a.country,
	})
}

// UnmarshalJSON implements json.Unmarshaler for deserialization purposes.
//
// IMPORTANT: This method exists ONLY to support JSON deserialization scenarios
// (e.g., API request binding, database JSON column retrieval).
// It is NOT intended for general Address creation from JSON data.
//
// For programmatic JSON parsing where you want explicit error handling and
// clearer intent, use ParseAddressFromJSON instead.
//
// The method maintains immutability by delegating to NewAddressFull factory,
// ensuring all validation rules are applied consistently.
func (a *Address) UnmarshalJSON(data []byte) error {
	var v addressJSON
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	// Allow empty addresses from JSON
	if v.Province == "" && v.City == "" && v.District == "" && v.Detail == "" {
		*a = EmptyAddress()
		return nil
	}

	addr, err := NewAddressFull(v.Province, v.City, v.District, v.Detail, v.PostalCode, v.Country)
	if err != nil {
		return err
	}
	*a = addr
	return nil
}

// ParseAddressFromJSON creates an Address from JSON data.
// This is the recommended way to create an Address from JSON when you want
// explicit control over error handling and clearer intent in your code.
//
// Unlike UnmarshalJSON (which is called implicitly by json.Unmarshal),
// this factory function makes the parsing operation explicit and returns
// a new Address value (not a pointer), maintaining immutability semantics.
//
// Example:
//
//	jsonData := []byte(`{"province":"北京市","city":"北京市","district":"海淀区","detail":"中关村"}`)
//	addr, err := valueobject.ParseAddressFromJSON(jsonData)
//	if err != nil {
//	    // handle parsing error
//	}
func ParseAddressFromJSON(data []byte) (Address, error) {
	var v addressJSON
	if err := json.Unmarshal(data, &v); err != nil {
		return Address{}, fmt.Errorf("failed to parse address JSON: %w", err)
	}

	// Allow empty addresses from JSON
	if v.Province == "" && v.City == "" && v.District == "" && v.Detail == "" {
		return EmptyAddress(), nil
	}

	return NewAddressFull(v.Province, v.City, v.District, v.Detail, v.PostalCode, v.Country)
}

// AddressDTO is a data transfer object for database operations
// This allows Address to be stored as a JSON column
type AddressDTO struct {
	Province   string `json:"province"`
	City       string `json:"city"`
	District   string `json:"district"`
	Detail     string `json:"detail"`
	PostalCode string `json:"postalCode,omitempty"`
	Country    string `json:"country,omitempty"`
}

// ToDTO converts Address to AddressDTO for database storage
func (a Address) ToDTO() AddressDTO {
	return AddressDTO{
		Province:   a.province,
		City:       a.city,
		District:   a.district,
		Detail:     a.detail,
		PostalCode: a.postalCode,
		Country:    a.country,
	}
}

// ToAddress converts AddressDTO back to Address
func (dto AddressDTO) ToAddress() (Address, error) {
	if dto.Province == "" && dto.City == "" && dto.District == "" && dto.Detail == "" {
		return EmptyAddress(), nil
	}
	return NewAddressFull(dto.Province, dto.City, dto.District, dto.Detail, dto.PostalCode, dto.Country)
}

// MustToAddress converts AddressDTO to Address, panics on error
func (dto AddressDTO) MustToAddress() Address {
	addr, err := dto.ToAddress()
	if err != nil {
		panic(err)
	}
	return addr
}

// Value implements driver.Valuer for database storage
// Stores as JSON string
func (a Address) Value() (driver.Value, error) {
	if a.IsEmpty() {
		return nil, nil
	}
	return json.Marshal(a)
}

// Scan implements sql.Scanner for database retrieval.
//
// IMPORTANT: This method exists ONLY to support GORM/database scanning scenarios.
// It is NOT intended for general Address creation from raw data.
//
// The method maintains immutability by delegating to UnmarshalJSON, which in turn
// uses the NewAddressFull factory, ensuring all validation rules are applied.
func (a *Address) Scan(value any) error {
	if value == nil {
		*a = EmptyAddress()
		return nil
	}

	var data []byte
	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return fmt.Errorf("cannot scan %T into Address", value)
	}

	// Handle empty string
	if len(data) == 0 || string(data) == "null" {
		*a = EmptyAddress()
		return nil
	}

	return json.Unmarshal(data, a)
}

// Validation functions

func validateProvince(province string) error {
	if province == "" {
		return fmt.Errorf("province cannot be empty")
	}
	if len(province) > 100 {
		return fmt.Errorf("province cannot exceed 100 characters")
	}
	return nil
}

func validateCity(city string) error {
	if city == "" {
		return fmt.Errorf("city cannot be empty")
	}
	if len(city) > 100 {
		return fmt.Errorf("city cannot exceed 100 characters")
	}
	return nil
}

func validateDistrict(district string) error {
	// District is optional, but if provided must be reasonable length
	if len(district) > 100 {
		return fmt.Errorf("district cannot exceed 100 characters")
	}
	return nil
}

func validateDetail(detail string) error {
	if detail == "" {
		return fmt.Errorf("detail address cannot be empty")
	}
	if len(detail) > 500 {
		return fmt.Errorf("detail address cannot exceed 500 characters")
	}
	return nil
}

func validatePostalCode(postalCode string) error {
	if len(postalCode) > 20 {
		return fmt.Errorf("postal code cannot exceed 20 characters")
	}
	return nil
}

// Common Chinese province names for reference/validation
var ChineseProvinces = []string{
	"北京市", "天津市", "上海市", "重庆市",
	"河北省", "山西省", "辽宁省", "吉林省", "黑龙江省",
	"江苏省", "浙江省", "安徽省", "福建省", "江西省", "山东省",
	"河南省", "湖北省", "湖南省", "广东省", "海南省",
	"四川省", "贵州省", "云南省", "陕西省", "甘肃省", "青海省",
	"台湾省",
	"内蒙古自治区", "广西壮族自治区", "西藏自治区", "宁夏回族自治区", "新疆维吾尔自治区",
	"香港特别行政区", "澳门特别行政区",
}

// IsValidChineseProvince checks if the province is a valid Chinese province name
func IsValidChineseProvince(province string) bool {
	for _, p := range ChineseProvinces {
		if p == province || strings.TrimSuffix(p, "省") == province ||
			strings.TrimSuffix(p, "市") == province ||
			strings.TrimSuffix(p, "自治区") == province ||
			strings.TrimSuffix(p, "特别行政区") == province {
			return true
		}
	}
	return false
}

// NormalizeProvince normalizes a province name to include proper suffix
func NormalizeProvince(province string) string {
	province = strings.TrimSpace(province)

	// Already complete
	for _, p := range ChineseProvinces {
		if p == province {
			return province
		}
	}

	// Direct municipalities
	directMunicipalities := []string{"北京", "天津", "上海", "重庆"}
	for _, dm := range directMunicipalities {
		if province == dm {
			return dm + "市"
		}
	}

	// Special administrative regions
	if province == "香港" {
		return "香港特别行政区"
	}
	if province == "澳门" {
		return "澳门特别行政区"
	}

	// Autonomous regions
	autonomousRegions := map[string]string{
		"内蒙古": "内蒙古自治区",
		"广西":  "广西壮族自治区",
		"西藏":  "西藏自治区",
		"宁夏":  "宁夏回族自治区",
		"新疆":  "新疆维吾尔自治区",
	}
	if full, ok := autonomousRegions[province]; ok {
		return full
	}

	// Regular provinces
	if !strings.HasSuffix(province, "省") {
		return province + "省"
	}

	return province
}
