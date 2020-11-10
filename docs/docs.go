// GENERATED BY THE COMMAND ABOVE; DO NOT EDIT
// This file was generated by swaggo/swag

package docs

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/alecthomas/template"
	"github.com/swaggo/swag"
)

var doc = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{.Description}}",
        "title": "{{.Title}}",
        "contact": {},
        "license": {},
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/api/group-categories": {
            "get": {
                "security": [
                    {
                        "APIKeyAuth": []
                    }
                ],
                "description": "Gives all group categories.",
                "consumes": [
                    "application/json"
                ],
                "operationId": "GetGroupCategories",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "array",
                            "items": {
                                "type": "string"
                            }
                        }
                    }
                }
            }
        },
        "/api/groups": {
            "get": {
                "security": [
                    {
                        "APIKeyAuth": []
                    }
                ],
                "description": "Gives the groups list. It can be filtered by category",
                "consumes": [
                    "application/json"
                ],
                "operationId": "GetGroups",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Category",
                        "name": "category",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/getGroupsResponse"
                            }
                        }
                    }
                }
            },
            "post": {
                "security": [
                    {
                        "AppUserAuth": []
                    }
                ],
                "description": "Creates a group. The user must be part of urn:mace:uiuc.edu:urbana:authman:app-rokwire-service-policy-rokwire groups access. Title must be a unique. Category must be one of the categories list. Privacy can be public or private",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "operationId": "CreateGroup",
                "parameters": [
                    {
                        "description": "body data",
                        "name": "data",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/createGroupRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/createResponse"
                        }
                    }
                }
            }
        },
        "/api/user/groups": {
            "get": {
                "security": [
                    {
                        "AppUserAuth": []
                    }
                ],
                "description": "Gives the user groups.",
                "consumes": [
                    "application/json"
                ],
                "operationId": "GetUserGroups",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/getUserGroupsResponse"
                            }
                        }
                    }
                }
            }
        },
        "/version": {
            "get": {
                "description": "Gives the service version.",
                "produces": [
                    "text/plain"
                ],
                "operationId": "Version",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "createGroupRequest": {
            "type": "object",
            "required": [
                "category",
                "privacy",
                "title"
            ],
            "properties": {
                "category": {
                    "type": "string"
                },
                "creator_email": {
                    "type": "string"
                },
                "creator_name": {
                    "type": "string"
                },
                "creator_photo_url": {
                    "type": "string"
                },
                "description": {
                    "type": "string"
                },
                "privacy": {
                    "type": "string"
                },
                "tags": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    }
                },
                "title": {
                    "type": "string"
                }
            }
        },
        "createResponse": {
            "type": "object",
            "properties": {
                "inserted_id": {
                    "type": "string"
                }
            }
        },
        "getGroupsResponse": {
            "type": "object",
            "properties": {
                "category": {
                    "type": "string"
                },
                "date_created": {
                    "type": "string"
                },
                "date_updated": {
                    "type": "string"
                },
                "description": {
                    "type": "string"
                },
                "id": {
                    "type": "string"
                },
                "image_url": {
                    "type": "string"
                },
                "members": {
                    "type": "array",
                    "items": {
                        "type": "object",
                        "properties": {
                            "email": {
                                "type": "string"
                            },
                            "id": {
                                "type": "string"
                            },
                            "name": {
                                "type": "string"
                            },
                            "photo_url": {
                                "type": "string"
                            },
                            "status": {
                                "type": "string"
                            }
                        }
                    }
                },
                "members_count": {
                    "type": "integer"
                },
                "membership_questions": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    }
                },
                "privacy": {
                    "type": "string"
                },
                "tags": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    }
                },
                "title": {
                    "type": "string"
                },
                "web_url": {
                    "type": "string"
                }
            }
        },
        "getUserGroupsResponse": {
            "type": "object",
            "properties": {
                "category": {
                    "type": "string"
                },
                "date_created": {
                    "type": "string"
                },
                "date_updated": {
                    "type": "string"
                },
                "description": {
                    "type": "string"
                },
                "id": {
                    "type": "string"
                },
                "image_url": {
                    "type": "string"
                },
                "members": {
                    "type": "array",
                    "items": {
                        "type": "object",
                        "properties": {
                            "date_created": {
                                "type": "string"
                            },
                            "date_updated": {
                                "type": "string"
                            },
                            "email": {
                                "type": "string"
                            },
                            "id": {
                                "type": "string"
                            },
                            "member_answers": {
                                "type": "array",
                                "items": {
                                    "type": "object",
                                    "properties": {
                                        "answer": {
                                            "type": "string"
                                        },
                                        "question": {
                                            "type": "string"
                                        }
                                    }
                                }
                            },
                            "name": {
                                "type": "string"
                            },
                            "photo_url": {
                                "type": "string"
                            },
                            "status": {
                                "type": "string"
                            }
                        }
                    }
                },
                "members_count": {
                    "type": "integer"
                },
                "membership_questions": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    }
                },
                "privacy": {
                    "type": "string"
                },
                "tags": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    }
                },
                "title": {
                    "type": "string"
                },
                "web_url": {
                    "type": "string"
                }
            }
        }
    },
    "securityDefinitions": {
        "APIKeyAuth": {
            "type": "apiKey",
            "name": "ROKWIRE-API-KEY",
            "in": "header"
        },
        "AppUserAuth": {
            "type": "apiKey",
            "name": "Authorization",
            "in": "header (add Bearer prefix to the Authorization value)"
        }
    }
}`

type swaggerInfo struct {
	Version     string
	Host        string
	BasePath    string
	Schemes     []string
	Title       string
	Description string
}

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = swaggerInfo{
	Version:     "1.0.2",
	Host:        "localhost",
	BasePath:    "/gr",
	Schemes:     []string{"https"},
	Title:       "Rokwire Groups Building Block API",
	Description: "Rokwire Groups Building Block API Documentation.",
}

type s struct{}

func (s *s) ReadDoc() string {
	sInfo := SwaggerInfo
	sInfo.Description = strings.Replace(sInfo.Description, "\n", "\\n", -1)

	t, err := template.New("swagger_info").Funcs(template.FuncMap{
		"marshal": func(v interface{}) string {
			a, _ := json.Marshal(v)
			return string(a)
		},
	}).Parse(doc)
	if err != nil {
		return doc
	}

	var tpl bytes.Buffer
	if err := t.Execute(&tpl, sInfo); err != nil {
		return doc
	}

	return tpl.String()
}

func init() {
	swag.Register(swag.Name, &s{})
}
