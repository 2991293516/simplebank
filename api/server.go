package api

import (
	"fmt"
	"log"
	db "simplebank/db/sqlc"
	"simplebank/token"
	"simplebank/util"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

type Server struct {
	config     util.Config
	tokenMaker token.Maker
	store      db.Store
	router     *gin.Engine
}

func NewServer(config util.Config, store db.Store) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("could not create token maker: %w", err)
	}

	server := &Server{
		config:     config,
		tokenMaker: tokenMaker,
		store:      store,
	}

	v, ok := binding.Validator.Engine().(*validator.Validate)
	if ok {
		v.RegisterValidation("currency", validCurrency)
	} else {
		log.Fatal("validator not found")
	}

	server.setUpRouter()

	return server, nil
}

func (server *Server) setUpRouter() {
	router := gin.Default()

	router.POST("/accounts", server.createAccount)
	router.GET("/accounts/:id", server.getAccount)
	router.GET("/accounts/list", server.listAccount)

	router.POST("/transfer", server.createTransfer)

	router.POST("/users", server.createUser)
	router.GET("/users/login", server.loginUser)

	server.router = router
}

func (server *Server) Start(address string) error {
	return server.router.Run(address)
}

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}
