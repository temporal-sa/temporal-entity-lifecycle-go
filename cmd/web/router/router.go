package router

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/temporal-sa/temporal-entity-lifecycle-go/cmd/web/handler"
	"go.temporal.io/sdk/client"
	"os"
)

type Router struct {
	*gin.Engine
	c client.Client
}

func New(c client.Client) (*Router, error) {
	r := &Router{
		Engine: gin.Default(),
		c:      c,
	}

	if r.c == nil {
		return nil, errors.New("temporal client required & missing")
	}
	rh, err := handler.New(r.c, os.Getenv("TEMPORAL_CLIENT_NAMESPACE"))
	if err != nil {
		return nil, err
	}
	r.LoadHTMLGlob("templates/*.html")
	r.GET("/approve_permission", rh.GETApprovePermission)
	r.GET("/create_user", rh.GETCreateUser)
	r.GET("/user", rh.GETUser)
	r.GET("/users", rh.GETUsers)
	r.GET("/request_permission", rh.GETRequestPermission)
	r.POST("/approve_permission", rh.POSTApprovePermission)
	r.POST("/create_user", rh.POSTCreateUser)
	r.POST("/delete_user", rh.POSTDeleteUser)
	r.POST("/undo_delete_user", rh.POSTUndoDeleteUser)
	r.POST("/request_permission", rh.POSTRequestPermission)
	return r, nil
}
