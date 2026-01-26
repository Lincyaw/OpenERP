package valueobject

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAddress(t *testing.T) {
	tests := []struct {
		name        string
		province    string
		city        string
		district    string
		detail      string
		opts        []AddressOption
		wantErr     bool
		errContains string
	}{
		{
			name:     "valid address with required fields",
			province: "广东省",
			city:     "深圳市",
			district: "南山区",
			detail:   "科技园南路123号",
			wantErr:  false,
		},
		{
			name:     "valid address without district",
			province: "北京市",
			city:     "北京市",
			district: "",
			detail:   "朝阳区建国门外大街1号",
			wantErr:  false,
		},
		{
			name:     "valid address with postal code",
			province: "上海市",
			city:     "上海市",
			district: "浦东新区",
			detail:   "陆家嘴环路1000号",
			opts:     []AddressOption{WithPostalCode("200120")},
			wantErr:  false,
		},
		{
			name:     "valid address with country",
			province: "广东省",
			city:     "广州市",
			district: "天河区",
			detail:   "珠江新城华夏路30号",
			opts:     []AddressOption{WithCountry("中国")},
			wantErr:  false,
		},
		{
			name:        "empty province",
			province:    "",
			city:        "深圳市",
			district:    "南山区",
			detail:      "科技园",
			wantErr:     true,
			errContains: "province cannot be empty",
		},
		{
			name:        "empty city",
			province:    "广东省",
			city:        "",
			district:    "南山区",
			detail:      "科技园",
			wantErr:     true,
			errContains: "city cannot be empty",
		},
		{
			name:        "empty detail",
			province:    "广东省",
			city:        "深圳市",
			district:    "南山区",
			detail:      "",
			wantErr:     true,
			errContains: "detail address cannot be empty",
		},
		{
			name:     "whitespace trimmed",
			province: "  广东省  ",
			city:     "  深圳市  ",
			district: "  南山区  ",
			detail:   "  科技园南路123号  ",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, err := NewAddress(tt.province, tt.city, tt.district, tt.detail, tt.opts...)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, strings.TrimSpace(tt.province), addr.Province())
			assert.Equal(t, strings.TrimSpace(tt.city), addr.City())
			assert.Equal(t, strings.TrimSpace(tt.district), addr.District())
			assert.Equal(t, strings.TrimSpace(tt.detail), addr.Detail())
		})
	}
}

func TestNewAddressWithPostalCode(t *testing.T) {
	addr, err := NewAddressWithPostalCode("广东省", "深圳市", "南山区", "科技园", "518000")
	require.NoError(t, err)
	assert.Equal(t, "518000", addr.PostalCode())
}

func TestNewAddressFull(t *testing.T) {
	addr, err := NewAddressFull("广东省", "深圳市", "南山区", "科技园", "518000", "中国")
	require.NoError(t, err)
	assert.Equal(t, "广东省", addr.Province())
	assert.Equal(t, "深圳市", addr.City())
	assert.Equal(t, "南山区", addr.District())
	assert.Equal(t, "科技园", addr.Detail())
	assert.Equal(t, "518000", addr.PostalCode())
	assert.Equal(t, "中国", addr.Country())
}

func TestMustNewAddress(t *testing.T) {
	t.Run("valid address", func(t *testing.T) {
		addr := MustNewAddress("广东省", "深圳市", "南山区", "科技园")
		assert.Equal(t, "广东省", addr.Province())
	})

	t.Run("panic on invalid address", func(t *testing.T) {
		assert.Panics(t, func() {
			MustNewAddress("", "深圳市", "南山区", "科技园")
		})
	})
}

func TestEmptyAddress(t *testing.T) {
	addr := EmptyAddress()
	assert.True(t, addr.IsEmpty())
	assert.Equal(t, "", addr.Province())
	assert.Equal(t, "", addr.City())
	assert.Equal(t, "", addr.District())
	assert.Equal(t, "", addr.Detail())
}

func TestAddress_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		addr     Address
		expected bool
	}{
		{
			name:     "empty address",
			addr:     EmptyAddress(),
			expected: true,
		},
		{
			name:     "non-empty address",
			addr:     MustNewAddress("广东省", "深圳市", "南山区", "科技园"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.addr.IsEmpty())
		})
	}
}

func TestAddress_FullAddress(t *testing.T) {
	tests := []struct {
		name     string
		addr     Address
		expected string
	}{
		{
			name:     "complete address",
			addr:     MustNewAddress("广东省", "深圳市", "南山区", "科技园南路123号", WithPostalCode("518000"), WithCountry("中国")),
			expected: "中国 广东省 深圳市 南山区 科技园南路123号 518000",
		},
		{
			name:     "address without postal code",
			addr:     MustNewAddress("广东省", "深圳市", "南山区", "科技园南路123号"),
			expected: "中国 广东省 深圳市 南山区 科技园南路123号",
		},
		{
			name:     "empty address",
			addr:     EmptyAddress(),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.addr.FullAddress())
		})
	}
}

func TestAddress_ShortAddress(t *testing.T) {
	addr := MustNewAddress("广东省", "深圳市", "南山区", "科技园南路123号")
	expected := "深圳市 南山区 科技园南路123号"
	assert.Equal(t, expected, addr.ShortAddress())
}

func TestAddress_RegionAddress(t *testing.T) {
	addr := MustNewAddress("广东省", "深圳市", "南山区", "科技园南路123号")
	expected := "广东省 深圳市 南山区"
	assert.Equal(t, expected, addr.RegionAddress())
}

func TestAddress_ChineseFullAddress(t *testing.T) {
	addr := MustNewAddress("广东省", "深圳市", "南山区", "科技园南路123号")
	expected := "广东省深圳市南山区科技园南路123号"
	assert.Equal(t, expected, addr.ChineseFullAddress())
}

func TestAddress_String(t *testing.T) {
	addr := MustNewAddress("广东省", "深圳市", "南山区", "科技园南路123号")
	// String() should return the same as FullAddress()
	assert.Equal(t, addr.FullAddress(), addr.String())
}

func TestAddress_Equals(t *testing.T) {
	addr1 := MustNewAddress("广东省", "深圳市", "南山区", "科技园", WithPostalCode("518000"))
	addr2 := MustNewAddress("广东省", "深圳市", "南山区", "科技园", WithPostalCode("518000"))
	addr3 := MustNewAddress("广东省", "深圳市", "南山区", "科技园", WithPostalCode("518001"))
	addr4 := MustNewAddress("广东省", "深圳市", "福田区", "科技园")

	assert.True(t, addr1.Equals(addr2))
	assert.False(t, addr1.Equals(addr3)) // Different postal code
	assert.False(t, addr1.Equals(addr4)) // Different district
}

func TestAddress_SameRegion(t *testing.T) {
	addr1 := MustNewAddress("广东省", "深圳市", "南山区", "科技园南路123号")
	addr2 := MustNewAddress("广东省", "深圳市", "南山区", "科技园北路456号")
	addr3 := MustNewAddress("广东省", "深圳市", "福田区", "福华三路")

	assert.True(t, addr1.SameRegion(addr2))  // Same province, city, district
	assert.False(t, addr1.SameRegion(addr3)) // Different district
}

func TestAddress_SameCity(t *testing.T) {
	addr1 := MustNewAddress("广东省", "深圳市", "南山区", "科技园")
	addr2 := MustNewAddress("广东省", "深圳市", "福田区", "福华三路")
	addr3 := MustNewAddress("广东省", "广州市", "天河区", "珠江新城")

	assert.True(t, addr1.SameCity(addr2))
	assert.False(t, addr1.SameCity(addr3))
}

func TestAddress_SameProvince(t *testing.T) {
	addr1 := MustNewAddress("广东省", "深圳市", "南山区", "科技园")
	addr2 := MustNewAddress("广东省", "广州市", "天河区", "珠江新城")
	addr3 := MustNewAddress("北京市", "北京市", "朝阳区", "建国门外大街")

	assert.True(t, addr1.SameProvince(addr2))
	assert.False(t, addr1.SameProvince(addr3))
}

func TestAddress_With_Methods(t *testing.T) {
	original := MustNewAddress("广东省", "深圳市", "南山区", "科技园", WithPostalCode("518000"))

	t.Run("WithProvince", func(t *testing.T) {
		updated, err := original.WithProvince("北京市")
		require.NoError(t, err)
		assert.Equal(t, "北京市", updated.Province())
		assert.Equal(t, "深圳市", updated.City()) // Unchanged
		// Original should be unchanged (immutability)
		assert.Equal(t, "广东省", original.Province())
	})

	t.Run("WithCity", func(t *testing.T) {
		updated, err := original.WithCity("广州市")
		require.NoError(t, err)
		assert.Equal(t, "广州市", updated.City())
		assert.Equal(t, "广东省", updated.Province()) // Unchanged
	})

	t.Run("WithDistrict", func(t *testing.T) {
		updated, err := original.WithDistrict("福田区")
		require.NoError(t, err)
		assert.Equal(t, "福田区", updated.District())
	})

	t.Run("WithDetail", func(t *testing.T) {
		updated, err := original.WithDetail("新地址123号")
		require.NoError(t, err)
		assert.Equal(t, "新地址123号", updated.Detail())
	})

	t.Run("WithUpdatedPostalCode", func(t *testing.T) {
		updated, err := original.WithUpdatedPostalCode("518001")
		require.NoError(t, err)
		assert.Equal(t, "518001", updated.PostalCode())
	})

	t.Run("WithUpdatedCountry", func(t *testing.T) {
		updated, err := original.WithUpdatedCountry("USA")
		require.NoError(t, err)
		assert.Equal(t, "USA", updated.Country())
	})
}

func TestAddress_JSONMarshalUnmarshal(t *testing.T) {
	t.Run("marshal and unmarshal", func(t *testing.T) {
		original := MustNewAddress("广东省", "深圳市", "南山区", "科技园", WithPostalCode("518000"), WithCountry("中国"))

		data, err := json.Marshal(original)
		require.NoError(t, err)

		var unmarshaled Address
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.True(t, original.Equals(unmarshaled))
	})

	t.Run("unmarshal empty address", func(t *testing.T) {
		data := `{"province":"","city":"","district":"","detail":""}`
		var addr Address
		err := json.Unmarshal([]byte(data), &addr)
		require.NoError(t, err)
		assert.True(t, addr.IsEmpty())
	})

	t.Run("json structure", func(t *testing.T) {
		addr := MustNewAddress("广东省", "深圳市", "南山区", "科技园", WithPostalCode("518000"))
		data, err := json.Marshal(addr)
		require.NoError(t, err)

		var m map[string]any
		err = json.Unmarshal(data, &m)
		require.NoError(t, err)

		assert.Equal(t, "广东省", m["province"])
		assert.Equal(t, "深圳市", m["city"])
		assert.Equal(t, "南山区", m["district"])
		assert.Equal(t, "科技园", m["detail"])
		assert.Equal(t, "518000", m["postalCode"])
	})
}

func TestAddressDTO_Conversion(t *testing.T) {
	t.Run("to DTO and back", func(t *testing.T) {
		original := MustNewAddress("广东省", "深圳市", "南山区", "科技园", WithPostalCode("518000"), WithCountry("中国"))

		dto := original.ToDTO()
		assert.Equal(t, "广东省", dto.Province)
		assert.Equal(t, "深圳市", dto.City)
		assert.Equal(t, "南山区", dto.District)
		assert.Equal(t, "科技园", dto.Detail)
		assert.Equal(t, "518000", dto.PostalCode)
		assert.Equal(t, "中国", dto.Country)

		converted, err := dto.ToAddress()
		require.NoError(t, err)
		assert.True(t, original.Equals(converted))
	})

	t.Run("empty DTO to empty address", func(t *testing.T) {
		dto := AddressDTO{}
		addr, err := dto.ToAddress()
		require.NoError(t, err)
		assert.True(t, addr.IsEmpty())
	})

	t.Run("MustToAddress", func(t *testing.T) {
		dto := AddressDTO{
			Province: "广东省",
			City:     "深圳市",
			District: "南山区",
			Detail:   "科技园",
		}
		addr := dto.MustToAddress()
		assert.Equal(t, "广东省", addr.Province())
	})

	t.Run("MustToAddress panics on invalid", func(t *testing.T) {
		dto := AddressDTO{
			Province: "",
			City:     "深圳市",
			District: "南山区",
			Detail:   "科技园",
		}
		assert.Panics(t, func() {
			dto.MustToAddress()
		})
	})
}

func TestAddress_ValueScan(t *testing.T) {
	t.Run("value and scan", func(t *testing.T) {
		original := MustNewAddress("广东省", "深圳市", "南山区", "科技园")

		value, err := original.Value()
		require.NoError(t, err)
		require.NotNil(t, value)

		var scanned Address
		err = scanned.Scan(value)
		require.NoError(t, err)
		assert.True(t, original.Equals(scanned))
	})

	t.Run("scan nil", func(t *testing.T) {
		var addr Address
		err := addr.Scan(nil)
		require.NoError(t, err)
		assert.True(t, addr.IsEmpty())
	})

	t.Run("scan empty string", func(t *testing.T) {
		var addr Address
		err := addr.Scan("")
		require.NoError(t, err)
		assert.True(t, addr.IsEmpty())
	})

	t.Run("scan null string", func(t *testing.T) {
		var addr Address
		err := addr.Scan("null")
		require.NoError(t, err)
		assert.True(t, addr.IsEmpty())
	})

	t.Run("scan bytes", func(t *testing.T) {
		original := MustNewAddress("广东省", "深圳市", "南山区", "科技园")
		data, _ := json.Marshal(original)

		var scanned Address
		err := scanned.Scan(data)
		require.NoError(t, err)
		assert.True(t, original.Equals(scanned))
	})

	t.Run("empty address value is nil", func(t *testing.T) {
		addr := EmptyAddress()
		value, err := addr.Value()
		require.NoError(t, err)
		assert.Nil(t, value)
	})
}

func TestAddress_ValidationLimits(t *testing.T) {
	longString := make([]byte, 200)
	for i := range longString {
		longString[i] = 'a'
	}
	veryLongString := string(longString)

	t.Run("province too long", func(t *testing.T) {
		_, err := NewAddress(veryLongString, "深圳市", "南山区", "科技园")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "province cannot exceed 100 characters")
	})

	t.Run("city too long", func(t *testing.T) {
		_, err := NewAddress("广东省", veryLongString, "南山区", "科技园")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "city cannot exceed 100 characters")
	})

	t.Run("district too long", func(t *testing.T) {
		_, err := NewAddress("广东省", "深圳市", veryLongString, "科技园")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "district cannot exceed 100 characters")
	})

	t.Run("detail too long", func(t *testing.T) {
		veryLongDetail := make([]byte, 600)
		for i := range veryLongDetail {
			veryLongDetail[i] = 'a'
		}
		_, err := NewAddress("广东省", "深圳市", "南山区", string(veryLongDetail))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "detail address cannot exceed 500 characters")
	})

	t.Run("postal code too long", func(t *testing.T) {
		_, err := NewAddress("广东省", "深圳市", "南山区", "科技园", WithPostalCode("12345678901234567890123"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "postal code cannot exceed 20 characters")
	})
}

func TestIsValidChineseProvince(t *testing.T) {
	tests := []struct {
		province string
		valid    bool
	}{
		{"广东省", true},
		{"广东", true},
		{"北京市", true},
		{"北京", true},
		{"香港特别行政区", true},
		{"香港", true},
		{"内蒙古自治区", true},
		{"内蒙古", true},
		{"InvalidProvince", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.province, func(t *testing.T) {
			assert.Equal(t, tt.valid, IsValidChineseProvince(tt.province))
		})
	}
}

func TestNormalizeProvince(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"广东", "广东省"},
		{"广东省", "广东省"},
		{"北京", "北京市"},
		{"北京市", "北京市"},
		{"上海", "上海市"},
		{"香港", "香港特别行政区"},
		{"澳门", "澳门特别行政区"},
		{"内蒙古", "内蒙古自治区"},
		{"广西", "广西壮族自治区"},
		{"西藏", "西藏自治区"},
		{"宁夏", "宁夏回族自治区"},
		{"新疆", "新疆维吾尔自治区"},
		{"  广东  ", "广东省"}, // Trimmed
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, NormalizeProvince(tt.input))
		})
	}
}

func TestParseAddressFromJSON(t *testing.T) {
	t.Run("valid address JSON", func(t *testing.T) {
		data := []byte(`{"province":"北京市","city":"北京市","district":"海淀区","detail":"中关村科技园","postalCode":"100080","country":"中国"}`)
		addr, err := ParseAddressFromJSON(data)
		require.NoError(t, err)
		assert.Equal(t, "北京市", addr.Province())
		assert.Equal(t, "北京市", addr.City())
		assert.Equal(t, "海淀区", addr.District())
		assert.Equal(t, "中关村科技园", addr.Detail())
		assert.Equal(t, "100080", addr.PostalCode())
		assert.Equal(t, "中国", addr.Country())
	})

	t.Run("empty address JSON returns empty address", func(t *testing.T) {
		data := []byte(`{"province":"","city":"","district":"","detail":""}`)
		addr, err := ParseAddressFromJSON(data)
		require.NoError(t, err)
		assert.True(t, addr.IsEmpty())
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		data := []byte(`{invalid json}`)
		_, err := ParseAddressFromJSON(data)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse address JSON")
	})

	t.Run("invalid address data returns validation error", func(t *testing.T) {
		data := []byte(`{"province":"","city":"北京市","district":"海淀区","detail":"中关村"}`)
		_, err := ParseAddressFromJSON(data)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "province cannot be empty")
	})

	t.Run("immutability - returns new value", func(t *testing.T) {
		data := []byte(`{"province":"广东省","city":"深圳市","district":"南山区","detail":"科技园"}`)
		addr1, err := ParseAddressFromJSON(data)
		require.NoError(t, err)
		addr2, err := ParseAddressFromJSON(data)
		require.NoError(t, err)

		// Both addresses should be equal but independent
		assert.True(t, addr1.Equals(addr2))
	})
}
