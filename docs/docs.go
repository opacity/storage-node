// GENERATED BY THE COMMAND ABOVE; DO NOT EDIT
// This file was generated by swaggo/swag at
// 2019-05-13 20:22:10.9111102 -0400 EDT m=+0.168016801

package docs

import (
	"bytes"

	"github.com/alecthomas/template"
	"github.com/swaggo/swag"
)

var doc = `{
    "swagger": "2.0",
    "info": {
        "description": "Opacity backend for file storage.",
        "title": "Storage Node",
        "termsOfService": "https://opacity.io/terms-of-service",
        "contact": {
            "name": "Opacity Staff",
            "url": "https://telegram.me/opacitystorage"
        },
        "license": {
            "name": "GNU GENERAL PUBLIC LICENSE"
        },
        "version": "1.0"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/admin/user_stats": {
            "get": {
                "description": "get statistics",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "summary": "get statistics",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "object",
                            "$ref": "#/definitions/routes.userStatsRes"
                        }
                    }
                }
            }
        },
        "/api/v1/account-data": {
            "post": {
                "description": "check the payment status of an account\nrequestBody should be a stringified version of (values are just examples):\n{\n\"timestamp\": 1557346389\n}",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "summary": "check the payment status of an account",
                "parameters": [
                    {
                        "description": "account payment status check object",
                        "name": "getAccountDataReq",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "type": "object",
                            "$ref": "#/definitions/routes.getAccountDataReq"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "object",
                            "$ref": "#/definitions/routes.accountPaidRes"
                        }
                    },
                    "400": {
                        "description": "bad request, unable to parse request body: (with the error)",
                        "schema": {
                            "type": "string"
                        }
                    },
                    "404": {
                        "description": "no account with that id: (with your accountID)",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            }
        },
        "/api/v1/accounts": {
            "post": {
                "description": "create an account\nrequestBody should be a stringified version of (values are just examples):\n{\n\"storageLimit\": 100,\n\"durationInMonths\": 12,\n\"metadataKey\": \"a 64-char hex string created deterministically from your account handle or private key\",\n}",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "summary": "create an account",
                "parameters": [
                    {
                        "description": "account creation object",
                        "name": "accountCreateReq",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "type": "object",
                            "$ref": "#/definitions/routes.accountCreateReq"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "object",
                            "$ref": "#/definitions/routes.accountCreateRes"
                        }
                    },
                    "400": {
                        "description": "bad request, unable to parse request body: (with the error)",
                        "schema": {
                            "type": "string"
                        }
                    },
                    "503": {
                        "description": "error encrypting private key: (with the error)",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            }
        },
        "/api/v1/download": {
            "get": {
                "description": "download a file\nrequestBody should be a stringified version of (values are just examples):\n{\n\"fileID\": \"the handle of the file\",\n}",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "summary": "download a file",
                "parameters": [
                    {
                        "description": "download object",
                        "name": "downloadFileReq",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "type": "object",
                            "$ref": "#/definitions/routes.downloadFileReq"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "object",
                            "$ref": "#/definitions/routes.downloadFileRes"
                        }
                    },
                    "400": {
                        "description": "bad request, unable to parse request body: (with the error)",
                        "schema": {
                            "type": "string"
                        }
                    },
                    "404": {
                        "description": "such data does not exist",
                        "schema": {
                            "type": "string"
                        }
                    },
                    "500": {
                        "description": "some information about the internal error",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            }
        },
        "/api/v1/file": {
            "delete": {
                "description": "delete a file\nrequestBody should be a stringified version of (values are just examples):\n{\n\"fileID\": \"the handle of the file\",\n}",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "summary": "delete a file",
                "parameters": [
                    {
                        "description": "file deletion object",
                        "name": "deleteFileReq",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "type": "object",
                            "$ref": "#/definitions/routes.deleteFileReq"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "object",
                            "$ref": "#/definitions/routes.deleteFileRes"
                        }
                    },
                    "400": {
                        "description": "bad request, unable to parse request body: (with the error)",
                        "schema": {
                            "type": "string"
                        }
                    },
                    "500": {
                        "description": "some information about the internal error",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            }
        },
        "/api/v1/init-upload": {
            "post": {
                "description": "start an upload\nrequestBody should be a stringified version of (values are just examples):\n{\n\"fileHandle\": \"a deterministically created file handle\",\n\"fileSizeInByte\": \"200000000000006\",\n\"endIndex\": 2\n}",
                "consumes": [
                    "multipart/form-data"
                ],
                "produces": [
                    "application/json"
                ],
                "summary": "start an upload",
                "parameters": [
                    {
                        "description": "an object to start a file upload",
                        "name": "InitFileUploadReq",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "type": "object",
                            "$ref": "#/definitions/routes.InitFileUploadReq"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "object",
                            "$ref": "#/definitions/routes.InitFileUploadRes"
                        }
                    },
                    "400": {
                        "description": "bad request, unable to parse request body: (with the error)",
                        "schema": {
                            "type": "string"
                        }
                    },
                    "403": {
                        "description": "signature did not match",
                        "schema": {
                            "type": "string"
                        }
                    },
                    "500": {
                        "description": "some information about the internal error",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            }
        },
        "/api/v1/metadata": {
            "get": {
                "description": "requestBody should be a stringified version of (values are just examples):\n{\n\"metadataKey\": \"a 64-char hex string created deterministically from your account handle or private key\",\n\"timestamp\": 1557346389\n}",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "summary": "Retrieve account metadata",
                "parameters": [
                    {
                        "description": "get metadata object",
                        "name": "getMetadataReq",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "type": "object",
                            "$ref": "#/definitions/routes.getMetadataReq"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "object",
                            "$ref": "#/definitions/routes.getMetadataRes"
                        }
                    },
                    "404": {
                        "description": "no value found for that key",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            },
            "post": {
                "description": "requestBody should be a stringified version of (values are just examples):\n{\n\"metadataKey\": \"a 64-char hex string created deterministically from your account handle or private key\",\n\"metadata\": \"your (updated) account metadata\",\n\"timestamp\": 1557346389\n}",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "summary": "Update metadata",
                "parameters": [
                    {
                        "description": "update metadata object",
                        "name": "updateMetadataReq",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "type": "object",
                            "$ref": "#/definitions/routes.updateMetadataReq"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "object",
                            "$ref": "#/definitions/routes.updateMetadataRes"
                        }
                    },
                    "400": {
                        "description": "bad request, unable to parse request body: (with the error)",
                        "schema": {
                            "type": "string"
                        }
                    },
                    "403": {
                        "description": "subscription expired",
                        "schema": {
                            "type": "string"
                        }
                    },
                    "404": {
                        "description": "no value found for that key",
                        "schema": {
                            "type": "string"
                        }
                    },
                    "500": {
                        "description": "some information about the internal error",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            }
        },
        "/api/v1/upload": {
            "post": {
                "description": "upload a chunk of a file. The first partIndex must be 1. The endIndex must be greater than or equal to partIndex.\nrequestBody should be a stringified version of (values are just examples):\n{\n\"fileHandle\": \"a deterministically created file handle\",\n\"partIndex\": 1,\n\"endIndex\": 2\n}",
                "consumes": [
                    "multipart/form-data"
                ],
                "produces": [
                    "application/json"
                ],
                "summary": "upload a chunk of a file",
                "parameters": [
                    {
                        "description": "an object to upload a chunk of a file",
                        "name": "UploadFileReq",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "type": "object",
                            "$ref": "#/definitions/routes.UploadFileReq"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "object",
                            "$ref": "#/definitions/routes.uploadFileRes"
                        }
                    },
                    "400": {
                        "description": "bad request, unable to parse request body: (with the error)",
                        "schema": {
                            "type": "string"
                        }
                    },
                    "403": {
                        "description": "Forbidden",
                        "schema": {
                            "type": "object",
                            "$ref": "#/definitions/routes.accountCreateRes"
                        }
                    },
                    "500": {
                        "description": "some information about the internal error",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "models.Invoice": {
            "type": "object",
            "required": [
                "cost",
                "ethAddress"
            ],
            "properties": {
                "cost": {
                    "type": "number",
                    "example": 1.56
                },
                "ethAddress": {
                    "type": "string",
                    "maxLength": 42,
                    "minLength": 42,
                    "example": "a 42-char eth address with 0x prefix"
                }
            }
        },
        "routes.InitFileUploadReq": {
            "type": "object",
            "required": [
                "metadata",
                "publicKey",
                "requestBody",
                "signature"
            ],
            "properties": {
                "metadata": {
                    "type": "string",
                    "example": "the metadata of the file you are about to upload, as an array of bytes"
                },
                "publicKey": {
                    "type": "string",
                    "maxLength": 66,
                    "minLength": 66,
                    "example": "a 66-character public key"
                },
                "requestBody": {
                    "type": "string",
                    "example": "should produce routes.InitFileUploadObj, see description for example"
                },
                "signature": {
                    "description": "signature without 0x prefix is broken into\nR: sig[0:63]\nS: sig[64:127]",
                    "type": "string",
                    "maxLength": 128,
                    "minLength": 128,
                    "example": "a 128 character string created when you signed the request with your private key or account handle"
                }
            }
        },
        "routes.InitFileUploadRes": {
            "type": "object",
            "properties": {
                "status": {
                    "type": "string",
                    "example": "Success"
                }
            }
        },
        "routes.UploadFileReq": {
            "type": "object",
            "required": [
                "chunkData",
                "publicKey",
                "requestBody",
                "signature"
            ],
            "properties": {
                "chunkData": {
                    "type": "string",
                    "example": "a binary string of the chunk data"
                },
                "publicKey": {
                    "type": "string",
                    "maxLength": 66,
                    "minLength": 66,
                    "example": "a 66-character public key"
                },
                "requestBody": {
                    "type": "string",
                    "example": "should produce routes.UploadFileObj, see description for example"
                },
                "signature": {
                    "description": "signature without 0x prefix is broken into\nR: sig[0:63]\nS: sig[64:127]",
                    "type": "string",
                    "maxLength": 128,
                    "minLength": 128,
                    "example": "a 128 character string created when you signed the request with your private key or account handle"
                }
            }
        },
        "routes.accountCreateReq": {
            "type": "object",
            "required": [
                "publicKey",
                "requestBody",
                "signature"
            ],
            "properties": {
                "publicKey": {
                    "type": "string",
                    "maxLength": 66,
                    "minLength": 66,
                    "example": "a 66-character public key"
                },
                "requestBody": {
                    "type": "string",
                    "example": "should produce routes.accountCreateObj, see description for example"
                },
                "signature": {
                    "description": "signature without 0x prefix is broken into\nR: sig[0:63]\nS: sig[64:127]",
                    "type": "string",
                    "maxLength": 128,
                    "minLength": 128,
                    "example": "a 128 character string created when you signed the request with your private key or account handle"
                }
            }
        },
        "routes.accountCreateRes": {
            "type": "object",
            "required": [
                "expirationDate"
            ],
            "properties": {
                "expirationDate": {
                    "type": "string"
                },
                "invoice": {
                    "type": "object",
                    "$ref": "#/definitions/models.Invoice"
                }
            }
        },
        "routes.accountGetObj": {
            "type": "object",
            "required": [
                "cost",
                "ethAddress",
                "expirationDate",
                "monthsInSubscription",
                "storageLimit"
            ],
            "properties": {
                "cost": {
                    "type": "number",
                    "example": 2
                },
                "createdAt": {
                    "type": "string"
                },
                "ethAddress": {
                    "description": "the eth address they will send payment to",
                    "type": "string",
                    "maxLength": 42,
                    "minLength": 42,
                    "example": "a 42-char eth address with 0x prefix"
                },
                "expirationDate": {
                    "type": "string"
                },
                "monthsInSubscription": {
                    "description": "number of months in their subscription",
                    "type": "integer",
                    "example": 12
                },
                "storageLimit": {
                    "description": "how much storage they are allowed, in GB",
                    "type": "integer",
                    "example": 100
                },
                "storageUsed": {
                    "description": "how much storage they have used, in GB",
                    "type": "number",
                    "example": 30
                },
                "updatedAt": {
                    "type": "string"
                }
            }
        },
        "routes.accountPaidRes": {
            "type": "object",
            "properties": {
                "account": {
                    "type": "object",
                    "$ref": "#/definitions/routes.accountGetObj"
                },
                "paymentStatus": {
                    "type": "string",
                    "example": "paid"
                }
            }
        },
        "routes.deleteFileReq": {
            "type": "object",
            "required": [
                "publicKey",
                "requestBody",
                "signature"
            ],
            "properties": {
                "publicKey": {
                    "type": "string",
                    "maxLength": 66,
                    "minLength": 66,
                    "example": "a 66-character public key"
                },
                "requestBody": {
                    "type": "string",
                    "example": "should produce routes.deleteFileObj, see description for example"
                },
                "signature": {
                    "description": "signature without 0x prefix is broken into\nR: sig[0:63]\nS: sig[64:127]",
                    "type": "string",
                    "maxLength": 128,
                    "minLength": 128,
                    "example": "a 128 character string created when you signed the request with your private key or account handle"
                }
            }
        },
        "routes.deleteFileRes": {
            "type": "object"
        },
        "routes.downloadFileReq": {
            "type": "object",
            "required": [
                "publicKey",
                "requestBody",
                "signature"
            ],
            "properties": {
                "publicKey": {
                    "type": "string",
                    "maxLength": 66,
                    "minLength": 66,
                    "example": "a 66-character public key"
                },
                "requestBody": {
                    "type": "string",
                    "example": "should produce routes.downloadFileObj, see description for example"
                },
                "signature": {
                    "description": "signature without 0x prefix is broken into\nR: sig[0:63]\nS: sig[64:127]",
                    "type": "string",
                    "maxLength": 128,
                    "minLength": 128,
                    "example": "a 128 character string created when you signed the request with your private key or account handle"
                }
            }
        },
        "routes.downloadFileRes": {
            "type": "object",
            "properties": {
                "fileDownloadUrl": {
                    "description": "Url should point to S3, thus client does not need to download it from this node.",
                    "type": "string",
                    "example": "a URL to use to download the file"
                }
            }
        },
        "routes.getAccountDataReq": {
            "type": "object",
            "required": [
                "publicKey",
                "requestBody",
                "signature"
            ],
            "properties": {
                "publicKey": {
                    "type": "string",
                    "maxLength": 66,
                    "minLength": 66,
                    "example": "a 66-character public key"
                },
                "requestBody": {
                    "type": "string",
                    "example": "should produce routes.accountGetReqObj, see description for example"
                },
                "signature": {
                    "description": "signature without 0x prefix is broken into\nR: sig[0:63]\nS: sig[64:127]",
                    "type": "string",
                    "maxLength": 128,
                    "minLength": 128,
                    "example": "a 128 character string created when you signed the request with your private key or account handle"
                }
            }
        },
        "routes.getMetadataReq": {
            "type": "object",
            "required": [
                "publicKey",
                "requestBody",
                "signature"
            ],
            "properties": {
                "publicKey": {
                    "type": "string",
                    "maxLength": 66,
                    "minLength": 66,
                    "example": "a 66-character public key"
                },
                "requestBody": {
                    "type": "string",
                    "example": "should produce routes.getMetadataObject, see description for example"
                },
                "signature": {
                    "description": "signature without 0x prefix is broken into\nR: sig[0:63]\nS: sig[64:127]",
                    "type": "string",
                    "maxLength": 128,
                    "minLength": 128,
                    "example": "a 128 character string created when you signed the request with your private key or account handle"
                }
            }
        },
        "routes.getMetadataRes": {
            "type": "object",
            "required": [
                "expirationDate"
            ],
            "properties": {
                "expirationDate": {
                    "type": "string"
                },
                "metadata": {
                    "type": "string",
                    "example": "your account metadata"
                }
            }
        },
        "routes.updateMetadataReq": {
            "type": "object",
            "required": [
                "publicKey",
                "requestBody",
                "signature"
            ],
            "properties": {
                "publicKey": {
                    "type": "string",
                    "maxLength": 66,
                    "minLength": 66,
                    "example": "a 66-character public key"
                },
                "requestBody": {
                    "type": "string",
                    "example": "should produce routes.updateMetadataObject, see description for example"
                },
                "signature": {
                    "description": "signature without 0x prefix is broken into\nR: sig[0:63]\nS: sig[64:127]",
                    "type": "string",
                    "maxLength": 128,
                    "minLength": 128,
                    "example": "a 128 character string created when you signed the request with your private key or account handle"
                }
            }
        },
        "routes.updateMetadataRes": {
            "type": "object",
            "required": [
                "expirationDate",
                "metadata",
                "metadataKey"
            ],
            "properties": {
                "expirationDate": {
                    "type": "string"
                },
                "metadata": {
                    "type": "string",
                    "example": "your (updated) account metadata"
                },
                "metadataKey": {
                    "type": "string",
                    "example": "a 64-char hex string created deterministically from your account handle or private key"
                }
            }
        },
        "routes.uploadFileRes": {
            "type": "object",
            "properties": {
                "status": {
                    "type": "string",
                    "example": "Chunk is uploaded"
                }
            }
        },
        "routes.userStatsRes": {
            "type": "object",
            "properties": {
                "uploadedFileSizeInMb": {
                    "type": "number"
                },
                "uploadedFilesCount": {
                    "type": "integer"
                },
                "userAccountsCount": {
                    "type": "integer"
                }
            }
        }
    }
}`

type swaggerInfo struct {
	Version     string
	Host        string
	BasePath    string
	Title       string
	Description string
}

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo swaggerInfo

type s struct{}

func (s *s) ReadDoc() string {
	t, err := template.New("swagger_info").Parse(doc)
	if err != nil {
		return doc
	}

	var tpl bytes.Buffer
	if err := t.Execute(&tpl, SwaggerInfo); err != nil {
		return doc
	}

	return tpl.String()
}

func init() {
	swag.Register(swag.Name, &s{})
}
