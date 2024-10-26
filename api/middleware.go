package api

import (
	"errors"
	"net/http"
	"simplebank/token"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	authorizationHeaderKey  = "authorization"
	authorizationTypeBearer = "bearer"
	authorizationPayloadKey = "authorization_payload"
)

var (
	ErrNotAuthorizationHeader = errors.New("authorization header is not provided")
	ErrWrongFormat            = errors.New("invalid authorization header format")
	ErrWrongAuthorizationType = errors.New("invalid authorization type")
)

func authMiddleware(tokenMaker token.Maker) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authorizationHeader := ctx.GetHeader(authorizationHeaderKey)

		if len(authorizationHeader) == 0 {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(ErrNotAuthorizationHeader))
			return
		}

		// authorizationHeader的格式(Bearer token)
		// 解析token
		field := strings.Fields(authorizationHeader)
		if len(field) < 2 {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(ErrWrongFormat))
			return
		}

		authorizationType := strings.ToLower(field[0])
		if authorizationType != authorizationTypeBearer {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(ErrWrongAuthorizationType))
			return
		}

		accessToken := field[1]
		payload, err := tokenMaker.VerifyToken(accessToken)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		ctx.Set(authorizationPayloadKey, payload)
		ctx.Next()
	}
}
