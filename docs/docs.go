package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "swagger": "2.0",
    "info": {
        "description": "TaskMaster API Documentation",
        "title": "TaskMaster API",
        "version": "1.0"
    },
    "host": "localhost:8080",
    "basePath": "/api/v1",
    "schemes": ["http"],
    "paths": {
        "/health": {
            "get": {
                "tags": ["Health"],
                "summary": "Health Check",
                "description": "Check if server is running",
                "responses": {
                    "200": {
                        "description": "Server is healthy"
                    }
                }
            }
        },
        "/api/v1/auth/login": {
            "post": {
                "tags": ["Authentication"],
                "summary": "User Login",
                "description": "Login with email and password",
                "consumes": ["application/json"],
                "produces": ["application/json"],
                "parameters": [
                    {
                        "in": "body",
                        "name": "credentials",
                        "description": "Login credentials",
                        "required": true,
                        "schema": {
                            "type": "object",
                            "properties": {
                                "email": {
                                    "type": "string",
                                    "example": "admin@taskmaster.dev"
                                },
                                "password": {
                                    "type": "string",
                                    "example": "admin123"
                                }
                            }
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Login successful"
                    },
                    "401": {
                        "description": "Invalid credentials"
                    }
                }
            }
        },
        "/api/v1/auth/register": {
            "post": {
                "tags": ["Authentication"],
                "summary": "User Registration",
                "description": "Register a new user",
                "consumes": ["application/json"],
                "produces": ["application/json"],
                "parameters": [
                    {
                        "in": "body",
                        "name": "user",
                        "description": "User registration data",
                        "required": true,
                        "schema": {
                            "type": "object",
                            "properties": {
                                "email": {
                                    "type": "string",
                                    "example": "user@example.com"
                                },
                                "username": {
                                    "type": "string", 
                                    "example": "testuser"
                                },
                                "password": {
                                    "type": "string",
                                    "example": "password123"
                                },
                                "role": {
                                    "type": "string",
                                    "enum": ["admin", "manager", "developer", "tester", "viewer"],
                                    "example": "developer"
                                }
                            }
                        }
                    }
                ],
                "responses": {
                    "201": {
                        "description": "User created successfully"
                    },
                    "400": {
                        "description": "Invalid input"
                    }
                }
            }
        },
        "/api/v1/users/me": {
            "get": {
                "tags": ["Users"],
                "summary": "Get Current User",
                "description": "Get information about the currently authenticated user",
                "produces": ["application/json"],
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "responses": {
                    "200": {
                        "description": "User information"
                    },
                    "401": {
                        "description": "Unauthorized"
                    }
                }
            }
        }
    },
    "securityDefinitions": {
        "BearerAuth": {
            "type": "apiKey",
            "name": "Authorization",
            "in": "header",
            "description": "Type 'Bearer' followed by a space and JWT token"
        }
    }
}`

var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "localhost:8080",
	BasePath:         "/api/v1",
	Schemes:          []string{"http"},
	Title:            "TaskMaster API",
	Description:      "TaskMaster API Documentation",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
