// Package generator provides data generation capabilities for the load generator.
package generator

import (
	"fmt"

	"github.com/brianvoe/gofakeit/v7"
)

// FakerGenerator generates fake data using the gofakeit library.
type FakerGenerator struct {
	faker  *gofakeit.Faker
	config *FakerConfig
	genFn  func(*gofakeit.Faker) any
}

// NewFakerGenerator creates a new faker generator with the given configuration.
func NewFakerGenerator(cfg *FakerConfig) (*FakerGenerator, error) {
	if cfg == nil {
		return nil, fmt.Errorf("%w: faker config is nil", ErrInvalidConfig)
	}
	if cfg.Type == "" {
		return nil, fmt.Errorf("%w: faker type is required", ErrInvalidConfig)
	}

	genFn, ok := fakerFunctions[cfg.Type]
	if !ok {
		return nil, fmt.Errorf("%w: unknown faker type: %s", ErrInvalidConfig, cfg.Type)
	}

	return &FakerGenerator{
		faker:  gofakeit.New(0), // Random seed
		config: cfg,
		genFn:  genFn,
	}, nil
}

// Generate produces a new fake value.
func (f *FakerGenerator) Generate() (any, error) {
	return f.genFn(f.faker), nil
}

// Type returns the generator type.
func (f *FakerGenerator) Type() GeneratorType {
	return TypeFaker
}

// fakerFunctions maps faker type names to generator functions.
var fakerFunctions = map[string]func(*gofakeit.Faker) any{
	// Person
	"name":       func(f *gofakeit.Faker) any { return f.Name() },
	"firstName":  func(f *gofakeit.Faker) any { return f.FirstName() },
	"lastName":   func(f *gofakeit.Faker) any { return f.LastName() },
	"namePrefix": func(f *gofakeit.Faker) any { return f.NamePrefix() },
	"nameSuffix": func(f *gofakeit.Faker) any { return f.NameSuffix() },

	// Contact
	"email": func(f *gofakeit.Faker) any { return f.Email() },
	"phone": func(f *gofakeit.Faker) any { return f.Phone() },

	// Address
	"address":     func(f *gofakeit.Faker) any { return f.Address().Address },
	"street":      func(f *gofakeit.Faker) any { return f.Street() },
	"city":        func(f *gofakeit.Faker) any { return f.City() },
	"state":       func(f *gofakeit.Faker) any { return f.State() },
	"stateAbbr":   func(f *gofakeit.Faker) any { return f.StateAbr() },
	"country":     func(f *gofakeit.Faker) any { return f.Country() },
	"countryAbbr": func(f *gofakeit.Faker) any { return f.CountryAbr() },
	"zipCode":     func(f *gofakeit.Faker) any { return f.Zip() },
	"latitude":    func(f *gofakeit.Faker) any { return f.Latitude() },
	"longitude":   func(f *gofakeit.Faker) any { return f.Longitude() },

	// Company
	"company":       func(f *gofakeit.Faker) any { return f.Company() },
	"companySuffix": func(f *gofakeit.Faker) any { return f.CompanySuffix() },
	"jobTitle":      func(f *gofakeit.Faker) any { return f.JobTitle() },
	"jobLevel":      func(f *gofakeit.Faker) any { return f.JobLevel() },

	// Internet
	"url":        func(f *gofakeit.Faker) any { return f.URL() },
	"domainName": func(f *gofakeit.Faker) any { return f.DomainName() },
	"username":   func(f *gofakeit.Faker) any { return f.Username() },
	"ipv4":       func(f *gofakeit.Faker) any { return f.IPv4Address() },
	"ipv6":       func(f *gofakeit.Faker) any { return f.IPv6Address() },
	"mac":        func(f *gofakeit.Faker) any { return f.MacAddress() },
	"userAgent":  func(f *gofakeit.Faker) any { return f.UserAgent() },

	// Payment
	"creditCard":     func(f *gofakeit.Faker) any { return f.CreditCardNumber(nil) },
	"creditCardType": func(f *gofakeit.Faker) any { return f.CreditCardType() },
	"creditCardExp":  func(f *gofakeit.Faker) any { return f.CreditCardExp() },
	"creditCardCvv":  func(f *gofakeit.Faker) any { return f.CreditCardCvv() },
	"achAccount":     func(f *gofakeit.Faker) any { return f.AchAccount() },
	"achRouting":     func(f *gofakeit.Faker) any { return f.AchRouting() },

	// Identifiers
	"uuid": func(f *gofakeit.Faker) any { return f.UUID() },
	"ssn":  func(f *gofakeit.Faker) any { return f.SSN() },

	// Text
	"word":           func(f *gofakeit.Faker) any { return f.Word() },
	"sentence":       func(f *gofakeit.Faker) any { return f.Sentence(5) },
	"paragraph":      func(f *gofakeit.Faker) any { return f.Paragraph(3, 3, 10, " ") },
	"loremWord":      func(f *gofakeit.Faker) any { return f.LoremIpsumWord() },
	"loremSentence":  func(f *gofakeit.Faker) any { return f.LoremIpsumSentence(5) },
	"loremParagraph": func(f *gofakeit.Faker) any { return f.LoremIpsumParagraph(3, 3, 10, " ") },
	"buzzWord":       func(f *gofakeit.Faker) any { return f.BuzzWord() },

	// Numbers
	"digit":  func(f *gofakeit.Faker) any { return f.Digit() },
	"number": func(f *gofakeit.Faker) any { return f.Number(1, 100) },
	"float":  func(f *gofakeit.Faker) any { return f.Float64Range(1, 100) },
	"price":  func(f *gofakeit.Faker) any { return f.Price(1, 1000) },
	"bool":   func(f *gofakeit.Faker) any { return f.Bool() },

	// Date/Time
	"date":      func(f *gofakeit.Faker) any { return f.Date().Format("2006-01-02") },
	"time":      func(f *gofakeit.Faker) any { return f.Date().Format("15:04:05") },
	"datetime":  func(f *gofakeit.Faker) any { return f.Date().Format("2006-01-02T15:04:05Z07:00") },
	"year":      func(f *gofakeit.Faker) any { return f.Year() },
	"month":     func(f *gofakeit.Faker) any { return f.Month() },
	"monthName": func(f *gofakeit.Faker) any { return f.MonthString() },
	"day":       func(f *gofakeit.Faker) any { return f.Day() },
	"weekday":   func(f *gofakeit.Faker) any { return f.WeekDay() },

	// File
	"fileExtension": func(f *gofakeit.Faker) any { return f.FileExtension() },
	"fileMimeType":  func(f *gofakeit.Faker) any { return f.FileMimeType() },

	// Color
	"color":    func(f *gofakeit.Faker) any { return f.Color() },
	"hexColor": func(f *gofakeit.Faker) any { return f.HexColor() },
	"rgbColor": func(f *gofakeit.Faker) any { return f.RGBColor() },

	// Vehicle
	"carMaker":        func(f *gofakeit.Faker) any { return f.CarMaker() },
	"carModel":        func(f *gofakeit.Faker) any { return f.CarModel() },
	"carType":         func(f *gofakeit.Faker) any { return f.CarType() },
	"carFuelType":     func(f *gofakeit.Faker) any { return f.CarFuelType() },
	"carTransmission": func(f *gofakeit.Faker) any { return f.CarTransmissionType() },

	// Food/Drink
	"fruit":     func(f *gofakeit.Faker) any { return f.Fruit() },
	"vegetable": func(f *gofakeit.Faker) any { return f.Vegetable() },
	"beer":      func(f *gofakeit.Faker) any { return f.BeerName() },

	// Animal
	"animal":     func(f *gofakeit.Faker) any { return f.Animal() },
	"animalType": func(f *gofakeit.Faker) any { return f.AnimalType() },
	"cat":        func(f *gofakeit.Faker) any { return f.Cat() },
	"dog":        func(f *gofakeit.Faker) any { return f.Dog() },

	// Product
	"productName":     func(f *gofakeit.Faker) any { return f.ProductName() },
	"productCategory": func(f *gofakeit.Faker) any { return f.ProductCategory() },
	"productFeature":  func(f *gofakeit.Faker) any { return f.ProductFeature() },

	// Currency
	"currency":     func(f *gofakeit.Faker) any { return f.Currency().Short },
	"currencyLong": func(f *gofakeit.Faker) any { return f.Currency().Long },

	// Language
	"language":     func(f *gofakeit.Faker) any { return f.Language() },
	"languageAbbr": func(f *gofakeit.Faker) any { return f.LanguageAbbreviation() },

	// Emoji
	"emoji":      func(f *gofakeit.Faker) any { return f.Emoji() },
	"emojiTag":   func(f *gofakeit.Faker) any { return f.EmojiTag() },
	"emojiAlias": func(f *gofakeit.Faker) any { return f.EmojiAlias() },
}

// SupportedFakerTypes returns all supported faker type names.
func SupportedFakerTypes() []string {
	types := make([]string, 0, len(fakerFunctions))
	for t := range fakerFunctions {
		types = append(types, t)
	}
	return types
}
