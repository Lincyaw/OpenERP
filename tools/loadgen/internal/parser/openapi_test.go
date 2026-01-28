package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewParser(t *testing.T) {
	p := NewParser()
	assert.NotNil(t, p)
	assert.Equal(t, 20, p.maxRefDepth)
}

func TestParser_ParseFile_NotFound(t *testing.T) {
	p := NewParser()
	_, err := p.ParseFile("/nonexistent/file.yaml")
	assert.ErrorIs(t, err, ErrSpecNotFound)
}

func TestParser_ParseBytes_InvalidYAML(t *testing.T) {
	p := NewParser()
	_, err := p.ParseBytes([]byte("invalid: yaml: content: ["))
	assert.ErrorIs(t, err, ErrInvalidSpec)
}

func TestParser_ParseBytes_NoVersion(t *testing.T) {
	p := NewParser()
	_, err := p.ParseBytes([]byte(`
info:
  title: Test API
`))
	assert.ErrorIs(t, err, ErrInvalidSpec)
}

func TestParser_ParseBytes_OpenAPI3_Minimal(t *testing.T) {
	spec := `
openapi: "3.0.0"
info:
  title: Test API
  version: "1.0"
servers:
  - url: http://localhost:8080
paths:
  /users:
    get:
      summary: List users
      operationId: listUsers
      tags:
        - users
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                type: array
                items:
                  type: object
                  properties:
                    id:
                      type: string
                    name:
                      type: string
`
	p := NewParser()
	result, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	assert.Equal(t, "3.0.0", result.Version)
	assert.Equal(t, "Test API", result.Title)
	assert.Equal(t, "http://localhost:8080", result.Host)
	assert.Len(t, result.Endpoints, 1)

	ep := result.Endpoints[0]
	assert.Equal(t, "/users", ep.Path)
	assert.Equal(t, "GET", ep.Method)
	assert.Equal(t, "listUsers", ep.OperationID)
	assert.Equal(t, "List users", ep.Summary)
	assert.Contains(t, ep.Tags, "users")
}

func TestParser_ParseBytes_OpenAPI3_WithParameters(t *testing.T) {
	spec := `
openapi: "3.0.0"
info:
  title: Test API
  version: "1.0"
paths:
  /users/{id}:
    get:
      summary: Get user by ID
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
            format: uuid
          description: User ID
        - name: include_profile
          in: query
          required: false
          schema:
            type: boolean
            default: false
        - name: X-Request-ID
          in: header
          required: false
          schema:
            type: string
      responses:
        "200":
          description: Success
`
	p := NewParser()
	result, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	assert.Len(t, result.Endpoints, 1)
	ep := result.Endpoints[0]

	assert.Len(t, ep.InputPins, 3)

	// Path parameter
	pathParam := findInputPinByName(ep.InputPins, "id")
	require.NotNil(t, pathParam)
	assert.Equal(t, ParameterLocationPath, pathParam.Location)
	assert.Equal(t, ParameterTypeString, pathParam.Type)
	assert.Equal(t, "uuid", pathParam.Format)
	assert.True(t, pathParam.Required)
	assert.Equal(t, "User ID", pathParam.Description)

	// Query parameter
	queryParam := findInputPinByName(ep.InputPins, "include_profile")
	require.NotNil(t, queryParam)
	assert.Equal(t, ParameterLocationQuery, queryParam.Location)
	assert.Equal(t, ParameterTypeBoolean, queryParam.Type)
	assert.False(t, queryParam.Required)
	assert.Equal(t, false, queryParam.Default)

	// Header parameter
	headerParam := findInputPinByName(ep.InputPins, "X-Request-ID")
	require.NotNil(t, headerParam)
	assert.Equal(t, ParameterLocationHeader, headerParam.Location)
}

func TestParser_ParseBytes_OpenAPI3_WithRequestBody(t *testing.T) {
	spec := `
openapi: "3.0.0"
info:
  title: Test API
  version: "1.0"
components:
  schemas:
    CreateUserRequest:
      type: object
      required:
        - name
        - email
      properties:
        name:
          type: string
          description: User name
        email:
          type: string
          format: email
paths:
  /users:
    post:
      summary: Create user
      requestBody:
        required: true
        description: User to create
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateUserRequest'
      responses:
        "201":
          description: Created
`
	p := NewParser()
	result, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	assert.Len(t, result.Endpoints, 1)
	ep := result.Endpoints[0]

	assert.Len(t, ep.InputPins, 1)
	bodyParam := ep.InputPins[0]
	assert.Equal(t, ParameterLocationBody, bodyParam.Location)
	assert.True(t, bodyParam.Required)
	assert.Equal(t, "User to create", bodyParam.Description)
	assert.NotNil(t, bodyParam.Schema)
}

func TestParser_ParseBytes_OpenAPI3_WithSecurity(t *testing.T) {
	spec := `
openapi: "3.0.0"
info:
  title: Test API
  version: "1.0"
components:
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
      description: Bearer token authentication
security:
  - BearerAuth: []
paths:
  /public:
    get:
      summary: Public endpoint
      security: []
      responses:
        "200":
          description: Success
  /private:
    get:
      summary: Private endpoint
      responses:
        "200":
          description: Success
`
	p := NewParser()
	result, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	// Check security definitions
	assert.Contains(t, result.SecurityDefinitions, "BearerAuth")
	assert.Equal(t, "http", result.SecurityDefinitions["BearerAuth"].Type)
	assert.Equal(t, "bearer", result.SecurityDefinitions["BearerAuth"].Scheme)

	// Find endpoints
	var publicEp, privateEp *EndpointUnit
	for i := range result.Endpoints {
		if result.Endpoints[i].Path == "/public" {
			publicEp = &result.Endpoints[i]
		}
		if result.Endpoints[i].Path == "/private" {
			privateEp = &result.Endpoints[i]
		}
	}

	require.NotNil(t, publicEp)
	require.NotNil(t, privateEp)

	// Public endpoint should not require auth
	assert.False(t, publicEp.RequiresAuth)
	assert.Empty(t, publicEp.SecuritySchemes)

	// Private endpoint should require auth (inherits global)
	assert.True(t, privateEp.RequiresAuth)
	assert.Contains(t, privateEp.SecuritySchemes, "BearerAuth")
}

func TestParser_ParseBytes_OpenAPI3_OutputPins(t *testing.T) {
	spec := `
openapi: "3.0.0"
info:
  title: Test API
  version: "1.0"
components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: string
          format: uuid
        name:
          type: string
        profile:
          type: object
          properties:
            bio:
              type: string
            avatar_url:
              type: string
paths:
  /users/{id}:
    get:
      summary: Get user
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
`
	p := NewParser()
	result, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	assert.Len(t, result.Endpoints, 1)
	ep := result.Endpoints[0]

	// Should have output pins for id, name, and nested profile fields
	assert.NotEmpty(t, ep.OutputPins)

	// Check for id output pin
	idPin := findOutputPinByName(ep.OutputPins, "id")
	require.NotNil(t, idPin)
	assert.Equal(t, "$.id", idPin.JSONPath)
	assert.Equal(t, ParameterTypeString, idPin.Type)
	assert.Equal(t, "uuid", idPin.Format)
}

func TestParser_ParseBytes_OpenAPI3_AllOf(t *testing.T) {
	spec := `
openapi: "3.0.0"
info:
  title: Test API
  version: "1.0"
components:
  schemas:
    BaseResponse:
      type: object
      properties:
        success:
          type: boolean
        message:
          type: string
    DataWrapper:
      type: object
      properties:
        data:
          type: object
paths:
  /test:
    get:
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                allOf:
                  - $ref: '#/components/schemas/BaseResponse'
                  - $ref: '#/components/schemas/DataWrapper'
`
	p := NewParser()
	result, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	assert.Len(t, result.Endpoints, 1)
	// allOf should merge schemas
	ep := result.Endpoints[0]
	assert.NotEmpty(t, ep.OutputPins)
}

func TestParser_ParseBytes_OpenAPI3_ArrayResponse(t *testing.T) {
	spec := `
openapi: "3.0.0"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      summary: List users
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                type: object
                properties:
                  data:
                    type: array
                    items:
                      type: object
                      properties:
                        id:
                          type: string
                        name:
                          type: string
                  total:
                    type: integer
`
	p := NewParser()
	result, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	assert.Len(t, result.Endpoints, 1)
	ep := result.Endpoints[0]

	// Should have output pins including array items
	totalPin := findOutputPinByName(ep.OutputPins, "total")
	require.NotNil(t, totalPin)
	assert.Equal(t, ParameterTypeInteger, totalPin.Type)
}

func TestParser_ParseBytes_OpenAPI3_Tags(t *testing.T) {
	spec := `
openapi: "3.0.0"
info:
  title: Test API
  version: "1.0"
tags:
  - name: users
    description: User management
  - name: products
    description: Product management
paths:
  /users:
    get:
      tags:
        - users
      responses:
        "200":
          description: Success
  /products:
    get:
      tags:
        - products
      responses:
        "200":
          description: Success
`
	p := NewParser()
	result, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	// Check parsed tags
	assert.Equal(t, "User management", result.Tags["users"])
	assert.Equal(t, "Product management", result.Tags["products"])

	// Test GetEndpointsByTag
	userEndpoints := result.GetEndpointsByTag("users")
	assert.Len(t, userEndpoints, 1)
	assert.Equal(t, "/users", userEndpoints[0].Path)

	productEndpoints := result.GetEndpointsByTag("products")
	assert.Len(t, productEndpoints, 1)
	assert.Equal(t, "/products", productEndpoints[0].Path)
}

func TestParser_ParseBytes_OpenAPI3_AllMethods(t *testing.T) {
	spec := `
openapi: "3.0.0"
info:
  title: Test API
  version: "1.0"
paths:
  /resource:
    get:
      summary: List
      responses:
        "200":
          description: Success
    post:
      summary: Create
      responses:
        "201":
          description: Created
    put:
      summary: Update
      responses:
        "200":
          description: Success
    patch:
      summary: Partial update
      responses:
        "200":
          description: Success
    delete:
      summary: Delete
      responses:
        "204":
          description: Deleted
`
	p := NewParser()
	result, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	assert.Len(t, result.Endpoints, 5)

	methods := make(map[string]bool)
	for _, ep := range result.Endpoints {
		methods[ep.Method] = true
	}

	assert.True(t, methods["GET"])
	assert.True(t, methods["POST"])
	assert.True(t, methods["PUT"])
	assert.True(t, methods["PATCH"])
	assert.True(t, methods["DELETE"])

	// Test GetEndpointsByMethod
	getEndpoints := result.GetEndpointsByMethod("GET")
	assert.Len(t, getEndpoints, 1)

	postEndpoints := result.GetEndpointsByMethod("POST")
	assert.Len(t, postEndpoints, 1)
}

func TestParser_ParseBytes_OpenAPI3_SuccessStatusCodes(t *testing.T) {
	spec := `
openapi: "3.0.0"
info:
  title: Test API
  version: "1.0"
paths:
  /create:
    post:
      responses:
        "201":
          description: Created
  /delete:
    delete:
      responses:
        "204":
          description: No Content
  /list:
    get:
      responses:
        "200":
          description: OK
`
	p := NewParser()
	result, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	for _, ep := range result.Endpoints {
		switch ep.Path {
		case "/create":
			assert.Contains(t, ep.SuccessStatusCodes, 201)
		case "/delete":
			assert.Contains(t, ep.SuccessStatusCodes, 204)
		case "/list":
			assert.Contains(t, ep.SuccessStatusCodes, 200)
		}
	}
}

func TestParser_ParseBytes_CircularReference(t *testing.T) {
	// Test that circular $ref doesn't cause infinite recursion
	spec := `
openapi: "3.0.0"
info:
  title: Circular Ref Test
  version: "1.0"
components:
  schemas:
    Node:
      type: object
      properties:
        id:
          type: string
        children:
          type: array
          items:
            $ref: '#/components/schemas/Node'
        parent:
          $ref: '#/components/schemas/Node'
paths:
  /nodes:
    get:
      summary: List nodes
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Node'
`
	p := NewParser()
	result, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	// Should not hang or crash
	assert.NotNil(t, result)
	assert.Equal(t, "Circular Ref Test", result.Title)
	assert.Len(t, result.Endpoints, 1)
}

func TestParser_ParseBytes_DeeplyNestedSchema(t *testing.T) {
	// Test maxRefDepth protection
	spec := `
openapi: "3.0.0"
info:
  title: Deep Nesting Test
  version: "1.0"
components:
  schemas:
    Level1:
      type: object
      properties:
        nested:
          $ref: '#/components/schemas/Level2'
    Level2:
      type: object
      properties:
        nested:
          $ref: '#/components/schemas/Level3'
    Level3:
      type: object
      properties:
        value:
          type: string
paths:
  /deep:
    get:
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Level1'
`
	p := NewParser()
	result, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	assert.NotNil(t, result)
	assert.Len(t, result.Endpoints, 1)
}

func TestParser_ParseBytes_OpenAPI3_Components(t *testing.T) {
	spec := `
openapi: "3.0.0"
info:
  title: Test API
  version: "1.0"
components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
      description: JWT Authorization header
  schemas:
    User:
      type: object
      properties:
        id:
          type: string
paths:
  /users:
    get:
      security:
        - bearerAuth: []
      responses:
        "200":
          description: Success
`
	p := NewParser()
	result, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	// Check security definitions
	assert.Contains(t, result.SecurityDefinitions, "bearerAuth")
	assert.Equal(t, "http", result.SecurityDefinitions["bearerAuth"].Type)

	// Check endpoint security
	assert.Len(t, result.Endpoints, 1)
	assert.True(t, result.Endpoints[0].RequiresAuth)
}

func TestParser_ParseBytes_Deprecated(t *testing.T) {
	spec := `
openapi: "3.0.0"
info:
  title: Test API
  version: "1.0"
paths:
  /old-endpoint:
    get:
      summary: Old endpoint
      deprecated: true
      responses:
        "200":
          description: Success
  /new-endpoint:
    get:
      summary: New endpoint
      responses:
        "200":
          description: Success
`
	p := NewParser()
	result, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	for _, ep := range result.Endpoints {
		if ep.Path == "/old-endpoint" {
			assert.True(t, ep.Deprecated)
		}
		if ep.Path == "/new-endpoint" {
			assert.False(t, ep.Deprecated)
		}
	}
}

func TestParser_GenerateEndpointName(t *testing.T) {
	tests := []struct {
		method   string
		path     string
		expected string
	}{
		{"GET", "/users", "get.users"},
		{"POST", "/users", "post.users"},
		{"GET", "/users/{id}", "get.users"},
		{"GET", "/users/{user_id}/posts/{post_id}", "get.users.posts"},
		{"GET", "/", "get.root"},
		{"DELETE", "/catalog/products/{id}", "delete.catalog.products"},
	}

	for _, tt := range tests {
		t.Run(tt.method+"_"+tt.path, func(t *testing.T) {
			result := generateEndpointName(tt.method, tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParser_MergeParameters(t *testing.T) {
	pathParams := []InputPin{
		{Name: "id", Location: ParameterLocationPath, Type: ParameterTypeString},
		{Name: "version", Location: ParameterLocationQuery, Type: ParameterTypeString},
	}

	opParams := []InputPin{
		{Name: "version", Location: ParameterLocationQuery, Type: ParameterTypeInteger}, // Override
		{Name: "filter", Location: ParameterLocationQuery, Type: ParameterTypeString},
	}

	merged := mergeParameters(pathParams, opParams)

	assert.Len(t, merged, 3)

	// Check that operation param overrides path param
	versionParam := findInputPinByName(merged, "version")
	require.NotNil(t, versionParam)
	assert.Equal(t, ParameterTypeInteger, versionParam.Type)
}

func TestOpenAPISpec_Summary(t *testing.T) {
	spec := &OpenAPISpec{
		Title:   "Test API",
		Version: "1.0",
		Endpoints: []EndpointUnit{
			{Path: "/users", Method: "GET", Tags: []string{"users"}, RequiresAuth: true},
			{Path: "/users", Method: "POST", Tags: []string{"users"}, RequiresAuth: true},
			{Path: "/products", Method: "GET", Tags: []string{"products"}, RequiresAuth: false},
		},
	}

	summary := spec.Summary()
	assert.Contains(t, summary, "Test API")
	assert.Contains(t, summary, "Total Endpoints: 3")
	assert.Contains(t, summary, "GET: 2")
	assert.Contains(t, summary, "POST: 1")
	assert.Contains(t, summary, "users: 2")
	assert.Contains(t, summary, "products: 1")
}

func TestOpenAPISpec_GetAuthenticatedEndpoints(t *testing.T) {
	spec := &OpenAPISpec{
		Endpoints: []EndpointUnit{
			{Path: "/public", RequiresAuth: false},
			{Path: "/private1", RequiresAuth: true},
			{Path: "/private2", RequiresAuth: true},
		},
	}

	auth := spec.GetAuthenticatedEndpoints()
	assert.Len(t, auth, 2)

	public := spec.GetPublicEndpoints()
	assert.Len(t, public, 1)
}

// Integration test with actual ERP swagger.yaml
func TestParser_ParseFile_ERPSwagger(t *testing.T) {
	// Path relative to test execution
	swaggerPath := "../../../../backend/docs/swagger.yaml"

	// Check if file exists
	if _, err := os.Stat(swaggerPath); os.IsNotExist(err) {
		// Try absolute path from workspace root
		swaggerPath = filepath.Join(os.Getenv("PWD"), "backend/docs/swagger.yaml")
		if _, err := os.Stat(swaggerPath); os.IsNotExist(err) {
			t.Skip("ERP swagger.yaml not found, skipping integration test")
		}
	}

	p := NewParser()
	result, err := p.ParseFile(swaggerPath)
	require.NoError(t, err)

	// Basic assertions about ERP API
	assert.True(t, result.Version == "3.1.0" || result.Version == "3.0.0", "Expected OpenAPI 3.x version")
	assert.Equal(t, "ERP Backend API", result.Title)

	// Should have many endpoints
	assert.Greater(t, len(result.Endpoints), 50, "ERP API should have more than 50 endpoints")

	// Should have common tags
	tagFound := false
	for _, ep := range result.Endpoints {
		if contains(ep.Tags, "auth") || contains(ep.Tags, "catalog") || contains(ep.Tags, "inventory") {
			tagFound = true
			break
		}
	}
	assert.True(t, tagFound, "Should have common ERP tags")

	// Check for authentication
	assert.NotEmpty(t, result.SecurityDefinitions, "Should have security definitions")

	// Check auth endpoints exist
	var loginEndpoint *EndpointUnit
	for i := range result.Endpoints {
		if result.Endpoints[i].Path == "/auth/login" && result.Endpoints[i].Method == "POST" {
			loginEndpoint = &result.Endpoints[i]
			break
		}
	}
	require.NotNil(t, loginEndpoint, "Should have login endpoint")
	assert.False(t, loginEndpoint.RequiresAuth, "Login endpoint should not require auth")

	// Check a protected endpoint
	protectedEndpoints := result.GetAuthenticatedEndpoints()
	assert.NotEmpty(t, protectedEndpoints, "Should have protected endpoints")

	// Print summary for debugging
	t.Logf("\n%s", result.Summary())
}

func TestParser_ParseFile_ERPSwagger_EndpointDetails(t *testing.T) {
	swaggerPath := "../../../../backend/docs/swagger.yaml"
	if _, err := os.Stat(swaggerPath); os.IsNotExist(err) {
		swaggerPath = filepath.Join(os.Getenv("PWD"), "backend/docs/swagger.yaml")
		if _, err := os.Stat(swaggerPath); os.IsNotExist(err) {
			t.Skip("ERP swagger.yaml not found, skipping integration test")
		}
	}

	p := NewParser()
	result, err := p.ParseFile(swaggerPath)
	require.NoError(t, err)

	// Find an endpoint with path parameters
	var productEndpoint *EndpointUnit
	for i := range result.Endpoints {
		if result.Endpoints[i].Path == "/catalog/products/{id}" && result.Endpoints[i].Method == "GET" {
			productEndpoint = &result.Endpoints[i]
			break
		}
	}

	if productEndpoint != nil {
		// Should have path parameter
		idParam := findInputPinByName(productEndpoint.InputPins, "id")
		assert.NotNil(t, idParam, "Should have id path parameter")
		if idParam != nil {
			assert.Equal(t, ParameterLocationPath, idParam.Location)
			assert.True(t, idParam.Required)
		}

		// Should have output pins
		assert.NotEmpty(t, productEndpoint.OutputPins, "Should have output pins")
	}

	// Find POST endpoint with body
	var createProductEndpoint *EndpointUnit
	for i := range result.Endpoints {
		if result.Endpoints[i].Path == "/catalog/products" && result.Endpoints[i].Method == "POST" {
			createProductEndpoint = &result.Endpoints[i]
			break
		}
	}

	if createProductEndpoint != nil {
		// Should have body parameter
		bodyParam := findInputPinByLocation(createProductEndpoint.InputPins, ParameterLocationBody)
		assert.NotNil(t, bodyParam, "Create product should have body parameter")
	}
}

// Helper functions
func findInputPinByName(pins []InputPin, name string) *InputPin {
	for i := range pins {
		if pins[i].Name == name {
			return &pins[i]
		}
	}
	return nil
}

func findInputPinByLocation(pins []InputPin, location ParameterLocation) *InputPin {
	for i := range pins {
		if pins[i].Location == location {
			return &pins[i]
		}
	}
	return nil
}

func findOutputPinByName(pins []OutputPin, name string) *OutputPin {
	for i := range pins {
		if pins[i].Name == name {
			return &pins[i]
		}
	}
	return nil
}

func findOutputPinByJSONPath(pins []OutputPin, path string) *OutputPin {
	for i := range pins {
		if pins[i].JSONPath == path {
			return &pins[i]
		}
	}
	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
